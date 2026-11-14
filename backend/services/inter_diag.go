package services

// Diagnóstico da integração com o Banco Inter: valida o certificado, descobre quais escopos a
// integração realmente tem (pedindo um token por escopo, sem usá-lo) e testa saldo. Serve à
// tela "Bancos" e evita que o usuário precise mandar prints ou segredos para conferir permissões.

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"
)

// InterEscopoTestado é o resultado de pedir um token com um único escopo.
type InterEscopoTestado struct {
	Escopo    string `json:"escopo"`
	Concedido bool   `json:"concedido"`
	Perigoso  bool   `json:"perigoso,omitempty"` // permite movimentar dinheiro
	// Indeterminado: o Inter recusou por excesso de requisições (429); não é falta de permissão.
	Indeterminado bool   `json:"indeterminado,omitempty"`
	Detalhe       string `json:"detalhe,omitempty"`
}

// InterDiagnostico reúne o que o teste de conexão descobriu.
type InterDiagnostico struct {
	CertValidade      string               `json:"cert_validade,omitempty"`
	CertDiasRestantes int                  `json:"cert_dias_restantes"`
	TokenOK           bool                 `json:"token_ok"`
	Escopos           []InterEscopoTestado `json:"escopos"`
	SaldoOK           bool                 `json:"saldo_ok"`
	SaldoDisponivel   float64              `json:"saldo_disponivel,omitempty"`
	BankingVersao     string               `json:"banking_versao,omitempty"`
	Avisos            []string             `json:"avisos,omitempty"`
}

// interEscoposSonda são os escopos verificados. Nomes desconhecidos pelo Inter aparecem como
// "não concedido"; os marcados como perigosos movimentam dinheiro e não devem estar ativos.
var interEscoposSonda = []struct {
	Nome     string
	Perigoso bool
}{
	{"extrato.read", false},
	{"cobv.read", false}, {"cobv.write", false},
	{"cob.read", false}, {"cob.write", false},
	{"pix.read", false},
	{"webhook.read", false}, {"webhook.write", false},
	{"boleto-cobranca.read", false}, {"boleto-cobranca.write", false},
	{"pix.write", true},
	{"pagamento-boleto.write", true}, {"pagamento-pix.write", true}, {"pagamento-darf.write", true},
}

// ValidarParCertificadoInter confere que o .crt e a .key formam um par e devolve o certificado.
func ValidarParCertificadoInter(certPEM, keyPEM string) (*x509.Certificate, error) {
	if _, err := tls.X509KeyPair(decodePEM(certPEM), decodePEM(keyPEM)); err != nil {
		return nil, fmt.Errorf("o certificado (.crt) e a chave (.key) não formam um par válido: %w", err)
	}
	bloco, _ := pem.Decode(decodePEM(certPEM))
	if bloco == nil {
		return nil, errors.New("certificado (.crt) não está em formato PEM")
	}
	return x509.ParseCertificate(bloco.Bytes)
}

// DiagnosticarInter roda o teste completo. Nunca devolve segredos.
func DiagnosticarInter(cfg ProviderConfig) *InterDiagnostico {
	d := &InterDiagnostico{Escopos: []InterEscopoTestado{}}
	cert, err := ValidarParCertificadoInter(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		d.Avisos = append(d.Avisos, err.Error())
		return d
	}
	d.CertValidade = cert.NotAfter.Format("2006-01-02")
	d.CertDiasRestantes = int(time.Until(cert.NotAfter).Hours() / 24)
	switch {
	case d.CertDiasRestantes < 0:
		d.Avisos = append(d.Avisos, "certificado VENCIDO: renove a integração no Internet Banking (Integrar > Minhas integrações > Renovar)")
	case d.CertDiasRestantes <= 90:
		d.Avisos = append(d.Avisos, fmt.Sprintf("certificado vence em %d dias: a renovação já está disponível no Internet Banking", d.CertDiasRestantes))
	}

	// 1) Uma única chamada com todos os escopos: a documentação do Inter diz que a resposta traz os
	//    escopos concedidos. 2) Se o Inter recusar o conjunto (escopo desconhecido/não permitido),
	//    cai para um token por escopo, em série e com intervalo (o endpoint responde 429 a rajadas).
	todos := make([]string, len(interEscoposSonda))
	for i, e := range interEscoposSonda {
		todos[i] = e.Nome
	}
	d.Escopos = make([]InterEscopoTestado, len(interEscoposSonda))
	_, concedidos, errConj := interObterTokenEscopos(cfg, strings.Join(todos, " "))
	if errConj == nil && strings.TrimSpace(concedidos) != "" {
		set := map[string]bool{}
		for _, c := range strings.Fields(concedidos) {
			set[c] = true
		}
		for i, e := range interEscoposSonda {
			d.Escopos[i] = InterEscopoTestado{Escopo: e.Nome, Perigoso: e.Perigoso, Concedido: set[e.Nome]}
		}
	} else {
		for i, e := range interEscoposSonda {
			r := InterEscopoTestado{Escopo: e.Nome, Perigoso: e.Perigoso}
			if _, err := interObterToken(cfg, e.Nome); err == nil {
				r.Concedido = true
			} else if errors.Is(err, ErrInterLimite) {
				r.Indeterminado = true
				r.Detalhe = "limite de requisições do Inter; teste de novo em alguns minutos"
			} else {
				r.Detalhe = limitarTexto(err.Error(), 160)
			}
			d.Escopos[i] = r
		}
	}
	for _, r := range d.Escopos {
		if r.Concedido {
			d.TokenOK = true
			if r.Perigoso {
				d.Avisos = append(d.Avisos, "a integração tem o escopo "+r.Escopo+", que permite movimentar dinheiro: recomendo criar outra integração sem esse escopo")
			}
		}
		if r.Indeterminado {
			d.Avisos = append(d.Avisos, "o Inter limitou as requisições (429) durante o teste: o resultado de alguns escopos é indeterminado; aguarde alguns minutos e teste de novo")
			break
		}
	}
	if !d.TokenOK {
		d.Avisos = append(d.Avisos, "nenhum token foi emitido: confira Client ID, Client Secret e se o certificado é o da MESMA integração (senão o Inter responde 401)")
		return d
	}

	if strings.TrimSpace(cfg.ContaCorrente) == "" {
		d.Avisos = append(d.Avisos, "informe a conta corrente (cabeçalho x-conta-corrente) para testar o saldo")
		return d
	}
	raw, versao, err := interBankingGET(cfg, "/saldo", "")
	if err != nil {
		d.Avisos = append(d.Avisos, "saldo: "+limitarTexto(err.Error(), 200))
		return d
	}
	var s struct {
		Disponivel float64 `json:"disponivel"`
	}
	if json.Unmarshal(raw, &s) == nil {
		d.SaldoOK, d.SaldoDisponivel, d.BankingVersao = true, s.Disponivel, versao
	}
	return d
}

func limitarTexto(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}
