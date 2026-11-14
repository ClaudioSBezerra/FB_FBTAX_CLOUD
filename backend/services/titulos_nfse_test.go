package services

import (
	"context"
	"testing"
	"time"
)

func TestVencimentoPadrao(t *testing.T) {
	brt := time.FixedZone("BRT", -3*3600)
	casos := []struct{ emissao, esperado string }{
		{"2026-12-01", "2026-12-10"}, // dia 01: vence no dia 10 do mesmo mês
		{"2026-12-10", "2026-12-10"}, // no próprio dia 10 ainda vale o mês
		{"2026-12-11", "2027-01-10"}, // depois do dia 10, mês seguinte (virada de ano)
		{"2026-11-30", "2026-12-10"},
	}
	for _, c := range casos {
		e, _ := time.ParseInLocation("2006-01-02", c.emissao, brt)
		if got := VencimentoPadrao(e).Format("2006-01-02"); got != c.esperado {
			t.Errorf("emissão %s: vencimento %s, esperado %s", c.emissao, got, c.esperado)
		}
	}
}

func (c *cenarioReceber) nfse(t *testing.T, status string, comContrato bool, comp string) string {
	t.Helper()
	var emp string
	if err := c.db.QueryRow(`INSERT INTO financeiro.empresas (razao_social, cnpj, logradouro, numero, bairro, cep, municipio, uf)
		VALUES ('EMP TITULO NFSE','38149716000128','R','1','B','60000000','Fortaleza','CE') RETURNING id`).Scan(&emp); err != nil {
		t.Fatal(err)
	}
	var contrato any
	if comContrato {
		contrato = c.contrato
	}
	var id string
	chave := any(nil)
	if status == "autorizada" || status == "cancelada" {
		chave = "CH" + time.Now().Format("150405.000000000")
	}
	if err := c.db.QueryRow(`INSERT INTO financeiro.nfse_emitidas (empresa_id, cliente_id, contrato_id, ambiente, serie, n_dps, id_dps, competencia, valor, descricao, status, chave_acesso)
		VALUES ($1,$2,$3,1,'00001',$4,$5,$6,4500,'Mensalidade  Farol',$7,$8) RETURNING id`,
		emp, c.cliente, contrato, time.Now().UnixNano()%1000000000, "DPS"+time.Now().Format("150405.000000000"), comp, status, chave).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c.db.Exec(`DELETE FROM financeiro.titulos_receber WHERE nfse_id = $1`, id)
		c.db.Exec(`DELETE FROM financeiro.nfse_emitidas WHERE id = $1`, id)
		c.db.Exec(`DELETE FROM financeiro.empresas WHERE id = $1`, emp)
	})
	return id
}

func TestGerarTituloDaNFSe(t *testing.T) {
	c := novoCenarioReceber(t)
	ctx := context.Background()
	venc := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)

	// só nota autorizada gera título
	if _, _, err := GerarTituloDaNFSe(ctx, c.db, c.nfse(t, "rejeitada", true, "2026-12-01"), "mensalidade", venc); err == nil {
		t.Fatal("nota rejeitada não pode gerar título")
	}
	if _, _, err := GerarTituloDaNFSe(ctx, c.db, c.nfse(t, "autorizada", true, "2027-01-01"), "invalido", venc); err == nil {
		t.Fatal("tipo inválido deve falhar")
	}

	// cria e é idempotente
	nf := c.nfse(t, "autorizada", true, "2026-12-01")
	id1, criado, err := GerarTituloDaNFSe(ctx, c.db, nf, "mensalidade", venc)
	if err != nil || !criado {
		t.Fatalf("criar: %v criado=%v", err, criado)
	}
	id2, criado2, err := GerarTituloDaNFSe(ctx, c.db, nf, "mensalidade", venc)
	if err != nil || criado2 || id2 != id1 {
		t.Fatalf("repetir deve devolver o mesmo título: %v %v %v", id2, criado2, err)
	}
	var valor, desc, vencStr, tipo string
	c.db.QueryRow(`SELECT valor::text, descricao, vencimento::text, tipo FROM financeiro.titulos_receber WHERE id=$1`, id1).Scan(&valor, &desc, &vencStr, &tipo)
	if valor != "4500.00" || desc != "Mensalidade Farol" || vencStr != "2026-12-10" || tipo != "mensalidade" {
		t.Errorf("título: %s %q %s %s", valor, desc, vencStr, tipo)
	}

	// cancelar a nota cancela o título em aberto
	n, pago, err := CancelarTituloDaNFSe(ctx, c.db, nf)
	if err != nil || n != 1 || pago {
		t.Fatalf("cancelar: %d %v %v", n, pago, err)
	}
	// depois de cancelada a nota, uma nova nota da mesma competência pode gerar outro título
	c.db.Exec(`UPDATE financeiro.nfse_emitidas SET status='cancelada' WHERE id=$1`, nf)
	nf2 := c.nfse(t, "autorizada", true, "2026-12-01")
	if _, criado, err := GerarTituloDaNFSe(ctx, c.db, nf2, "mensalidade", venc); err != nil || !criado {
		t.Fatalf("nova nota após cancelamento: %v %v", err, criado)
	}
}

func TestGerarTituloDaNFSe_LigaTituloJaCadastradoEPago(t *testing.T) {
	c := novoCenarioReceber(t)
	ctx := context.Background()
	venc := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)

	// título da mensalidade cadastrado à mão antes da nota: a nota o adota, sem duplicar
	var manual string
	if err := c.db.QueryRow(`INSERT INTO financeiro.titulos_receber (cliente_id, contrato_id, tipo, competencia, descricao, valor, vencimento)
		VALUES ($1,$2,'mensalidade','2026-12-01','manual',4500,'2026-12-10') RETURNING id`, c.cliente, c.contrato).Scan(&manual); err != nil {
		t.Fatal(err)
	}
	nf := c.nfse(t, "autorizada", true, "2026-12-01")
	id, criado, err := GerarTituloDaNFSe(ctx, c.db, nf, "mensalidade", venc)
	if err != nil || criado || id != manual {
		t.Fatalf("deveria ligar o título existente: id=%s criado=%v err=%v", id, criado, err)
	}

	// nota sem contrato gera título avulso
	avulsa := c.nfse(t, "autorizada", false, "2026-12-01")
	if _, criado, err := GerarTituloDaNFSe(ctx, c.db, avulsa, "outro", venc); err != nil || !criado {
		t.Fatalf("título avulso: %v %v", err, criado)
	}

	// título já pago: cancelar a nota avisa e não mexe
	c.db.Exec(`UPDATE financeiro.titulos_receber SET status='pago', pago_em=CURRENT_DATE, valor_pago=4500 WHERE id=$1`, manual)
	n, pago, err := CancelarTituloDaNFSe(ctx, c.db, nf)
	if err != nil || n != 0 || !pago {
		t.Fatalf("título pago: n=%d pago=%v err=%v", n, pago, err)
	}
}
