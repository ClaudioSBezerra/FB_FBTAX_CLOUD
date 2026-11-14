package services

// Vencimento das credenciais que mantêm o sistema funcionando: certificado da integração do Inter
// (1 ano) e certificado A1 da NFS-e. Sem elas o sistema deixa de cobrar, conciliar e emitir notas,
// então o aviso começa 30 dias antes (tela e e-mail) e se repete nos marcos 15, 7, 3, 1 e 0 dias.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LimiteAlertaCredencialDias é a antecedência do primeiro aviso.
const LimiteAlertaCredencialDias = 30

// marcosAlertaCredencial são os dias restantes em que um e-mail é enviado (uma vez por validade).
var marcosAlertaCredencial = []int{30, 15, 7, 3, 1, 0}

// CredencialVencimento descreve uma credencial e quanto falta para vencer.
type CredencialVencimento struct {
	Chave         string `json:"chave"`
	Tipo          string `json:"tipo"` // inter | nfse
	Nome          string `json:"nome"`
	Validade      string `json:"validade"` // AAAA-MM-DD
	DiasRestantes int    `json:"dias_restantes"`
	Nivel         string `json:"nivel"` // ok | atencao (≤30 dias) | critico (≤7) | vencida
	Acao          string `json:"acao"`  // o que fazer para renovar
	Link          string `json:"link"`  // tela do sistema onde atualizar
}

func nivelCredencial(dias int) string {
	switch {
	case dias < 0:
		return "vencida"
	case dias <= 7:
		return "critico"
	case dias <= LimiteAlertaCredencialDias:
		return "atencao"
	}
	return "ok"
}

// brasilia é o fuso em que os dias são contados (os certificados informam a validade em UTC).
var brasilia = time.FixedZone("-03", -3*3600)

// diasAte conta os dias de calendário (em Brasília) entre hoje e a validade (o último dia de uso).
func diasAte(validade time.Time, agora time.Time) int {
	v := time.Date(validade.Year(), validade.Month(), validade.Day(), 0, 0, 0, 0, time.UTC)
	a := agora.In(brasilia)
	h := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	return int(v.Sub(h).Hours() / 24)
}

// ListarVencimentosCredenciais devolve todas as credenciais conhecidas, da que vence primeiro à última.
func ListarVencimentosCredenciais(ctx context.Context, db *sql.DB, agora time.Time) ([]CredencialVencimento, error) {
	var out []CredencialVencimento

	// Certificado A1 da NFS-e (validade gravada no upload, extraída do próprio .pfx).
	rows, err := db.QueryContext(ctx, `
		SELECT c.empresa_id::text, c.validade, COALESCE(e.razao_social, '')
		FROM financeiro.nfse_certificado c LEFT JOIN financeiro.empresas e ON e.id = c.empresa_id`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, razao string
		var v time.Time
		if err := rows.Scan(&id, &v, &razao); err != nil {
			rows.Close()
			return nil, err
		}
		d := diasAte(v, agora)
		out = append(out, CredencialVencimento{
			Chave: "nfse:" + id, Tipo: "nfse", Nome: "Certificado A1 da NFS-e" + sufixo(razao),
			Validade: v.Format("2006-01-02"), DiasRestantes: d, Nivel: nivelCredencial(d),
			Acao: "Compre/renove o certificado A1 e envie o novo arquivo em Financeiro › NFS-e.", Link: "/admin/financeiro/nfse",
		})
	}
	rows.Close()

	// Certificado da integração do Inter (validade de 1 ano; renovação liberada 90 dias antes).
	rows, err = db.QueryContext(ctx, `
		SELECT id::text, apelido, provedor_config FROM financeiro.contas_financeiras
		WHERE ativa = true AND provedor = 'inter' AND provedor_config IS NOT NULL AND provedor_config <> ''`)
	if err != nil {
		return nil, err
	}
	type conta struct{ id, apelido, cfg string }
	var contas []conta
	for rows.Next() {
		var c conta
		if rows.Scan(&c.id, &c.apelido, &c.cfg) == nil {
			contas = append(contas, c)
		}
	}
	rows.Close()
	for _, c := range contas {
		js, err := DecryptConfig(c.cfg)
		if err != nil {
			continue
		}
		var cfg ProviderConfig
		if json.Unmarshal([]byte(js), &cfg) != nil || cfg.CertPEM == "" || cfg.KeyPEM == "" {
			continue
		}
		cert, err := ValidarParCertificadoInter(cfg.CertPEM, cfg.KeyPEM)
		if err != nil {
			continue
		}
		validade := cert.NotAfter.In(brasilia)
		d := diasAte(validade, agora)
		out = append(out, CredencialVencimento{
			Chave: "inter:" + c.id, Tipo: "inter", Nome: "Certificado da integração do Inter — " + c.apelido,
			Validade: validade.Format("2006-01-02"), DiasRestantes: d, Nivel: nivelCredencial(d),
			Acao: "No Internet Banking: Integrar › Minhas integrações › Renovar (baixe o novo .crt e .key) e atualize em Financeiro › Manutenção do banco. O Client ID e o Secret não mudam.",
			Link: "/admin/financeiro/bancos",
		})
	}

	// mais urgentes primeiro
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].DiasRestantes < out[j-1].DiasRestantes; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}

func sufixo(razao string) string {
	if strings.TrimSpace(razao) == "" {
		return ""
	}
	return " — " + razao
}

// ErrSMTPNaoConfigurado indica que o e-mail não pôde ser enviado por falta de configuração SMTP.
var ErrSMTPNaoConfigurado = errors.New("SMTP não configurado")

// EnviarEmailTexto envia um e-mail de texto simples pelo SMTP do sistema.
func EnviarEmailTexto(to []string, assunto, corpo string) error {
	cfg := GetEmailConfig()
	if cfg.Password == "" {
		return ErrSMTPNaoConfigurado
	}
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		cfg.From, strings.Join(to, ", "), assunto, corpo))
	return sendMailSSL(cfg, to, msg)
}

// DestinatariosAlertaCredenciais: FINANCEIRO_ADMIN_EMAIL (ou o usuário SMTP) e o e-mail da empresa.
func DestinatariosAlertaCredenciais(ctx context.Context, db *sql.DB) []string {
	var out []string
	vistos := map[string]bool{}
	add := func(e string) {
		e = strings.TrimSpace(e)
		if e != "" && !vistos[strings.ToLower(e)] {
			vistos[strings.ToLower(e)] = true
			out = append(out, e)
		}
	}
	if e := os.Getenv("FINANCEIRO_ADMIN_EMAIL"); e != "" {
		add(e)
	} else {
		add(os.Getenv("SMTP_USER"))
	}
	var email sql.NullString
	db.QueryRowContext(ctx, `SELECT email FROM financeiro.empresas ORDER BY created_at LIMIT 1`).Scan(&email)
	add(email.String)
	return out
}

// AvaliarAlertasCredenciais envia UM e-mail com as credenciais que cruzaram um marco novo (30, 15,
// 7, 3, 1 ou 0 dias) e registra os marcos como avisados. Se o envio falhar, nada é registrado e a
// próxima execução tenta de novo. Devolve quantas credenciais foram avisadas.
func AvaliarAlertasCredenciais(ctx context.Context, db *sql.DB, agora time.Time, enviar func(to []string, assunto, corpo string) error) (int, error) {
	creds, err := ListarVencimentosCredenciais(ctx, db, agora)
	if err != nil {
		return 0, err
	}
	type pendente struct {
		c      CredencialVencimento
		marcos []int
	}
	var novos []pendente
	for _, c := range creds {
		if c.DiasRestantes > LimiteAlertaCredencialDias {
			continue
		}
		validade, _ := time.Parse("2006-01-02", c.Validade)
		var faltam []int
		for _, m := range marcosAlertaCredencial {
			if c.DiasRestantes > m {
				continue
			}
			var ja bool
			if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM financeiro.credencial_alertas WHERE chave=$1 AND validade=$2 AND marco=$3)`, c.Chave, validade.Format("2006-01-02"), m).Scan(&ja); err != nil {
				return 0, err
			}
			if !ja {
				faltam = append(faltam, m)
			}
		}
		if len(faltam) > 0 {
			novos = append(novos, pendente{c, faltam})
		}
	}
	if len(novos) == 0 {
		return 0, nil
	}

	var b strings.Builder
	b.WriteString("Fortes Bezerra — Módulo Financeiro\n\nAtenção: credenciais perto de vencer. Sem elas o sistema deixa de acessar o banco e de emitir NFS-e.\n\n")
	for _, n := range novos {
		c := n.c
		situacao := fmt.Sprintf("vence em %d dia(s)", c.DiasRestantes)
		if c.DiasRestantes < 0 {
			situacao = fmt.Sprintf("VENCIDA há %d dia(s)", -c.DiasRestantes)
		} else if c.DiasRestantes == 0 {
			situacao = "vence HOJE"
		}
		fmt.Fprintf(&b, "• %s\n  Validade: %s (%s)\n  O que fazer: %s\n\n", c.Nome, dataBR(c.Validade), situacao, c.Acao)
	}
	b.WriteString("Esta é uma mensagem automática. Não responda a este e-mail.")

	to := DestinatariosAlertaCredenciais(ctx, db)
	if len(to) == 0 {
		return 0, errors.New("nenhum destinatário configurado (FINANCEIRO_ADMIN_EMAIL ou SMTP_USER)")
	}
	assunto := "ALERTA: credencial vencendo — " + novos[0].c.Nome
	if len(novos) > 1 {
		assunto = fmt.Sprintf("ALERTA: %d credenciais vencendo", len(novos))
	}
	if err := enviar(to, assunto, b.String()); err != nil {
		return 0, err
	}
	for _, n := range novos {
		for _, m := range n.marcos {
			if _, err := db.ExecContext(ctx, `INSERT INTO financeiro.credencial_alertas (chave, validade, marco) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, n.c.Chave, n.c.Validade, m); err != nil {
				return 0, err
			}
		}
	}
	return len(novos), nil
}

func dataBR(iso string) string {
	if t, err := time.Parse("2006-01-02", iso); err == nil {
		return t.Format("02/01/2006")
	}
	return iso
}

// StartCredentialExpiryChecker verifica o vencimento das credenciais ao subir e a cada 6 horas.
func StartCredentialExpiryChecker(db *sql.DB) {
	go func() {
		time.Sleep(90 * time.Second) // deixa o banco e as migrations assentarem
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			n, err := AvaliarAlertasCredenciais(ctx, db, time.Now(), EnviarEmailTexto)
			cancel()
			switch {
			case errors.Is(err, ErrSMTPNaoConfigurado):
				log.Printf("[Credenciais] há credencial perto de vencer, mas o SMTP não está configurado: o aviso fica só na tela")
			case err != nil:
				log.Printf("[Credenciais] verificação de vencimento: %v", err)
			case n > 0:
				log.Printf("[Credenciais] alerta de vencimento enviado (%d credencial(is))", n)
			}
			time.Sleep(6 * time.Hour)
		}
	}()
}
