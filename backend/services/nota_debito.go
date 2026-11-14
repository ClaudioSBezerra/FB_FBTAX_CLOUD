package services

// Nota de Débito de reembolso de despesas de viagem (modelo FOR.FIN.0002): preenche o .xlsx
// embutido com os dados do emissor, do receptor e das despesas consolidadas por tipo.

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

//go:embed templates/FOR.FIN.0002.xlsx
var templateNotaDebito []byte

const (
	notaDebitoAba          = "FOR.FIN.0002"
	NotaDebitoPrazoDias    = 10 // o modelo manda vencer 10 dias após a emissão
	notaDebitoLinhaInicial = 28
	notaDebitoLinhaFinal   = 42
)

// TiposDespesaViagem é a lista fixa da aba "Validação" do modelo, na ordem da planilha.
var TiposDespesaViagem = []string{
	"PASSAGEM AÉREA / RODOVIÁRIA",
	"TÁXI / APLICATIVO",
	"ALUGUEL / KILOMETRAGEM / COMBUSTÍVEL",
	"ALIMENTAÇÃO - CAFÉ DA MANHÃ",
	"ALIMENTAÇÃO - ALMOÇO",
	"ALIMENTAÇÃO - JANTAR",
	"HOSPEDAGEM",
	"TAXA DE ESTACIONAMENTO/ PEDÁGIO",
	"EXTRAS DE HOTEL",
	"DESCARREGO",
	"REPRESENTAÇÃO",
	"OUTRAS",
}

// TipoDespesaValido diz se o tipo está na lista do modelo.
func TipoDespesaValido(t string) bool {
	for _, x := range TiposDespesaViagem {
		if x == t {
			return true
		}
	}
	return false
}

// DespesaNota é a despesa já aprovada que entra na nota.
type DespesaNota struct {
	Data      time.Time
	Tipo      string
	Descricao string
	Valor     float64
}

// LinhaNota é uma linha consolidada (um tipo de despesa) da nota.
type LinhaNota struct {
	Tipo      string
	Descricao string
	Valor     float64
}

// ConsolidarDespesas agrupa por tipo, na ordem do modelo. Um lançamento sozinho mantém a sua
// descrição; vários viram "<Tipo> de dd/mm/aaaa a dd/mm/aaaa". Valores somados em centavos.
func ConsolidarDespesas(ds []DespesaNota) []LinhaNota {
	type acc struct {
		cent     int64
		ini, fim time.Time
		n        int
		desc     string
	}
	por := map[string]*acc{}
	for _, d := range ds {
		a := por[d.Tipo]
		if a == nil {
			a = &acc{ini: d.Data, fim: d.Data, desc: d.Descricao}
			por[d.Tipo] = a
		}
		a.n++
		a.cent += int64(d.Valor*100 + 0.5)
		if d.Data.Before(a.ini) {
			a.ini = d.Data
		}
		if d.Data.After(a.fim) {
			a.fim = d.Data
		}
	}
	var out []LinhaNota
	for _, tipo := range TiposDespesaViagem {
		a := por[tipo]
		if a == nil {
			continue
		}
		desc := a.desc
		if a.n > 1 {
			desc = fmt.Sprintf("%s de %s a %s", rotuloTipo(tipo), a.ini.Format("02/01/2006"), a.fim.Format("02/01/2006"))
		}
		out = append(out, LinhaNota{Tipo: tipo, Descricao: desc, Valor: float64(a.cent) / 100})
	}
	return out
}

func rotuloTipo(t string) string {
	switch t {
	case "TÁXI / APLICATIVO":
		return "Táxi / aplicativo"
	case "ALIMENTAÇÃO - CAFÉ DA MANHÃ":
		return "Alimentação (café da manhã)"
	case "ALIMENTAÇÃO - ALMOÇO":
		return "Alimentação (almoço)"
	case "ALIMENTAÇÃO - JANTAR":
		return "Alimentação (jantar)"
	}
	s := strings.ToLower(t)
	return strings.ToUpper(s[:1]) + s[1:]
}

// DadosNotaDebito reúne tudo o que o modelo pede.
type DadosNotaDebito struct {
	Emissao                                time.Time
	Vencimento                             time.Time
	PeriodoIni, PeriodoFim                 time.Time
	Destino                                string
	EmissorNome, EmissorCNPJ, EmissorFone  string
	EmissorIE, EmissorEndereco             string
	Banco, Agencia, Conta                  string
	ReceptorNome, ReceptorCNPJ, ReceptorIE string
	ReceptorEndereco                       string
	CentroCustoDesc, CentroCustoNum        string
	ContaContabilDesc, ContaContabilNum    string
	Solicitante                            string
	Linhas                                 []LinhaNota
}

// Total soma as linhas (em centavos, para não acumular erro de ponto flutuante).
func (d DadosNotaDebito) Total() float64 {
	var c int64
	for _, l := range d.Linhas {
		c += int64(l.Valor*100 + 0.5)
	}
	return float64(c) / 100
}

// FormatarCNPJ aplica a máscara 00.000.000/0000-00 quando há 14 dígitos; senão devolve o original.
func FormatarCNPJ(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if len(d) != 14 {
		return s
	}
	return d[0:2] + "." + d[2:5] + "." + d[5:8] + "/" + d[8:12] + "-" + d[12:14]
}

// GerarNotaDebitoXLSX preenche o modelo e devolve o .xlsx.
func GerarNotaDebitoXLSX(d DadosNotaDebito) ([]byte, error) {
	if len(d.Linhas) == 0 {
		return nil, errors.New("a nota precisa de ao menos uma despesa")
	}
	if len(d.Linhas) > notaDebitoLinhaFinal-notaDebitoLinhaInicial+1 {
		return nil, fmt.Errorf("o modelo comporta no máximo %d linhas", notaDebitoLinhaFinal-notaDebitoLinhaInicial+1)
	}
	f, err := excelize.OpenReader(bytes.NewReader(templateNotaDebito))
	if err != nil {
		return nil, fmt.Errorf("abrir modelo: %w", err)
	}
	defer f.Close()

	set := func(cell string, v any) {
		if err == nil {
			err = f.SetCellValue(notaDebitoAba, cell, v)
		}
	}
	set("AJ2", d.Emissao)
	set("A5", d.EmissorNome)
	set("AD5", FormatarCNPJ(d.EmissorCNPJ))
	set("A7", d.EmissorIE)
	set("AD7", d.EmissorFone)
	set("A9", d.EmissorEndereco)
	set("A11", d.Destino)
	set("AD11", d.PeriodoIni.Format("02/01/2006")+" a "+d.PeriodoFim.Format("02/01/2006"))
	set("A14", d.Banco)
	set("N14", d.Agencia)
	set("AC14", d.Conta)
	set("A17", d.ReceptorNome)
	set("A19", FormatarCNPJ(d.ReceptorCNPJ))
	set("AD19", d.ReceptorIE)
	set("A21", d.ReceptorEndereco)
	set("A24", d.CentroCustoDesc)
	set("AD24", d.CentroCustoNum)
	set("A26", d.ContaContabilDesc)
	set("AD26", d.ContaContabilNum)
	for r := notaDebitoLinhaInicial; r <= notaDebitoLinhaFinal; r++ {
		i := r - notaDebitoLinhaInicial
		if i < len(d.Linhas) {
			l := d.Linhas[i]
			set(fmt.Sprintf("A%d", r), d.Emissao)
			set(fmt.Sprintf("E%d", r), l.Descricao)
			set(fmt.Sprintf("X%d", r), 1)
			set(fmt.Sprintf("AD%d", r), l.Tipo)
			set(fmt.Sprintf("AJ%d", r), l.Valor)
		} else {
			for _, c := range []string{"A", "E", "X", "AD", "AJ"} {
				set(fmt.Sprintf("%s%d", c, r), nil)
			}
		}
	}
	set("AJ46", d.Vencimento)
	set("C51", d.Solicitante)
	set("D53", d.Emissao)
	if err != nil {
		return nil, fmt.Errorf("preencher modelo: %w", err)
	}
	// Totais e resumo são fórmulas do modelo: força o Excel/LibreOffice a recalcular ao abrir.
	full := true
	if err := f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: &full}); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return gravarTotaisEmCache(buf.Bytes(), d)
}

// Células de resumo do modelo: tipo (rótulo fixo da planilha) -> célula com o SUMIF.
var resumoNotaDebito = map[string]string{
	"PASSAGEM AÉREA / RODOVIÁRIA": "O56", "TÁXI / APLICATIVO": "O57",
	"ALUGUEL / KILOMETRAGEM / COMBUSTÍVEL": "O58", "ALIMENTAÇÃO - CAFÉ DA MANHÃ": "O59",
	"ALIMENTAÇÃO - ALMOÇO": "O60", "ALIMENTAÇÃO - JANTAR": "O61",
	"HOSPEDAGEM": "AJ56", "EXTRAS DE HOTEL": "AJ57", "DESCARREGO": "AJ58",
	"REPRESENTAÇÃO": "AJ59", "OUTRAS": "AJ60",
}

// gravarTotaisEmCache escreve os resultados corretos nas fórmulas do modelo. Excel e LibreOffice
// recalculam ao abrir, mas visualizadores de e-mail e de celular mostram só o valor guardado,
// que no modelo é o da nota antiga.
func gravarTotaisEmCache(xlsx []byte, d DadosNotaDebito) ([]byte, error) {
	valores := map[string]float64{"AJ44": d.Total()}
	for _, c := range resumoNotaDebito {
		valores[c] = 0
	}
	cent := map[string]int64{}
	for _, l := range d.Linhas {
		if c, ok := resumoNotaDebito[l.Tipo]; ok {
			cent[c] += int64(l.Valor*100 + 0.5)
		}
	}
	for c, v := range cent {
		valores[c] = float64(v) / 100
	}
	zr, err := zip.NewReader(bytes.NewReader(xlsx), int64(len(xlsx)))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		if zf.Name == "xl/worksheets/sheet1.xml" {
			x := string(body)
			for cel, v := range valores {
				re := regexp.MustCompile(`(<c r="` + cel + `"[^>]*><f>[^<]*</f>)<v>[^<]*</v>`)
				x = re.ReplaceAllString(x, "${1}<v>"+strconv.FormatFloat(v, 'f', 2, 64)+"</v>")
			}
			body = []byte(x)
		}
		w, err := zw.Create(zf.Name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// CodigoNotaDebito monta FOR.FIN.0002-MMAAAA, com sufixo -n a partir da segunda nota do mês.
func CodigoNotaDebito(emissao time.Time, existentes []string) string {
	base := "FOR.FIN.0002-" + emissao.Format("012006")
	if !contem(existentes, base) {
		return base
	}
	sort.Strings(existentes)
	for n := 2; ; n++ {
		c := fmt.Sprintf("%s-%d", base, n)
		if !contem(existentes, c) {
			return c
		}
	}
}

func contem(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
