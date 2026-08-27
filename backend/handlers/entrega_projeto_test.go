package handlers

import (
	"strings"
	"testing"

	"fb_cloud/services"
)

// TestGerarTermoEntregaPDF cobre o caminho completo do Termo de Entrega no
// Padrão FBTECH — capa, tabelas, linhas pautadas e blocos de assinatura.
func TestGerarTermoEntregaPDF(t *testing.T) {
	emp := emitente{
		RazaoSocial: "FORTES BEZERRA TECNOLOGIA E CONSULTORIA LTDA",
		CNPJ:        "38.149.716/0001-28",
		Municipio:   "Aparecida de Goiânia",
		UF:          "GO",
	}

	t.Run("termo preenchido", func(t *testing.T) {
		e := &EntregaProjeto{
			Numero: "TE-2026-001", Versao: "1.0",
			Titulo:      "Termo de Entrega e Aceite de Projeto",
			Projeto:     "Implantação FB_SMARTPICK — Grupo JC",
			ClienteNome: "GRUPO JC DISTRIBUIDORA LTDA", ClienteCNPJ: "12345678000199",
			ProdutoNome: "SmartPick", ProdutoCod: "FB_SMARTPICK",
			Ambiente:     "WMS Winthor (Totvs) — CD Goiânia",
			DataInicio:   "2026-05-04",
			DataGolive:   "2026-08-10",
			DataEntrega:  "2026-08-26",
			GarantiaDias: 30, AssistidaDias: 14, LinhasLivres: 4,
			ResponsavelTecnico: "Claudio Bezerra",
			Itens: []EntregaItem{
				{Descricao: "Integração com WMS Winthor", Situacao: "entregue"},
				{Descricao: "Painel executivo (CEO)", Situacao: "parcial"},
			},
			Pendencias: []EntregaPendencia{
				{Descricao: "Exportação XLSX", Responsavel: "FBTECH", Prazo: "2026-09-15"},
			},
			Signatarios: []EntregaSignatario{
				{Papel: "PELA CONTRATADA", Nome: "Claudio Bezerra"},
				{Papel: "PELO CLIENTE"},
			},
		}
		b, err := gerarTermoEntregaPDF(e, emp)
		if err != nil {
			t.Fatalf("gerarTermoEntregaPDF: %v", err)
		}
		if len(b) < 2000 {
			t.Fatalf("PDF suspeito de estar vazio: %d bytes", len(b))
		}
		if !strings.HasPrefix(string(b[:5]), "%PDF-") {
			t.Fatalf("saída não é um PDF: %q", b[:5])
		}
	})

	// Sem itens, pendências ou signatários, o termo ainda precisa ser emitido:
	// as seções viram linhas pautadas para preenchimento à mão.
	t.Run("termo em branco para preencher à mão", func(t *testing.T) {
		e := &EntregaProjeto{
			Numero: "TE-2026-002", Versao: "1.0",
			Titulo:       "Termo de Entrega e Aceite de Projeto",
			ClienteNome:  "CLIENTE EXEMPLO LTDA",
			ProdutoNome:  "SmartPick",
			ProdutoCod:   "FB_SMARTPICK",
			DataEntrega:  "2026-08-26",
			GarantiaDias: 30, AssistidaDias: 14, LinhasLivres: 6,
		}
		b, err := gerarTermoEntregaPDF(e, emp)
		if err != nil {
			t.Fatalf("gerarTermoEntregaPDF: %v", err)
		}
		if len(b) < 2000 {
			t.Fatalf("PDF suspeito de estar vazio: %d bytes", len(b))
		}
	})
}

func TestPorExtensoPrazos(t *testing.T) {
	casos := map[int]string{14: "quatorze", 30: "trinta", 45: "quarenta e cinco", 90: "noventa"}
	for n, esperado := range casos {
		if got := services.PorExtenso(n); got != esperado {
			t.Errorf("PorExtenso(%d) = %q, esperado %q", n, got, esperado)
		}
	}
}
