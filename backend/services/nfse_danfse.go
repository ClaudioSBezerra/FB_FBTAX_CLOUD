package services

// DANFSe (Documento Auxiliar da NFS-e) gerado localmente a partir do XML da NFS-e autorizada,
// reproduzindo a organização do DANFSe v2.0 oficial (cabeçalho com chave e QR, prestador,
// tomador, serviço, tributação municipal/federal, IBS/CBS, valores e informações
// complementares). É um documento auxiliar: a validade da nota é a do XML autorizado, e o PDF
// oficial do portal nacional continua sendo a referência quando disponível.

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/code"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/linestyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

type danfseEndereco struct {
	XLgr    string `xml:"xLgr"`
	Nro     string `xml:"nro"`
	XCpl    string `xml:"xCpl"`
	XBairro string `xml:"xBairro"`
	CMun    string `xml:"cMun"`
	UF      string `xml:"UF"`
	CEP     string `xml:"CEP"`
}

type danfseNFSe struct {
	XMLName xml.Name `xml:"NFSe"`
	Inf     struct {
		ID        string `xml:"Id,attr"`
		XLocEmi   string `xml:"xLocEmi"`
		XLocPrest string `xml:"xLocPrestacao"`
		NNFSe     string `xml:"nNFSe"`
		CLocIncid string `xml:"cLocIncid"`
		XLocIncid string `xml:"xLocIncid"`
		XTribNac  string `xml:"xTribNac"`
		AmbGer    string `xml:"ambGer"`
		DhProc    string `xml:"dhProc"`
		Emit      struct {
			CNPJ  string         `xml:"CNPJ"`
			XNome string         `xml:"xNome"`
			Ender danfseEndereco `xml:"enderNac"`
			Fone  string         `xml:"fone"`
			Email string         `xml:"email"`
		} `xml:"emit"`
		IBSCBS struct {
			CLocalidade string `xml:"cLocalidadeIncid"`
			XLocalidade string `xml:"xLocalidadeIncid"`
			Valores     struct {
				VBC string `xml:"vBC"`
				UF  struct {
					P string `xml:"pIBSUF"`
				} `xml:"uf"`
				Mun struct {
					P string `xml:"pIBSMun"`
				} `xml:"mun"`
				Fed struct {
					P string `xml:"pCBS"`
				} `xml:"fed"`
			} `xml:"valores"`
			Tot struct {
				VTotNF string `xml:"vTotNF"`
				IBS    struct {
					Total string `xml:"vIBSTot"`
					UF    struct {
						V string `xml:"vIBSUF"`
					} `xml:"gIBSUFTot"`
					Mun struct {
						V string `xml:"vIBSMun"`
					} `xml:"gIBSMunTot"`
				} `xml:"gIBS"`
				CBS struct {
					V string `xml:"vCBS"`
				} `xml:"gCBS"`
			} `xml:"totCIBS"`
		} `xml:"IBSCBS"`
		DPS struct {
			Inf struct {
				TpAmb   string `xml:"tpAmb"`
				DhEmi   string `xml:"dhEmi"`
				Serie   string `xml:"serie"`
				NDPS    string `xml:"nDPS"`
				DCompet string `xml:"dCompet"`
				Prest   struct {
					RegTrib struct {
						OpSimpNac   string `xml:"opSimpNac"`
						RegApTribSN string `xml:"regApTribSN"`
					} `xml:"regTrib"`
				} `xml:"prest"`
				Toma struct {
					CNPJ  string `xml:"CNPJ"`
					CPF   string `xml:"CPF"`
					XNome string `xml:"xNome"`
					End   struct {
						Nac struct {
							CMun string `xml:"cMun"`
							CEP  string `xml:"CEP"`
						} `xml:"endNac"`
						XLgr    string `xml:"xLgr"`
						Nro     string `xml:"nro"`
						XCpl    string `xml:"xCpl"`
						XBairro string `xml:"xBairro"`
					} `xml:"end"`
					Fone  string `xml:"fone"`
					Email string `xml:"email"`
				} `xml:"toma"`
				Serv struct {
					Loc   string `xml:"locPrest>cLocPrestacao"`
					CTrib string `xml:"cServ>cTribNac"`
					Desc  string `xml:"cServ>xDescServ"`
					CNBS  string `xml:"cServ>cNBS"`
				} `xml:"serv"`
				Valores struct {
					VServ string `xml:"vServPrest>vServ"`
				} `xml:"valores"`
				IBSCBS struct {
					CST        string `xml:"valores>trib>gIBSCBS>CST"`
					CClass     string `xml:"valores>trib>gIBSCBS>cClassTrib"`
					CSTNovo    string `xml:"valores>trib>CST"`
					CClassNovo string `xml:"valores>trib>cClassTrib"`
					COp        string `xml:"cIndOp"`
				} `xml:"IBSCBS"`
			} `xml:"infDPS"`
		} `xml:"DPS"`
	} `xml:"infNFSe"`
}

// DANFSeExtras traz dados que não estão no XML (o município do tomador só tem o código IBGE).
type DANFSeExtras struct {
	TomadorMunicipio string
	TomadorUF        string
	Cancelada        bool
}

func mascaraDoc(s string) string {
	d := soDigitos(s)
	switch len(d) {
	case 14:
		return d[0:2] + "." + d[2:5] + "." + d[5:8] + "/" + d[8:12] + "-" + d[12:]
	case 11:
		return d[0:3] + "." + d[3:6] + "." + d[6:9] + "-" + d[9:]
	}
	return s
}

func mascaraIBGE(s string) string {
	if d := soDigitos(s); len(d) == 7 {
		return d[:2] + "." + d[2:]
	}
	return vazio(s)
}

func mascaraCEP(s string) string {
	if d := soDigitos(s); len(d) == 8 {
		return d[:5] + "-" + d[5:]
	}
	return s
}

func mascaraFone(s string) string {
	d := soDigitos(s)
	switch len(d) {
	case 11:
		return "(" + d[:2] + ") " + d[2:7] + "-" + d[7:]
	case 10:
		return "(" + d[:2] + ") " + d[2:6] + "-" + d[6:]
	}
	if d == "" {
		return "-"
	}
	return s
}

func reais(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	var v float64
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%f", &v); err != nil {
		return s
	}
	inteiro := int64(v)
	cent := int64((v-float64(inteiro))*100 + 0.5)
	if cent >= 100 {
		inteiro++
		cent -= 100
	}
	str := fmt.Sprintf("%d", inteiro)
	var b strings.Builder
	for i, r := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return fmt.Sprintf("R$ %s,%02d", b.String(), cent)
}

// somaReais soma dois valores decimais em texto ("24.00" + "216.00") e formata em reais.
func somaReais(a, b string) string {
	var x, y float64
	fmt.Sscanf(strings.TrimSpace(a), "%f", &x)
	fmt.Sscanf(strings.TrimSpace(b), "%f", &y)
	return reais(fmt.Sprintf("%.2f", x+y))
}

func pct(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return strings.Replace(s, ".", ",", 1) + " %"
}

func vazio(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return strings.TrimSpace(s)
}

func dataHora(s string) string {
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(s)); err == nil {
		return t.Format("02/01/2006 15:04:05")
	}
	return vazio(s)
}

func dataSimples(s string) string {
	if t, err := time.Parse("2006-01-02", strings.TrimSpace(s)); err == nil {
		return t.Format("02/01/2006")
	}
	return vazio(s)
}

// GerarDANFSe monta o PDF a partir do XML da NFS-e (já descompactado).
func GerarDANFSe(nfseXML string, ex DANFSeExtras) ([]byte, error) {
	var n danfseNFSe
	if err := xml.Unmarshal([]byte(nfseXML), &n); err != nil {
		return nil, fmt.Errorf("XML da NFS-e ilegível: %w", err)
	}
	chave := strings.TrimPrefix(n.Inf.ID, "NFS")
	if len(chave) != 50 {
		return nil, errors.New("XML da NFS-e sem chave de acesso válida")
	}
	d := n.Inf.DPS.Inf

	m := maroto.New(config.NewBuilder().WithLeftMargin(8).WithRightMargin(8).WithTopMargin(6).WithBottomMargin(5).Build())

	rot := props.Text{Size: 6.5, Style: fontstyle.Bold, Color: CorApoio}
	val := props.Text{Size: 8, Color: CorTexto}
	tit := props.Text{Size: 8, Style: fontstyle.Bold, Color: CorClaro, Left: 1}

	// campo = rótulo pequeno em negrito sobre o valor.
	campo := func(tam int, rotulo, valor string) core.Col {
		return col.New(tam).Add(
			text.New(rotulo, props.Text{Size: 6.5, Style: fontstyle.Bold, Color: CorApoio, Top: 0.5}),
			text.New(valor, props.Text{Size: 8, Color: CorTexto, Top: 4}),
		)
	}
	secao := func(nome string) core.Row {
		return row.New(5).WithStyle(&props.Cell{BackgroundColor: CorMarca}).Add(col.New(12).Add(text.New(nome, tit)))
	}
	linhaSep := func() core.Row {
		return row.New(1).Add(col.New(12).Add(line.New(props.Line{Color: CorFiaR, Style: linestyle.Solid})))
	}
	_ = rot
	_ = val

	// ── Cabeçalho ──
	m.AddRows(
		row.New(14).Add(
			col.New(3).Add(text.New("NFS-e", props.Text{Size: 16, Style: fontstyle.Bold, Color: CorMarca, Top: 2}), text.New("Nota Fiscal de Serviço eletrônica", props.Text{Size: 6.5, Color: CorApoio, Top: 9})),
			col.New(6).Add(text.New("DANFSe v2.0", props.Text{Size: 11, Style: fontstyle.Bold, Align: align.Center, Top: 1}), text.New("Documento Auxiliar da NFS-e", props.Text{Size: 8, Align: align.Center, Top: 6.5})),
			col.New(3).Add(text.New("Município: "+n.Inf.XLocEmi+" - "+n.Inf.Emit.Ender.UF, props.Text{Size: 7, Align: align.Right, Style: fontstyle.Bold, Top: 1}),
				text.New("Ambiente Gerador: "+vazio(n.Inf.AmbGer)+"   Tipo de Ambiente: "+vazio(d.TpAmb), props.Text{Size: 6, Align: align.Right, Top: 5})),
		),
		linhaSep(),
	)
	situacao := "NFS-e Gerada"
	if ex.Cancelada {
		situacao = "NFS-e CANCELADA"
	}
	m.AddRows(
		row.New(19).Add(
			col.New(9).Add(
				text.New("CHAVE DE ACESSO DA NFS-e", props.Text{Size: 6.5, Style: fontstyle.Bold, Color: CorApoio}),
				text.New(chave, props.Text{Size: 9, Style: fontstyle.Bold, Top: 4}),
				text.New("A autenticidade desta NFS-e pode ser verificada pela leitura do código QR ou pela consulta da chave de acesso no portal nacional da NFS-e.", props.Text{Size: 6.5, Color: CorApoio, Top: 10}),
			),
			code.NewQrCol(3, "https://www.nfse.gov.br/ConsultaPublica/?tpc=1&chave="+chave, props.Rect{Center: true, Percent: 92}),
		),
		row.New(8.1).Add(
			campo(3, "NÚMERO DA NFS-e", vazio(n.Inf.NNFSe)),
			campo(3, "COMPETÊNCIA DA NFS-e", dataSimples(d.DCompet)),
			campo(6, "DATA E HORA DA EMISSÃO DA NFS-e", dataHora(n.Inf.DhProc)),
		),
		row.New(8.1).Add(
			campo(3, "NÚMERO DA DPS", vazio(d.NDPS)),
			campo(3, "SÉRIE DA DPS", vazio(d.Serie)),
			campo(6, "DATA E HORA DA EMISSÃO DA DPS", dataHora(d.DhEmi)),
		),
		row.New(8.1).Add(
			campo(3, "EMITENTE DA NFS-e", "Prestador"),
			campo(3, "SITUAÇÃO DA NFS-e", situacao),
			campo(6, "FINALIDADE", "NFS-e regular"),
		),
	)

	// ── Prestador ──
	e := n.Inf.Emit
	ender := strings.Join(nonEmpty(e.Ender.XLgr, e.Ender.Nro, e.Ender.XBairro), ", ")
	regime := "Optante - Microempresa ou Empresa de Pequeno Porte"
	if d.Prest.RegTrib.OpSimpNac == "1" {
		regime = "Não optante"
	} else if d.Prest.RegTrib.OpSimpNac == "2" {
		regime = "Optante - Microempreendedor Individual (MEI)"
	}
	m.AddRows(
		secao("PRESTADOR / FORNECEDOR"),
		row.New(8.1).Add(campo(4, "CNPJ / CPF / NIF", mascaraDoc(e.CNPJ)), campo(4, "Indicador Municipal (Inscrição)", "-"), campo(4, "Telefone", mascaraFone(e.Fone))),
		row.New(8.1).Add(campo(6, "Nome / Nome Empresarial", vazio(e.XNome)), campo(3, "Município / UF", n.Inf.XLocEmi+" / "+e.Ender.UF), campo(3, "Código IBGE / CEP", mascaraIBGE(e.Ender.CMun)+" / "+mascaraCEP(e.Ender.CEP))),
		row.New(8.1).Add(campo(8, "Endereço", ender), campo(4, "E-mail", vazio(e.Email))),
		row.New(8.1).Add(campo(5, "Simples Nacional na Data de Competência", regime), campo(7, "Regime de Apuração Tributária pelo SN", "Regime de apuração dos tributos federais e municipal pelo Simples Nacional")),
	)

	// ── Tomador ──
	t := d.Toma
	docT := t.CNPJ
	if docT == "" {
		docT = t.CPF
	}
	munT := vazio(ex.TomadorMunicipio)
	if ex.TomadorUF != "" {
		munT += " / " + ex.TomadorUF
	}
	enderT := strings.Join(nonEmpty(t.End.XLgr, t.End.Nro, t.End.XCpl, t.End.XBairro), ", ")
	m.AddRows(
		secao("TOMADOR / ADQUIRENTE"),
		row.New(8.1).Add(campo(4, "CNPJ / CPF / NIF", mascaraDoc(docT)), campo(4, "Indicador Municipal (Inscrição)", "-"), campo(4, "Telefone", mascaraFone(t.Fone))),
		row.New(8.1).Add(campo(6, "Nome / Nome Empresarial", vazio(t.XNome)), campo(3, "Município / UF", munT), campo(3, "Código IBGE / CEP", mascaraIBGE(t.End.Nac.CMun)+" / "+mascaraCEP(t.End.Nac.CEP))),
		row.New(12).Add(campo(8, "Endereço", enderT), campo(4, "E-mail", vazio(t.Email))),
		row.New(5).Add(col.New(12).Add(text.New("DESTINATÁRIO DA OPERAÇÃO NÃO IDENTIFICADO NA NFS-e   |   INTERMEDIÁRIO DA OPERAÇÃO NÃO IDENTIFICADO NA NFS-e", props.Text{Size: 6.5, Align: align.Center, Color: CorApoio, Top: 1}))),
	)

	// ── Serviço ──
	s := d.Serv
	cTrib := s.CTrib
	if len(cTrib) == 6 {
		cTrib = cTrib[0:2] + "." + cTrib[2:4] + "." + cTrib[4:] + " / -"
	}
	nbs := s.CNBS
	if len(nbs) == 9 {
		nbs = nbs[0:1] + "." + nbs[1:5] + "." + nbs[5:7] + "." + nbs[7:]
	}
	m.AddRows(
		secao("SERVIÇO PRESTADO"),
		row.New(8.1).Add(campo(4, "Código de Tributação Nacional/Municipal", vazio(cTrib)), campo(4, "Código da NBS", vazio(nbs)), campo(4, "Local da Prestação", vazio(n.Inf.XLocPrest))),
		row.New(7).Add(col.New(12).Add(text.New(vazio(n.Inf.XTribNac), props.Text{Size: 8, Color: CorTexto, Top: 1}))),
		row.New(12).Add(col.New(12).Add(
			text.New("Descrição do Serviço", props.Text{Size: 6.5, Style: fontstyle.Bold, Color: CorApoio}),
			text.New(vazio(s.Desc), props.Text{Size: 8, Color: CorTexto, Top: 4}),
		)),
	)

	// ── Tributação ──
	m.AddRows(
		secao("TRIBUTAÇÃO MUNICIPAL (ISSQN)"),
		row.New(8.1).Add(campo(4, "Tipo de Tributação do ISSQN", "Operação Tributável"), campo(4, "Município de Incidência do ISSQN", vazio(n.Inf.XLocIncid)), campo(4, "Retenção do ISSQN", "Não Retido")),
		row.New(8.1).Add(campo(4, "BC ISSQN", "-"), campo(4, "Alíquota Aplicada", "-"), campo(4, "ISSQN Apurado", "-")),
		secao("TRIBUTAÇÃO FEDERAL (EXCETO CBS)"),
		row.New(8.1).Add(campo(4, "IRRF", "-"), campo(4, "Contribuição Previdenciária - Retida", "-"), campo(4, "Contribuições Sociais - Retidas", "-")),
		row.New(8.1).Add(campo(4, "PIS - Débito Apuração Própria", "-"), campo(4, "COFINS - Débito Apuração Própria", "-"), campo(4, "Descrição Contrib. Sociais - Retidas", "0 - PIS/COFINS/CSLL Não Retidos")),
	)
	ib := n.Inf.IBSCBS
	cst, cls := d.IBSCBS.CST, d.IBSCBS.CClass
	if cst == "" {
		cst, cls = d.IBSCBS.CSTNovo, d.IBSCBS.CClassNovo
	}
	m.AddRows(
		secao("TRIBUTAÇÃO IBS/CBS"),
		row.New(8.1).Add(campo(4, "CST / cClassTrib", vazio(cst)+" / "+vazio(cls)), campo(8, "Indicador de Operação / Código IBGE Incidência / Município Incidência", vazio(d.IBSCBS.COp)+" / "+vazio(ib.CLocalidade)+" / "+vazio(ib.XLocalidade))),
		row.New(8.1).Add(campo(4, "Base de Cálculo Após Exclusões e Reduções", reais(ib.Valores.VBC)), campo(4, "Alíquota - IBS UF / IBS Mun", pct(ib.Valores.UF.P)+" / "+pct(ib.Valores.Mun.P)), campo(4, "Alíquota - CBS", pct(ib.Valores.Fed.P))),
		row.New(8.1).Add(campo(4, "Valor Apurado Estadual - IBS", reais(ib.Tot.IBS.UF.V)), campo(4, "Valor Apurado Municipal - IBS", reais(ib.Tot.IBS.Mun.V)), campo(4, "Valor Total Apurado - CBS", reais(ib.Tot.CBS.V))),
	)

	// ── Valor total ──
	m.AddRows(
		secao("VALOR TOTAL DA NFS-e"),
		row.New(8.1).Add(campo(4, "VALOR DA OPERAÇÃO / SERVIÇO", reais(d.Valores.VServ)), campo(4, "Desconto Incondicionado", "-"), campo(4, "Desconto Condicionado", "-")),
		row.New(8.1).Add(campo(4, "Total das Retenções (ISSQN / Federais)", "-"), campo(4, "VALOR LÍQUIDO DA NFS-e", reais(d.Valores.VServ)), campo(4, "Total do IBS/CBS", somaReais(ib.Tot.IBS.Total, ib.Tot.CBS.V))),
		secao("INFORMAÇÕES COMPLEMENTARES"),
		row.New(10).Add(col.New(12).Add(text.New("Totais aproximados dos Tributos cfe. Lei n° 12.741/2012: Federais: -; Estaduais: -; Municipais: -", props.Text{Size: 7.5, Top: 1}))),
		row.New(8).Add(col.New(12).Add(text.New("Documento auxiliar gerado pelo FBTax Cloud a partir do XML autorizado da NFS-e. Consulte a validade pela chave de acesso no portal nacional da NFS-e.", props.Text{Size: 6.5, Color: CorApoio, Top: 2}))),
	)

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("gerar PDF: %w", err)
	}
	return doc.GetBytes(), nil
}

func nonEmpty(ss ...string) []string {
	var out []string
	for _, s := range ss {
		if s = strings.Join(strings.Fields(s), " "); s != "" {
			out = append(out, s)
		}
	}
	return out
}
