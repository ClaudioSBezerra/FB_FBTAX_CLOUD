package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestNivelEDiasCredencial(t *testing.T) {
	for dias, want := range map[int]string{-1: "vencida", 0: "critico", 7: "critico", 8: "atencao", 30: "atencao", 31: "ok", 364: "ok"} {
		if got := nivelCredencial(dias); got != want {
			t.Errorf("%d dias: %s, esperado %s", dias, got, want)
		}
	}
	agora := time.Date(2026, 11, 14, 23, 59, 0, 0, brasilia)
	if d := diasAte(time.Date(2026, 12, 14, 0, 0, 0, 0, time.UTC), agora); d != 30 {
		t.Errorf("14/12 a partir de 14/11 = %d dias, esperado 30", d)
	}
	if d := diasAte(time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC), agora); d != 0 {
		t.Errorf("vence hoje = %d", d)
	}
}

func TestAlertasCredenciais_ListaNivelEEnviaUmaVezPorMarco(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definido")
	}
	t.Setenv("FINANCEIRO_SECRET", "segredo-de-teste")
	t.Setenv("FINANCEIRO_ADMIN_EMAIL", "gestor@exemplo.com.br")
	db, err := sql.Open("postgres", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	db.Exec(`DELETE FROM financeiro.credencial_alertas`)
	db.Exec(`DELETE FROM financeiro.nfse_certificado`)
	db.Exec(`DELETE FROM financeiro.empresas`)

	// A1 da NFS-e vence em 20 dias e o certificado do Inter em 5 dias.
	var empID string
	if err := db.QueryRow(`INSERT INTO financeiro.empresas (razao_social, cnpj, logradouro, numero, bairro, cep, municipio, uf, email)
		VALUES ('EMPRESA ALERTA','38149716000128','R','1','B','74000000','X','GO','empresa@exemplo.com.br') RETURNING id`).Scan(&empID); err != nil {
		t.Fatal(err)
	}
	db.Exec(`INSERT INTO financeiro.nfse_certificado (empresa_id, certificado_pfx, certificado_senha, validade) VALUES ($1,'x','y', CURRENT_DATE + 20)`, empID)
	certPEM, keyPEM := parCertInter(t, time.Now().In(brasilia).AddDate(0, 0, 5))
	cfgJSON, _ := json.Marshal(ProviderConfig{ClientID: "a", ClientSecret: "b", CertPEM: certPEM, KeyPEM: keyPEM})
	enc, _ := EncryptConfig(string(cfgJSON))
	var contaID string
	if err := db.QueryRow(`INSERT INTO financeiro.contas_financeiras (apelido, banco, provedor, provedor_config) VALUES ('INTER ALERTA','077','inter',$1) RETURNING id`, enc).Scan(&contaID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM financeiro.credencial_alertas`)
		db.Exec(`DELETE FROM financeiro.nfse_certificado WHERE empresa_id = $1`, empID)
		db.Exec(`DELETE FROM financeiro.contas_financeiras WHERE id = $1`, contaID)
		db.Exec(`DELETE FROM financeiro.empresas WHERE id = $1`, empID)
	})

	agora := time.Now()
	creds, err := ListarVencimentosCredenciais(ctx, db, agora)
	if err != nil || len(creds) != 2 {
		t.Fatalf("listar: %d %v", len(creds), err)
	}
	if creds[0].Tipo != "inter" || creds[0].Nivel != "critico" || creds[0].DiasRestantes != 5 {
		t.Errorf("a mais urgente deveria ser o Inter (5 dias, crítico): %+v", creds[0])
	}
	if creds[1].Tipo != "nfse" || creds[1].Nivel != "atencao" || creds[1].DiasRestantes != 20 {
		t.Errorf("A1 da NFS-e (20 dias, atenção): %+v", creds[1])
	}

	var enviados []string
	var dest [][]string
	ok := func(to []string, assunto, corpo string) error {
		enviados = append(enviados, assunto+"\n"+corpo)
		dest = append(dest, to)
		return nil
	}
	n, err := AvaliarAlertasCredenciais(ctx, db, agora, ok)
	if err != nil || n != 2 || len(enviados) != 1 {
		t.Fatalf("primeira avaliação: n=%d err=%v e-mails=%d", n, err, len(enviados))
	}
	if !strings.Contains(enviados[0], "2 credenciais") || !strings.Contains(enviados[0], "INTER ALERTA") || !strings.Contains(enviados[0], "Renovar") || !strings.Contains(enviados[0], "NFS-e") {
		t.Errorf("conteúdo do alerta: %s", enviados[0])
	}
	if len(dest[0]) != 2 || dest[0][0] != "gestor@exemplo.com.br" || dest[0][1] != "empresa@exemplo.com.br" {
		t.Errorf("destinatários: %v", dest[0])
	}
	// Mesma situação: não repete o e-mail.
	if n, _ := AvaliarAlertasCredenciais(ctx, db, agora, ok); n != 0 || len(enviados) != 1 {
		t.Errorf("não deveria reenviar no mesmo marco: n=%d e-mails=%d", n, len(enviados))
	}
	// Dias depois, cruzando novos marcos (3 dias para o Inter): avisa de novo só o Inter.
	n, _ = AvaliarAlertasCredenciais(ctx, db, agora.AddDate(0, 0, 2), ok)
	if n != 1 || len(enviados) != 2 || !strings.Contains(enviados[1], "INTER ALERTA") || strings.Contains(enviados[1], "Certificado A1 da NFS-e") {
		t.Errorf("novo marco do Inter: n=%d e-mails=%d %s", n, len(enviados), enviados[len(enviados)-1])
	}
	// Falha de envio: nada é registrado, e a próxima tentativa envia.
	db.Exec(`DELETE FROM financeiro.credencial_alertas`)
	falha := func([]string, string, string) error { return errors.New("smtp caiu") }
	if n, err := AvaliarAlertasCredenciais(ctx, db, agora, falha); err == nil || n != 0 {
		t.Fatalf("falha de envio deveria devolver erro: n=%d err=%v", n, err)
	}
	if n, _ := AvaliarAlertasCredenciais(ctx, db, agora, ok); n != 2 {
		t.Errorf("depois da falha deveria reenviar: n=%d", n)
	}
	// Renovação (nova validade) recomeça os avisos.
	db.Exec(`UPDATE financeiro.nfse_certificado SET validade = CURRENT_DATE + 25 WHERE empresa_id = $1`, empID)
	if n, _ := AvaliarAlertasCredenciais(ctx, db, agora, ok); n != 1 {
		t.Errorf("nova validade dentro de 30 dias recomeça o aviso: n=%d", n)
	}
	// Fora da janela de 30 dias: sem alerta.
	db.Exec(`DELETE FROM financeiro.credencial_alertas`)
	db.Exec(`UPDATE financeiro.nfse_certificado SET validade = CURRENT_DATE + 200 WHERE empresa_id = $1`, empID)
	db.Exec(`UPDATE financeiro.contas_financeiras SET provedor_config = NULL WHERE id = $1`, contaID)
	if n, _ := AvaliarAlertasCredenciais(ctx, db, agora, ok); n != 0 {
		t.Errorf("credencial com 200 dias não gera alerta: n=%d", n)
	}
	_ = fmt.Sprint()
}
