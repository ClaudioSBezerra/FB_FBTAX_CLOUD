package handlers

import (
	"testing"

	"fb_cloud/services"
)

// LGPD (#1): e-mail/WhatsApp só entram depois do consentimento.
func TestAplicarAtualizacoes_ConsentimentoAntesDoContato(t *testing.T) {
	l := leadVendas{}
	aplicarAtualizacoes(&l, []services.AtualizacaoLead{{Nome: "Ana", Empresa: "X", Email: "ana@x.com"}})
	if l.Email != "" {
		t.Error("e-mail sem consentimento não pode ser gravado")
	}
	if l.Nome != "Ana" || l.Empresa != "X" {
		t.Errorf("qualificação deveria entrar: %+v", l)
	}
	aplicarAtualizacoes(&l, []services.AtualizacaoLead{{Email: "Ana@X.com", Whatsapp: "62 9999", ConsentimentoContato: true}})
	if l.Email != "ana@x.com" || l.Whatsapp != "62 9999" || !l.Consentimento {
		t.Errorf("com consentimento deveria gravar: %+v", l)
	}
	aplicarAtualizacoes(&l, []services.AtualizacaoLead{{Nome: ""}})
	if l.Nome != "Ana" {
		t.Error("campo vazio não pode apagar o que já existe")
	}
}

func TestStatusLead(t *testing.T) {
	l := leadVendas{}
	if statusLead(&l, false) != "conversa" {
		t.Error("sem dados = conversa")
	}
	l.Empresa, l.ERP = "X", "SAP"
	if statusLead(&l, false) != "qualificado" {
		t.Error("empresa + ERP = qualificado")
	}
	if statusLead(&l, true) != "quente" {
		t.Error("encaminhado = quente")
	}
	l.Status = "contatado"
	if statusLead(&l, true) != "contatado" {
		t.Error("não volta a quente depois de contatado pela equipe")
	}
}

func TestOrigensLandingNoCORS(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://www.fbtax.cloud")
	m := GetAllowedOrigins()
	if !m["https://apuracaoassistida.ia.br"] || !m["https://www.fbtax.cloud"] {
		t.Errorf("origens: %v", m)
	}
}
