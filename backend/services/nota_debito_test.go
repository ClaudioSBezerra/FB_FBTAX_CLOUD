package services

import (
	"archive/zip"
	"bytes"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

func dia(s string) time.Time { t, _ := time.Parse("2006-01-02", s); return t }

func TestConsolidarDespesas(t *testing.T) {
	ls := ConsolidarDespesas([]DespesaNota{
		{dia("2026-11-10"), "TÁXI / APLICATIVO", "Uber", 17.98},
		{dia("2026-11-08"), "TÁXI / APLICATIVO", "Uber", 57.06},
		{dia("2026-11-08"), "HOSPEDAGEM", "Hospedagem AIRBNB", 3812.46},
		{dia("2026-11-09"), "ALIMENTAÇÃO - ALMOÇO", "Almoço", 65.83},
	})
	if len(ls) != 3 {
		t.Fatalf("linhas: %+v", ls)
	}
	// ordem do modelo: táxi, almoço, hospedagem
	if ls[0].Tipo != "TÁXI / APLICATIVO" || ls[1].Tipo != "ALIMENTAÇÃO - ALMOÇO" || ls[2].Tipo != "HOSPEDAGEM" {
		t.Errorf("ordem: %+v", ls)
	}
	if ls[0].Valor != 75.04 || ls[0].Descricao != "Táxi / aplicativo de 08/11/2026 a 10/11/2026" {
		t.Errorf("táxi: %+v", ls[0])
	}
	if ls[2].Descricao != "Hospedagem AIRBNB" {
		t.Errorf("lançamento único mantém a descrição: %+v", ls[2])
	}
}

func TestGerarNotaDebitoXLSX_PreencheModeloEMantemFormulasEValidacao(t *testing.T) {
	d := DadosNotaDebito{
		Emissao: dia("2026-11-14"), Vencimento: dia("2026-11-24"),
		PeriodoIni: dia("2026-11-08"), PeriodoFim: dia("2026-11-27"),
		Destino: "Recife - PE", EmissorNome: "FORTES BEZERRA TESTE", EmissorCNPJ: "38149716000128",
		ReceptorNome: "FERREIRA COSTA & CIA LTDA", ReceptorCNPJ: "08.456.789/0001-00", Solicitante: "Fulano",
		Linhas: []LinhaNota{
			{"TÁXI / APLICATIVO", "Uber", 204.92},
			{"HOSPEDAGEM", "Hospedagem AIRBNB", 3812.46},
		},
	}
	if d.Total() != 4017.38 {
		t.Fatalf("total %v", d.Total())
	}
	b, err := GerarNotaDebitoXLSX(d)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	raw := func(c string) string {
		v, _ := f.GetCellValue("FOR.FIN.0002", c, excelize.Options{RawCellValue: true})
		return v
	}
	if raw("A5") != "FORTES BEZERRA TESTE" || raw("AD5") != "38.149.716/0001-28" || raw("A17") != "FERREIRA COSTA & CIA LTDA" {
		t.Errorf("emissor/receptor: %q %q %q", raw("A5"), raw("AD5"), raw("A17"))
	}
	if raw("E28") != "Uber" || raw("AD29") != "HOSPEDAGEM" || raw("AJ28") != "204.92" || raw("AJ29") != "3812.46" {
		t.Errorf("linhas: %q %q %q %q", raw("E28"), raw("AD29"), raw("AJ28"), raw("AJ29"))
	}
	if raw("E30") != "" || raw("AJ30") != "" || raw("AD30") != "" {
		t.Errorf("linha de exemplo do modelo deve ser limpa: %q %q", raw("E30"), raw("AJ30"))
	}
	if raw("AD11") != "08/11/2026 a 27/11/2026" {
		t.Errorf("período: %q", raw("AD11"))
	}
	// total e resumo continuam sendo fórmulas do modelo (o motor do excelize conta células
	// mescladas em dobro, então o valor calculado só é conferido no Excel/LibreOffice)
	for cell, trecho := range map[string]string{"AJ44": "SUM(", "O57": "SUMIF(", "AJ56": "SUMIF("} {
		if fm, _ := f.GetCellFormula("FOR.FIN.0002", cell); !strings.Contains(fm, trecho) {
			t.Errorf("fórmula de %s perdida: %q", cell, fm)
		}
	}
	// valores guardados nas fórmulas (o que visualizadores de e-mail/celular mostram)
	zc, _ := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	for _, zf := range zc.File {
		if zf.Name == "xl/worksheets/sheet1.xml" {
			r, _ := zf.Open()
			x, _ := io.ReadAll(r)
			for cel, v := range map[string]string{"AJ44": "4017.38", "O57": "204.92", "AJ56": "3812.46", "O60": "0.00"} {
				if !strings.Contains(string(x), `<c r="`+cel+`"`) || !regexp.MustCompile(`<c r="`+cel+`"[^>]*><f>[^<]*</f><v>`+regexp.QuoteMeta(v)+`</v>`).Match(x) {
					t.Errorf("valor em cache de %s deveria ser %s", cel, v)
				}
			}
		}
	}
	// a lista suspensa de tipos (validação de dados do modelo) sobrevive
	z, _ := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	for _, zf := range z.File {
		if zf.Name == "xl/worksheets/sheet1.xml" {
			r, _ := zf.Open()
			x, _ := io.ReadAll(r)
			if !strings.Contains(string(x), "Validação!$B$3:$B$14") {
				t.Error("lista de tipos de despesa perdida")
			}
		}
	}
	// o modelo embutido não carrega dados do emissor
	if strings.Contains(string(templateNotaDebito), "154159476") {
		t.Error("modelo embutido não deve conter a conta bancária")
	}
}

func TestGerarNotaDebitoXLSX_Limites(t *testing.T) {
	if _, err := GerarNotaDebitoXLSX(DadosNotaDebito{}); err == nil {
		t.Error("sem linhas deve falhar")
	}
	var muitas []LinhaNota
	for i := 0; i < 16; i++ {
		muitas = append(muitas, LinhaNota{"OUTRAS", "x", 1})
	}
	if _, err := GerarNotaDebitoXLSX(DadosNotaDebito{Linhas: muitas}); err == nil {
		t.Error("mais de 15 linhas deve falhar")
	}
}

func TestCodigoNotaDebito(t *testing.T) {
	e := dia("2026-11-14")
	if c := CodigoNotaDebito(e, nil); c != "FOR.FIN.0002-112026" {
		t.Errorf("%s", c)
	}
	if c := CodigoNotaDebito(e, []string{"FOR.FIN.0002-112026"}); c != "FOR.FIN.0002-112026-2" {
		t.Errorf("%s", c)
	}
	if c := CodigoNotaDebito(e, []string{"FOR.FIN.0002-112026", "FOR.FIN.0002-112026-2"}); c != "FOR.FIN.0002-112026-3" {
		t.Errorf("%s", c)
	}
}

func TestTipoDespesaValido(t *testing.T) {
	if !TipoDespesaValido("HOSPEDAGEM") || TipoDespesaValido("hospedagem") || TipoDespesaValido("") {
		t.Error("lista fixa do modelo")
	}
}
