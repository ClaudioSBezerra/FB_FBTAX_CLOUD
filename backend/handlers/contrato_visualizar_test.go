package handlers

import (
	"strings"
	"testing"
)

// TestGerarContratoPDF cobre o Contrato de Prestação de Serviços após a
// migração para o Padrão FBTECH. Vale como trava de regressão do kit: o
// contrato exercita Bloco, Tabela, Marcador, Campo e dois BlocoAssinaturas.
func TestGerarContratoPDF(t *testing.T) {
	valor := 4500.0
	d := &ContratoDetalhe{
		ID: "id", Numero: "FB-2026-007", DataInicio: "2026-06-15",
		Periodicidade: "mensal", ValorTotal: 4500.00, Status: "ativo",
		Observacoes: "Mensalidades negociadas em instrumento apartado.",
		Cliente: ContratoCliente{
			RazaoSocial: "JC DISTRIBUIÇÃO", CNPJ: "06314327000114",
			Municipio: "Aparecida de Goiânia", UF: "GO",
			Logradouro: "Rua 01, QD.01 Lote.02", NumeroEnd: "S/N",
		},
		Empresa: ContratoEmpresa{
			RazaoSocial: "FORTES BEZERRA TECNOLOGIA E CONSULTORIA LTDA",
			CNPJ:        "38149716000128", Logradouro: "Av. Central", Numero: "100",
			Municipio: "Aparecida de Goiânia", UF: "GO",
		},
		CNPJs: []ContratoCNPJ{
			{CNPJ: "06314327000114", Principal: true},
			{CNPJ: "06314327000203", Descricao: "CD Anápolis"},
		},
		Itens: []ContratoItem{
			{Produto: "SmartPick", Plano: "Enterprise", ValorItem: &valor},
			{Produto: "Farol", Plano: "Standard"}, // sem valor: imprime "Sob consulta"
		},
	}

	b, err := gerarContratoPDF(d)
	if err != nil {
		t.Fatalf("gerarContratoPDF: %v", err)
	}
	if !strings.HasPrefix(string(b[:5]), "%PDF-") {
		t.Fatalf("saída não é um PDF: %q", b[:5])
	}
	if len(b) < 10000 {
		t.Fatalf("PDF menor que o esperado para um contrato de 9 cláusulas: %d bytes", len(b))
	}

	// Contrato sem empresa cadastrada precisa cair nos dados institucionais.
	vazio := &ContratoDetalhe{
		ID: "id2", Numero: "FB-2026-008", DataInicio: "2026-06-15",
		Periodicidade: "mensal", ValorTotal: 100,
		Cliente: ContratoCliente{RazaoSocial: "CLIENTE EXEMPLO LTDA"},
	}
	if _, err := gerarContratoPDF(vazio); err != nil {
		t.Fatalf("contrato sem empresa cadastrada: %v", err)
	}
}
