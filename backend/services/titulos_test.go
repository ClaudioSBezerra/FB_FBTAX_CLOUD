package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestValidadeTokenAposPagamento(t *testing.T) {
	hoje := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	venc := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	if got := ValidadeTokenAposPagamento(venc, hoje); !got.Equal(venc.AddDate(0, 0, 45)) {
		t.Errorf("vencimento futuro: %v", got)
	}
	atrasado := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) // venc+45 = 15/09, já passou
	if got := ValidadeTokenAposPagamento(atrasado, hoje); !got.Equal(hoje.AddDate(0, 0, 15)) {
		t.Errorf("título muito atrasado deve liberar por 15 dias: %v", got)
	}
}

type cenarioReceber struct {
	db                                       *sql.DB
	cliente, contrato, plano, produto, conta string
}

func novoCenarioReceber(t *testing.T) *cenarioReceber {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	c := &cenarioReceber{db: db}
	suf := fmt.Sprintf("%d", time.Now().UnixNano())
	must := func(q string, args ...any) string {
		var id string
		if err := db.QueryRow(q, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return id
	}
	c.cliente = must(`INSERT INTO financeiro.clientes (razao_social, cnpj) VALUES ('JC TESTE', $1) RETURNING id`, suf[len(suf)-14:])
	// o cliente cadastrou a matriz; o pagamento virá de uma filial (mesma raiz)
	db.Exec(`UPDATE financeiro.clientes SET cnpj = '06314327000121' WHERE id = $1`, c.cliente)
	c.contrato = must(`INSERT INTO financeiro.contratos (cliente_id, data_inicio, valor_total) VALUES ($1, CURRENT_DATE, 4500) RETURNING id`, c.cliente)
	c.produto = must(`INSERT INTO financeiro.produtos (codigo, nome) VALUES ($1, 'Produto teste') RETURNING id`, "T"+suf)
	c.plano = must(`INSERT INTO financeiro.planos (produto_id, nome) VALUES ($1, 'Plano teste') RETURNING id`, c.produto)
	c.conta = must(`INSERT INTO financeiro.contas_financeiras (apelido, banco) VALUES ('Conta receber teste','077') RETURNING id`)
	if _, err := db.Exec(`INSERT INTO financeiro.tokens (contrato_id, plano_id, token, valid_from, valid_until, status) VALUES ($1,$2,$3, CURRENT_DATE - 60, CURRENT_DATE - 15, 'em_carencia')`, c.contrato, c.plano, "tok"+suf); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM financeiro.titulos_receber WHERE cliente_id = $1`, c.cliente)
		db.Exec(`DELETE FROM financeiro.tokens WHERE contrato_id = $1`, c.contrato)
		db.Exec(`DELETE FROM financeiro.contratos WHERE id = $1`, c.contrato)
		db.Exec(`DELETE FROM financeiro.clientes WHERE id = $1`, c.cliente)
		db.Exec(`DELETE FROM financeiro.planos WHERE id = $1`, c.plano)
		db.Exec(`DELETE FROM financeiro.produtos WHERE id = $1`, c.produto)
		db.Exec(`DELETE FROM financeiro.transacoes WHERE conta_id = $1`, c.conta)
		db.Exec(`DELETE FROM financeiro.contas_financeiras WHERE id = $1`, c.conta)
	})
	return c
}

func (c *cenarioReceber) titulo(t *testing.T, tipo string, valor float64, vencDias int, criadoDiasAtras int) string {
	t.Helper()
	var id string
	comp := any(nil)
	if tipo == "mensalidade" {
		comp = time.Now().AddDate(0, 0, vencDias).Format("2006-01") + "-01"
	}
	if err := c.db.QueryRow(`
		INSERT INTO financeiro.titulos_receber (cliente_id, contrato_id, tipo, competencia, descricao, valor, vencimento, created_at)
		VALUES ($1,$2,$3,$4,'teste',$5, CURRENT_DATE + $6::int, NOW() - ($7::int || ' days')::interval) RETURNING id`,
		c.cliente, c.contrato, tipo, comp, valor, vencDias, criadoDiasAtras).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (c *cenarioReceber) credito(t *testing.T, valor float64, diasAtras int, cnpjPagador string) string {
	t.Helper()
	det := "NULL"
	if cnpjPagador != "" {
		det = fmt.Sprintf(`'{"detalhes":{"cpfCnpjPagador":"%s","nomePagador":"PAGADOR"}}'`, cnpjPagador)
	}
	var id string
	q := fmt.Sprintf(`INSERT INTO financeiro.transacoes (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext, detalhes)
		VALUES ($1, CURRENT_DATE - $2::int, 'PIX', $3, 'credito', 'PIX', $4, %s) RETURNING id`, det)
	if err := c.db.QueryRow(q, c.conta, diasAtras, valor, fmt.Sprintf("rc-%d", time.Now().UnixNano())).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (c *cenarioReceber) status(t *testing.T, id string) (status, origem string) {
	t.Helper()
	var o sql.NullString
	c.db.QueryRow(`SELECT status, baixa_origem FROM financeiro.titulos_receber WHERE id = $1`, id).Scan(&status, &o)
	return status, o.String
}

func TestConciliarTitulos_PorCNPJEValor_RenovaToken(t *testing.T) {
	c := novoCenarioReceber(t)
	ctx := context.Background()
	mens := c.titulo(t, "mensalidade", 4500, 10, 1)
	credito := c.credito(t, 4500, 1, "06314327000203") // outra filial, mesma raiz 06314327
	n, err := ConciliarTitulos(ctx, c.db)
	if err != nil || n < 1 {
		t.Fatalf("conciliar: %d %v", n, err)
	}
	if st, o := c.status(t, mens); st != "pago" || o != "automatica" {
		t.Fatalf("título deveria estar pago automaticamente: %s %s", st, o)
	}
	var tid string
	c.db.QueryRow(`SELECT transacao_id FROM financeiro.titulos_receber WHERE id = $1`, mens).Scan(&tid)
	if tid != credito {
		t.Errorf("deveria estar ligado ao lançamento %s, está em %s", credito, tid)
	}
	// Token renovado: antigo encerrado e novo ativo até vencimento + 45 dias.
	var ativos int
	var ate time.Time
	c.db.QueryRow(`SELECT COUNT(*), MAX(valid_until) FROM financeiro.tokens WHERE contrato_id = $1 AND status = 'ativo'`, c.contrato).Scan(&ativos, &ate)
	esperado := time.Now().AddDate(0, 0, 10+45)
	if ativos != 1 || ate.Format("2006-01-02") != esperado.Format("2006-01-02") {
		t.Errorf("token renovado: ativos=%d até %v, esperado %v", ativos, ate, esperado.Format("2006-01-02"))
	}
	var pred sql.NullString
	c.db.QueryRow(`SELECT predecessor_id FROM financeiro.tokens WHERE contrato_id = $1 AND status = 'ativo'`, c.contrato).Scan(&pred)
	if !pred.Valid {
		t.Error("o token novo deve apontar para o predecessor")
	}
	// Reexecutar não baixa nada de novo.
	if n2, _ := ConciliarTitulos(ctx, c.db); n2 != 0 {
		t.Errorf("segunda execução baixou %d", n2)
	}
}

func TestConciliarTitulos_NaoBaixaSemCertezaOuFora(t *testing.T) {
	c := novoCenarioReceber(t)
	ctx := context.Background()

	// 1) CNPJ de outra empresa: não casa.
	t1 := c.titulo(t, "adesao", 1234.56, 5, 1)
	c.credito(t, 1234.56, 1, "11222333000181")
	// 2) sem detalhe do pagador: não casa automaticamente.
	t2 := c.titulo(t, "projeto", 777, 5, 1)
	c.credito(t, 777, 1, "")
	// 3) pagamento anterior à criação do título (mês passado, mesmo valor): não casa.
	t3 := c.titulo(t, "projeto", 888, 5, 1)
	c.credito(t, 888, 30, "06314327000203")
	// 4) dois pagamentos candidatos para um título: ambíguo.
	t4 := c.titulo(t, "projeto", 999, 5, 1)
	c.credito(t, 999, 1, "06314327000203")
	c.credito(t, 999, 0, "06314327000203")

	if _, err := ConciliarTitulos(ctx, c.db); err != nil {
		t.Fatal(err)
	}
	for nome, id := range map[string]string{"outro CNPJ": t1, "sem detalhe": t2, "anterior à criação": t3, "ambíguo": t4} {
		if st, _ := c.status(t, id); st != "aberto" {
			t.Errorf("%s: o título não deveria ser baixado automaticamente (status %s)", nome, st)
		}
	}
}

func TestConciliarTitulos_DoisTitulosDoisPagamentos_PareiaPelaOrdem(t *testing.T) {
	c := novoCenarioReceber(t)
	antigo := c.titulo(t, "projeto", 500, -20, 25)
	novo := c.titulo(t, "projeto", 500, 10, 25)
	p1 := c.credito(t, 500, 12, "06314327000203")
	p2 := c.credito(t, 500, 2, "06314327000203")
	if n, err := ConciliarTitulos(context.Background(), c.db); err != nil || n != 2 {
		t.Fatalf("esperava 2 baixas: %d %v", n, err)
	}
	var t1, t2 string
	c.db.QueryRow(`SELECT transacao_id FROM financeiro.titulos_receber WHERE id = $1`, antigo).Scan(&t1)
	c.db.QueryRow(`SELECT transacao_id FROM financeiro.titulos_receber WHERE id = $1`, novo).Scan(&t2)
	if t1 != p1 || t2 != p2 {
		t.Errorf("o título mais antigo deveria casar com o pagamento mais antigo: %s/%s vs %s/%s", t1, t2, p1, p2)
	}
}

func TestBaixarTitulo_AdesaoNaoRenovaToken_ELancamentoNaoBaixaDoisTitulos(t *testing.T) {
	c := novoCenarioReceber(t)
	adesao := c.titulo(t, "adesao", 3000, 5, 0)
	tx, _ := c.db.Begin()
	renovados, err := BaixarTitulo(tx, adesao, nil, time.Now(), 3000, "manual", "teste")
	if err != nil || renovados != 0 {
		t.Fatalf("adesão não renova token: %d %v", renovados, err)
	}
	tx.Commit()
	var ativos int
	c.db.QueryRow(`SELECT COUNT(*) FROM financeiro.tokens WHERE contrato_id = $1 AND status = 'ativo'`, c.contrato).Scan(&ativos)
	if ativos != 0 {
		t.Errorf("adesão não deveria ativar token (ativos=%d)", ativos)
	}
	// Título já pago não baixa de novo.
	tx2, _ := c.db.Begin()
	if _, err := BaixarTitulo(tx2, adesao, nil, time.Now(), 3000, "manual", ""); err == nil {
		t.Error("título pago não pode ser baixado outra vez")
	}
	tx2.Rollback()
	// Um lançamento não baixa dois títulos.
	a, b := c.titulo(t, "projeto", 100, 5, 0), c.titulo(t, "projeto", 100, 6, 0)
	cr := c.credito(t, 100, 0, "06314327000203")
	for i, id := range []string{a, b} {
		tx, _ := c.db.Begin()
		_, err := BaixarTitulo(tx, id, &cr, time.Now(), 100, "manual", "")
		if i == 0 && err != nil {
			t.Fatal(err)
		}
		if i == 1 && err == nil {
			t.Error("o mesmo lançamento não pode baixar dois títulos")
		}
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}
}
