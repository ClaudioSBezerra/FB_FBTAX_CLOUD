// Package services — fbdoc.go implementa o Padrão de Documentos FBTECH.
//
// O padrão foi derivado dos modelos de proposta adotados como referência pela
// empresa (Proposta Técnica, Proposta Comercial e Proposta de Licenciamento) e
// reescrito com a identidade visual FBTECH & IA. Todo documento institucional
// gerado pelo FBTax Cloud deve ser construído com este kit, garantindo capa,
// cabeçalho, rodapé, numeração de seções, tabelas e blocos de assinatura
// idênticos entre si.
//
// Referência do padrão: docs/templates/PADRAO-DOCUMENTOS-FBTECH.md
package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/linestyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// ── Paleta da marca ──────────────────────────────────────────────────────────
//
// Cores extraídas de docs/templates/fbtech-logo-dark.svg. Os modelos de
// referência usavam o azul #00174C como cor institucional; no padrão FBTECH o
// papel equivalente é do grafite-petróleo #0B1620, com o teal #14B8A6 como cor
// de destaque.

var (
	// CorMarca é o grafite-petróleo do fundo da logo — títulos e capa.
	CorMarca = &props.Color{Red: 11, Green: 22, Blue: 32}
	// CorDestaque é o teal da marca — filetes, numeração e ênfase.
	CorDestaque = &props.Color{Red: 20, Green: 184, Blue: 166}
	// CorTexto é o corpo de texto padrão.
	CorTexto = &props.Color{Red: 34, Green: 42, Blue: 53}
	// CorApoio é usada em legendas, rodapé e texto secundário.
	CorApoio = &props.Color{Red: 100, Green: 116, Blue: 139}
	// CorFiaR é o cinza dos filetes e bordas de tabela.
	CorFiaR = &props.Color{Red: 203, Green: 213, Blue: 225}
	// CorZebra é o fundo alternado das linhas de tabela.
	CorZebra = &props.Color{Red: 241, Green: 245, Blue: 249}
	// CorCabecalhoTabela é o fundo do cabeçalho de tabela.
	CorCabecalhoTabela = &props.Color{Red: 11, Green: 22, Blue: 32}
	// CorClaro é usada sobre fundos escuros.
	CorClaro = &props.Color{Red: 241, Green: 245, Blue: 249}
	// CorCampo é o traço das linhas destinadas ao preenchimento manual.
	CorCampo = &props.Color{Red: 148, Green: 163, Blue: 184}
)

// ── Métricas tipográficas ────────────────────────────────────────────────────
//
// Os modelos de referência foram diagramados em Calibri/Century Gothic, fontes
// proprietárias que não podem ser embarcadas no binário. O padrão FBTECH adota
// Helvetica (métrica Arial, presente em todo leitor de PDF) preservando a mesma
// hierarquia de corpos e a mesma escala de espaçamento dos originais.

const (
	CorpoTituloCapa    = 26.0
	CorpoSubtituloCapa = 13.0
	CorpoH1            = 13.0
	CorpoH2            = 11.0
	CorpoH3            = 9.5
	CorpoTexto         = 9.5
	CorpoTabela        = 8.5
	CorpoLegenda       = 7.5

	MargemEsquerda = 18.0
	MargemDireita  = 18.0
	MargemSuperior = 12.0
	MargemInferior = 14.0
)

// larguraHelveticaBold devolve, em milímetros, a largura de s em Helvetica-Bold
// no corpo informado. Usado para encaixar as duas cores da assinatura da marca
// sem depender de rasterização.
func larguraHelveticaBold(s string, corpo float64) float64 {
	// Larguras AFM (em milésimos de em) das letras usadas na marca.
	larguras := map[rune]float64{
		'F': 611, 'B': 722, 'T': 611, 'E': 667, 'C': 722, 'H': 722,
		'I': 278, 'A': 722, ' ': 278, '&': 722,
	}
	total := 0.0
	for _, r := range s {
		if w, ok := larguras[r]; ok {
			total += w
		} else {
			total += 600
		}
	}
	// milésimos de em → pontos → milímetros
	return total / 1000.0 * corpo * 25.4 / 72.0
}

// LogoFBTECH devolve a linha com a assinatura da marca FBTECH & IA desenhada em
// primitivas vetoriais do PDF — quadrado teal com o monograma "FB" seguido do
// logotipo bicolor. Reproduz docs/templates/fbtech-logo-dark.svg sem depender de
// rasterização externa, mantendo nitidez em qualquer zoom ou impressão.
//
// corpo é o tamanho da tipografia da marca (17 na capa, 9 no cabeçalho) e
// altura é a altura total da faixa. fundo é a cor sobre a qual a marca é
// aplicada; nil imprime sobre papel branco.
//
// Toda a faixa é uma única linha com apenas duas colunas. O fundo é pintado
// pelas colunas, e não por WithStyle na linha, porque o maroto deixa de pintar
// o fundo das colunas quando a linha tem estilo próprio — o que apagaria o
// quadrado teal do monograma. Manter uma linha só e o menor número possível de
// colunas evita as emendas visíveis entre células preenchidas adjacentes.
func LogoFBTECH(corpo, altura float64, fundo *props.Color) core.Row {
	corWordmark := CorMarca
	if fundo != nil {
		corWordmark = CorClaro
	}

	// Deslocamento vertical que centraliza uma linha de texto de tamanho c na
	// faixa: metade da altura menos metade da altura de x da fonte.
	centro := func(c float64) float64 {
		return altura*0.5 - c*0.35*25.4/72.0
	}

	// Coluna 1: quadrado teal com o monograma "FB" em negativo.
	monograma := col.New(1).
		WithStyle(&props.Cell{BackgroundColor: CorDestaque}).
		Add(text.New("FB", props.Text{
			Size: corpo * 0.72, Style: fontstyle.Bold, Family: fontfamily.Helvetica,
			Align: align.Center, Color: CorMarca, Top: centro(corpo * 0.72),
		}))

	// Coluna 2: logotipo bicolor. Os dois textos ocupam a mesma célula; o
	// deslocamento à esquerda do segundo é a largura exata do primeiro.
	recuo := corpo * 0.30
	larguraFB := larguraHelveticaBold("FB", corpo)

	wordmark := col.New(11).
		Add(text.New("FB", props.Text{
			Size: corpo, Style: fontstyle.Bold, Family: fontfamily.Helvetica,
			Align: align.Left, Color: corWordmark, Left: recuo, Top: centro(corpo),
		})).
		Add(text.New("TECH & IA", props.Text{
			Size: corpo, Style: fontstyle.Bold, Family: fontfamily.Helvetica,
			Align: align.Left, Color: CorDestaque, Left: recuo + larguraFB, Top: centro(corpo),
		}))
	if fundo != nil {
		wordmark.WithStyle(&props.Cell{BackgroundColor: fundo})
	}

	return row.New(altura).Add(monograma, wordmark)
}

// ── Metadados do documento ───────────────────────────────────────────────────

// DocMeta descreve o cabeçalho institucional de qualquer documento FBTECH.
// Segue a nomenclatura dos modelos de referência: uma sigla de tipo, o número
// sequencial e a versão compõem o código impresso em todas as páginas
// (ex.: "TE 33309 v.1.0").
type DocMeta struct {
	Sigla        string    // TE, PT, PC, PL — tipo do documento
	Numero       string    // número sequencial do documento
	Versao       string    // ex.: "1.0"
	Titulo       string    // título impresso na capa
	Subtitulo    string    // linha de apoio da capa (produto, projeto)
	Cliente      string    // razão social do destinatário
	Cidade       string    // cidade de emissão
	Data         time.Time // data de emissão
	Emitente     string    // razão social da FBTECH
	EmitCNPJ     string    // CNPJ do emitente
	Confidencial bool      // imprime a tarja de confidencialidade na capa
}

// Codigo devolve o identificador curto do documento, repetido no cabeçalho de
// todas as páginas — ex.: "TE 33309 v.1.0".
func (m DocMeta) Codigo() string {
	partes := []string{}
	if m.Sigla != "" {
		partes = append(partes, m.Sigla)
	}
	if m.Numero != "" {
		partes = append(partes, m.Numero)
	}
	cod := strings.Join(partes, " ")
	if m.Versao != "" {
		cod += " v." + m.Versao
	}
	return cod
}

// DataExtenso devolve a data por extenso usada na capa e no fecho.
func (m DocMeta) DataExtenso() string {
	meses := []string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho",
		"julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}
	return fmt.Sprintf("%d de %s de %d", m.Data.Day(), meses[int(m.Data.Month())], m.Data.Year())
}

// ── Documento ────────────────────────────────────────────────────────────────

// Doc encapsula um documento no Padrão FBTECH. Construa com NovoDoc, empilhe o
// conteúdo com os métodos de seção e finalize com Bytes.
type Doc struct {
	m      core.Maroto
	Meta   DocMeta
	altCab float64 // altura do cabeçalho registrado, usada na quebra de página
	sec    int     // contador da seção corrente (1., 2., 3. …)
	sub    int     // contador da subseção (1.1., 1.2. …)
	subsub int     // contador da sub-subseção (1.1.1. …)
}

// NovoDoc cria um documento já configurado com as margens, a fonte, os
// metadados PDF, o cabeçalho e o rodapé do padrão FBTECH.
func NovoDoc(meta DocMeta) *Doc {
	cfg := config.NewBuilder().
		WithLeftMargin(MargemEsquerda).
		WithRightMargin(MargemDireita).
		WithTopMargin(MargemSuperior).
		WithBottomMargin(MargemInferior).
		WithDefaultFont(&props.Font{
			Family: fontfamily.Helvetica,
			Size:   CorpoTexto,
			Color:  CorTexto,
		}).
		WithPageNumber(props.PageNumber{
			Pattern: "Página {current} de {total}",
			Place:   props.RightBottom,
			Family:  fontfamily.Helvetica,
			Size:    CorpoLegenda,
			Color:   CorApoio,
		}).
		WithTitle(meta.Titulo, true).
		WithAuthor(meta.Emitente, true).
		WithSubject(meta.Subtitulo, true).
		WithCreator("FBTax Cloud", true).
		WithCreationDate(meta.Data).
		Build()

	d := &Doc{m: maroto.New(cfg), Meta: meta}
	d.registrarCabecalho()
	d.registrarRodape()
	return d
}

// registrarCabecalho fixa em todas as páginas a faixa de identificação do
// documento: logo à esquerda, código e data à direita, sobre um filete teal.
func (d *Doc) registrarCabecalho() {
	// A soma das alturas é guardada porque FitlnCurrentPage contabiliza o
	// cabeçalho duas vezes; QuebraPagina compensa esse viés.
	d.altCab = 8 + 2 + 4
	d.m.RegisterHeader(
		row.New(8).Add(
			col.New(6).Add(text.New(d.Meta.Emitente, props.Text{
				Size: CorpoLegenda + 0.5, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorMarca, Top: 2,
			})),
			col.New(6).Add(text.New(
				fmt.Sprintf("%s  ·  %s", d.Meta.Codigo(), d.Meta.Data.Format("02/01/2006")),
				props.Text{
					Size: CorpoLegenda, Align: align.Right,
					Family: fontfamily.Helvetica, Color: CorApoio, Top: 2,
				})),
		),
		line.NewRow(2, props.Line{
			Style: linestyle.Solid, SizePercent: 100, Thickness: 0.6, Color: CorDestaque,
		}),
		row.New(4),
	)
}

// registrarRodape fixa a assinatura institucional no pé de todas as páginas.
// A numeração de páginas é injetada pelo WithPageNumber, à direita.
func (d *Doc) registrarRodape() {
	rodape := d.Meta.Emitente
	if d.Meta.EmitCNPJ != "" {
		rodape += " · CNPJ " + d.Meta.EmitCNPJ
	}
	if d.Meta.Confidencial {
		rodape += " · Confidencial"
	}
	d.m.RegisterFooter(
		line.NewRow(2, props.Line{
			Style: linestyle.Solid, SizePercent: 100, Thickness: 0.3, Color: CorFiaR,
		}),
		// Largura total: a numeração de páginas é posicionada pelo maroto na
		// margem inferior, fora destas linhas. Razões sociais longas não podem
		// quebrar o rodapé em duas linhas.
		row.New(6).Add(
			col.New(12).Add(text.New(rodape, props.Text{
				Size: CorpoLegenda, Family: fontfamily.Helvetica, Color: CorApoio, Top: 1.5,
			})),
		),
	)
}

// Add empilha linhas cruas no documento, para os casos que os helpers de seção
// não cobrem.
func (d *Doc) Add(rows ...core.Row) { d.m.AddRows(rows...) }

// Esp acrescenta um respiro vertical de h milímetros.
func (d *Doc) Esp(h float64) { d.m.AddRows(row.New(h)) }

// Bytes finaliza o documento e devolve o PDF serializado.
func (d *Doc) Bytes() ([]byte, error) {
	doc, err := d.m.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}

// ── Capa ─────────────────────────────────────────────────────────────────────

// Capa imprime a folha de rosto do padrão: bloco escuro com a assinatura da
// marca, título do documento, destinatário e código/versão, seguida da tarja de
// confidencialidade quando aplicável. Deve ser a primeira chamada do documento.
func (d *Doc) Capa() {
	// Bloco escuro com a marca — equivalente à capa institucional dos modelos
	// de referência, agora na identidade FBTECH. As faixas de respiro pintam o
	// fundo pela coluna, mantendo a mesma regra usada dentro da logo.
	d.m.AddRows(
		row.New(8),
		LogoFBTECH(17, 34, CorMarca),
	)

	d.m.AddRows(
		row.New(10),
		line.NewRow(3, props.Line{
			Style: linestyle.Solid, SizePercent: 22, Thickness: 1.4,
			Color: CorDestaque, OffsetPercent: 0,
		}),
		row.New(5),
	)

	titulo := strings.ToUpper(d.Meta.Titulo)
	d.m.AddRows(
		row.New(alturaTexto(titulo, CorpoTituloCapa, 12)).Add(col.New(12).Add(
			text.New(titulo, props.Text{
				Size: CorpoTituloCapa, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorMarca, Align: align.Left,
			}))),
	)
	if d.Meta.Subtitulo != "" {
		d.m.AddRows(
			row.New(3),
			row.New(alturaTexto(d.Meta.Subtitulo, CorpoSubtituloCapa, 12)).Add(col.New(12).Add(
				text.New(d.Meta.Subtitulo, props.Text{
					Size: CorpoSubtituloCapa, Family: fontfamily.Helvetica,
					Color: CorApoio, Align: align.Left,
				}))),
		)
	}

	d.m.AddRows(row.New(14))
	if d.Meta.Cliente != "" {
		d.campoCapa("Cliente", strings.ToUpper(d.Meta.Cliente))
	}
	d.campoCapa("Documento", d.Meta.Codigo())
	d.campoCapa("Emissão", d.Meta.DataExtenso())
	if d.Meta.Cidade != "" {
		d.campoCapa("Local", d.Meta.Cidade)
	}

	if d.Meta.Confidencial {
		aviso := "Este documento é confidencial e foi preparado exclusivamente para os " +
			"representantes do destinatário acima identificado. Seu conteúdo e eventuais " +
			"anexos não devem ser divulgados, total ou parcialmente, a terceiros."
		d.m.AddRows(
			row.New(16),
			line.NewRow(2, props.Line{
				Style: linestyle.Solid, SizePercent: 100, Thickness: 0.3, Color: CorFiaR,
			}),
			row.New(3),
			row.New(alturaTexto(aviso, CorpoLegenda+0.5, 12)).Add(col.New(12).Add(
				text.New(aviso, props.Text{
					Size: CorpoLegenda + 0.5, Family: fontfamily.Helvetica,
					Color: CorApoio, Align: align.Left,
				}))),
		)
	}
	d.QuebraPagina()
}

// campoCapa imprime um par rótulo/valor na folha de rosto.
func (d *Doc) campoCapa(rotulo, valor string) {
	d.m.AddRows(row.New(7).Add(
		col.New(3).Add(text.New(rotulo, props.Text{
			Size: CorpoTexto, Family: fontfamily.Helvetica, Color: CorApoio,
		})),
		col.New(9).Add(text.New(valor, props.Text{
			Size: CorpoTexto, Style: fontstyle.Bold,
			Family: fontfamily.Helvetica, Color: CorMarca,
		})),
	))
}

// QuebraPagina força o início de uma nova página sem deixar folhas em branco.
//
// O maroto quebra a página quando uma linha não cabe no espaço restante; um
// espaçador arbitrariamente alto geraria uma página vazia inteira. Por isso o
// espaço remanescente é localizado por busca binária com FitlnCurrentPage e
// preenchido exatamente, de modo que a próxima linha caia na página seguinte.
func (d *Doc) QuebraPagina() {
	alturaPagina := d.m.GetCurrentConfig().Dimensions.Height
	if !d.m.FitlnCurrentPage(0.1) {
		return // já está no limite: a próxima linha abre nova página
	}
	menor, maior := 0.0, alturaPagina
	for i := 0; i < 24; i++ {
		meio := (menor + maior) / 2
		if d.m.FitlnCurrentPage(meio) {
			menor = meio
		} else {
			maior = meio
		}
	}
	// FitlnCurrentPage soma o cabeçalho ao total já acumulado nas linhas, que
	// também o inclui; o espaço realmente livre é maior em altCab. Deixa uma
	// folga mínima para que o espaçador ainda caiba e a próxima linha não.
	espaco := menor + d.altCab - 0.05
	if espaco > 0.2 {
		d.m.AddRows(row.New(espaco))
	}
}

// ManterJunto garante que um bloco de altura mm não seja partido entre páginas:
// se não couber no espaço restante, abre a página seguinte antes de emiti-lo.
//
// A checagem é conservadora — FitlnCurrentPage subestima o espaço livre em
// altCab —, o que aqui é desejável: no limite quebra um pouco antes do
// necessário, nunca no meio do bloco.
func (d *Doc) ManterJunto(altura float64) {
	if !d.m.FitlnCurrentPage(altura) {
		d.QuebraPagina()
	}
}

// ── Seções numeradas ─────────────────────────────────────────────────────────

// Secao abre uma seção de primeiro nível ("1. Título"), reiniciando os
// contadores de subseção.
func (d *Doc) Secao(titulo string) {
	d.ManterJunto(30) // título + primeiras linhas na mesma página
	d.sec++
	d.sub, d.subsub = 0, 0
	d.m.AddRows(
		row.New(4),
		row.New(9).Add(
			col.New(1).Add(text.New(fmt.Sprintf("%d.", d.sec), props.Text{
				Size: CorpoH1, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorDestaque,
			})),
			col.New(11).Add(text.New(titulo, props.Text{
				Size: CorpoH1, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorMarca,
			})),
		),
		line.NewRow(2, props.Line{
			Style: linestyle.Solid, SizePercent: 100, Thickness: 0.4, Color: CorFiaR,
		}),
		row.New(3),
	)
}

// Bloco abre um título com o mesmo peso de uma seção, porém sem numeração
// automática. É a forma correta para documentos que já trazem a própria
// numeração no texto — contratos, cujas cláusulas se identificam como
// "CLÁUSULA 1ª", e anexos com numeração herdada de outro instrumento.
//
// Não altera os contadores de Secao/Subsecao.
func (d *Doc) Bloco(titulo string) {
	d.ManterJunto(30) // título + primeiras linhas na mesma página
	d.m.AddRows(
		row.New(4),
		row.New(alturaTexto(titulo, CorpoH2, 12)).Add(col.New(12).Add(
			text.New(titulo, props.Text{
				Size: CorpoH2, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorMarca,
			}))),
		line.NewRow(2, props.Line{
			Style: linestyle.Solid, SizePercent: 100, Thickness: 0.4, Color: CorFiaR,
		}),
		row.New(2.5),
	)
}

// Subsecao abre uma seção de segundo nível ("1.1. Título").
func (d *Doc) Subsecao(titulo string) {
	d.ManterJunto(24)
	d.sub++
	d.subsub = 0
	d.m.AddRows(
		row.New(3),
		row.New(7).Add(col.New(12).Add(text.New(
			fmt.Sprintf("%d.%d.  %s", d.sec, d.sub, titulo),
			props.Text{
				Size: CorpoH2, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorMarca,
			}))),
		row.New(2),
	)
}

// Subsubsecao abre uma seção de terceiro nível ("1.1.1. Título").
func (d *Doc) Subsubsecao(titulo string) {
	d.subsub++
	d.m.AddRows(
		row.New(2),
		row.New(6).Add(col.New(12).Add(text.New(
			fmt.Sprintf("%d.%d.%d.  %s", d.sec, d.sub, d.subsub, titulo),
			props.Text{
				Size: CorpoH3, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorTexto,
			}))),
		row.New(1.5),
	)
}

// ── Corpo de texto ───────────────────────────────────────────────────────────

// alturaTexto estima a altura necessária para s no corpo informado, dentro da
// largura útil da página. Evita que parágrafos longos sejam cortados.
func alturaTexto(s string, corpo float64, colunas int) float64 {
	larguraUtil := 210.0 - MargemEsquerda - MargemDireita
	larguraUtil *= float64(colunas) / 12.0
	// ~0,50 em por caractere em Helvetica; converte em → mm.
	larguraCaractere := corpo * 0.50 * 25.4 / 72.0
	porLinha := larguraUtil / larguraCaractere
	if porLinha < 1 {
		porLinha = 1
	}
	linhas := float64(len([]rune(s)))/porLinha + 1
	linhas += float64(strings.Count(s, "\n"))
	return linhas * corpo * 1.35 * 25.4 / 72.0
}

// Paragrafo imprime um bloco de texto corrido, dimensionando a altura
// automaticamente.
//
// O alinhamento é à esquerda, e não justificado como nos modelos de origem: o
// maroto justifica também a última linha do parágrafo, esticando-a de ponta a
// ponta sempre que o espaçamento resultante não ultrapassa dez vezes o normal.
// O resultado seria irregular de parágrafo para parágrafo.
func (d *Doc) Paragrafo(s string) {
	d.m.AddRows(
		row.New(alturaTexto(s, CorpoTexto, 12)).Add(col.New(12).Add(
			text.New(s, props.Text{
				Size: CorpoTexto, Family: fontfamily.Helvetica,
				Color: CorTexto, Align: align.Left,
			}))),
		row.New(2),
	)
}

// Marcador imprime um item de lista com bullet teal.
func (d *Doc) Marcador(s string) {
	altura := alturaTexto(s, CorpoTexto, 11)
	d.m.AddRows(row.New(altura).Add(
		col.New(1).Add(text.New("•", props.Text{
			Size: CorpoTexto, Style: fontstyle.Bold,
			Family: fontfamily.Helvetica, Color: CorDestaque, Align: align.Center,
		})),
		col.New(11).Add(text.New(s, props.Text{
			Size: CorpoTexto, Family: fontfamily.Helvetica,
			Color: CorTexto, Align: align.Left,
		})),
	))
}

// Nota imprime uma observação secundária, em corpo reduzido.
func (d *Doc) Nota(s string) {
	d.m.AddRows(
		row.New(alturaTexto(s, CorpoLegenda+0.5, 12)).Add(col.New(12).Add(
			text.New(s, props.Text{
				Size: CorpoLegenda + 0.5, Family: fontfamily.Helvetica,
				Color: CorApoio, Align: align.Left,
			}))),
		row.New(2),
	)
}

// ── Tabelas ──────────────────────────────────────────────────────────────────

// Coluna descreve uma coluna de tabela: rótulo, largura em unidades do grid de
// 12 e alinhamento do conteúdo.
type Coluna struct {
	Titulo  string
	Largura int
	Align   align.Type
}

// Tabela imprime um cabeçalho escuro seguido das linhas de dados com fundo
// alternado. As larguras das colunas devem somar 12.
func (d *Doc) Tabela(colunas []Coluna, linhas [][]string) {
	cabecalho := row.New(7).WithStyle(&props.Cell{BackgroundColor: CorCabecalhoTabela})
	for _, c := range colunas {
		al := c.Align
		if al == "" {
			al = align.Left
		}
		cabecalho.Add(col.New(c.Largura).Add(text.New(c.Titulo, props.Text{
			Size: CorpoTabela, Style: fontstyle.Bold, Family: fontfamily.Helvetica,
			Color: CorClaro, Align: al, Top: 2, Left: 1.5, Right: 1.5,
		})))
	}
	d.m.AddRows(cabecalho)

	for i, l := range linhas {
		altura := 6.5
		for j, celula := range l {
			if j < len(colunas) {
				if h := alturaTexto(celula, CorpoTabela, colunas[j].Largura) + 3; h > altura {
					altura = h
				}
			}
		}
		linha := row.New(altura)
		if i%2 == 1 {
			linha.WithStyle(&props.Cell{BackgroundColor: CorZebra})
		}
		for j, c := range colunas {
			valor := ""
			if j < len(l) {
				valor = l[j]
			}
			al := c.Align
			if al == "" {
				al = align.Left
			}
			linha.Add(col.New(c.Largura).Add(text.New(valor, props.Text{
				Size: CorpoTabela, Family: fontfamily.Helvetica,
				Color: CorTexto, Align: al, Top: 2, Left: 1.5, Right: 1.5,
			})))
		}
		d.m.AddRows(linha)
	}
	d.m.AddRows(
		line.NewRow(1.5, props.Line{
			Style: linestyle.Solid, SizePercent: 100, Thickness: 0.3, Color: CorFiaR,
		}),
		row.New(3),
	)
}

// Campo imprime um par rótulo/valor no corpo do documento.
func (d *Doc) Campo(rotulo, valor string) {
	d.m.AddRows(row.New(6).Add(
		col.New(4).Add(text.New(rotulo, props.Text{
			Size: CorpoTexto, Style: fontstyle.Bold,
			Family: fontfamily.Helvetica, Color: CorMarca,
		})),
		col.New(8).Add(text.New(valor, props.Text{
			Size: CorpoTexto, Family: fontfamily.Helvetica, Color: CorTexto,
		})),
	))
}

// ── Linhas para preenchimento manual ─────────────────────────────────────────

// LinhaPreenchivel imprime um rótulo seguido de um traço contínuo destinado ao
// preenchimento à mão na via impressa. Quando valor não é vazio, o dado é
// impresso sobre a linha — é assim que o módulo materializa no papel os campos
// informados na tela.
func (d *Doc) LinhaPreenchivel(rotulo, valor string, larguraRotulo int) {
	larguraValor := 12 - larguraRotulo
	campo := col.New(larguraValor).
		WithStyle(&props.Cell{
			BorderColor:     CorCampo,
			BorderType:      border.Bottom,
			BorderThickness: 0.3,
		})
	if valor != "" {
		campo.Add(text.New(valor, props.Text{
			Size: CorpoTexto, Family: fontfamily.Helvetica,
			Color: CorTexto, Left: 2, Top: 2.5,
		}))
	}
	d.m.AddRows(
		row.New(8).Add(
			col.New(larguraRotulo).Add(text.New(rotulo, props.Text{
				Size: CorpoTexto, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorMarca, Top: 2.5,
			})),
			campo,
		),
		row.New(2),
	)
}

// LinhasEmBranco imprime n linhas pautadas sem rótulo, para anotações manuscritas
// (pendências, ressalvas, observações levantadas no aceite).
func (d *Doc) LinhasEmBranco(n int) {
	for i := 0; i < n; i++ {
		d.m.AddRows(
			row.New(7).Add(col.New(12).WithStyle(&props.Cell{
				BorderColor:     CorCampo,
				BorderType:      border.Bottom,
				BorderThickness: 0.3,
			})),
			row.New(2),
		)
	}
}

// ── Assinaturas ──────────────────────────────────────────────────────────────

// Signatario descreve um bloco de assinatura do termo.
type Signatario struct {
	Papel string // ex.: "PELA CONTRATADA", "GESTOR DO CD"
	Nome  string // preenchido quando conhecido; vazio imprime linha em branco
	Cargo string
	Org   string
	Doc   string // número do documento
	// RotuloDoc é o rótulo impresso antes de Doc. Vazio imprime "CPF" — o caso
	// comum, de signatário pessoa física. Use "CNPJ" quando quem assina a linha
	// é a própria pessoa jurídica.
	RotuloDoc string
}

// BlocoAssinaturas imprime os signatários em pares, cada um com linha de
// assinatura e os campos de identificação. Nomes conhecidos são impressos;
// os demais campos ficam em branco para preenchimento no ato da assinatura.
func (d *Doc) BlocoAssinaturas(signatarios []Signatario) {
	linhaAssinatura := func() core.Col {
		return col.New(5).Add(text.New("_________________________________________", props.Text{
			Size: CorpoTexto, Family: fontfamily.Helvetica,
			Color: CorCampo, Align: align.Center,
		}))
	}
	identificacao := func(s Signatario) core.Col {
		c := col.New(5)
		nome := s.Nome
		if nome == "" {
			nome = "Nome: ______________________________"
		}
		c.Add(text.New(nome, props.Text{
			Size: CorpoTabela, Style: fontstyle.Bold,
			Family: fontfamily.Helvetica, Color: CorMarca, Align: align.Center,
		}))
		return c
	}

	for i := 0; i < len(signatarios); i += 2 {
		esq := signatarios[i]
		var dir *Signatario
		if i+1 < len(signatarios) {
			dir = &signatarios[i+1]
		}

		// Papel de cada signatário.
		linhaPapel := row.New(6).Add(
			col.New(5).Add(text.New(esq.Papel, props.Text{
				Size: CorpoTabela, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorDestaque, Align: align.Center,
			})),
			col.New(2),
		)
		if dir != nil {
			linhaPapel.Add(col.New(5).Add(text.New(dir.Papel, props.Text{
				Size: CorpoTabela, Style: fontstyle.Bold,
				Family: fontfamily.Helvetica, Color: CorDestaque, Align: align.Center,
			})))
		}

		linhaTraco := row.New(6).Add(linhaAssinatura(), col.New(2))
		linhaNome := row.New(5).Add(identificacao(esq), col.New(2))
		if dir != nil {
			linhaTraco.Add(linhaAssinatura())
			linhaNome.Add(identificacao(*dir))
		}

		// Um par de assinaturas não pode ser partido: papel, traço e campos de
		// identificação precisam ficar na mesma página para que o documento
		// tenha validade prática ao ser assinado.
		d.ManterJunto(58)
		d.m.AddRows(row.New(12), linhaPapel, row.New(4), linhaTraco, linhaNome)

		// Cargo, organização e documento — em branco quando não informados.
		for _, campo := range []struct {
			rotulo string
			get    func(Signatario) string
		}{
			{"Cargo", func(s Signatario) string { return s.Cargo }},
			{"Empresa", func(s Signatario) string { return s.Org }},
			{"CPF", func(s Signatario) string { return s.Doc }},
		} {
			texto := func(s Signatario) string {
				rotulo := campo.rotulo
				if rotulo == "CPF" && s.RotuloDoc != "" {
					rotulo = s.RotuloDoc
				}
				if v := campo.get(s); v != "" {
					return rotulo + ": " + v
				}
				return rotulo + ": ____________________________"
			}
			l := row.New(5).Add(
				col.New(5).Add(text.New(texto(esq), props.Text{
					Size: CorpoLegenda, Family: fontfamily.Helvetica,
					Color: CorApoio, Align: align.Center,
				})),
				col.New(2),
			)
			if dir != nil {
				l.Add(col.New(5).Add(text.New(texto(*dir), props.Text{
					Size: CorpoLegenda, Family: fontfamily.Helvetica,
					Color: CorApoio, Align: align.Center,
				})))
			}
			d.m.AddRows(l)
		}
	}
}

// LocalEData imprime o fecho com cidade e data por extenso a preencher.
func (d *Doc) LocalEData(cidade string) {
	d.m.AddRows(
		row.New(8),
		row.New(7).Add(col.New(12).Add(text.New(
			fmt.Sprintf("%s, ______ de ____________________ de ________.", cidade),
			props.Text{
				Size: CorpoTexto, Family: fontfamily.Helvetica,
				Color: CorTexto, Align: align.Left,
			}))),
	)
}

// PorExtenso escreve um número inteiro por extenso, usado nos prazos impressos
// nos documentos ("30 (trinta) dias").
func PorExtenso(n int) string {
	if n < 0 {
		return "menos " + PorExtenso(-n)
	}
	unidades := []string{"zero", "um", "dois", "três", "quatro", "cinco", "seis", "sete",
		"oito", "nove", "dez", "onze", "doze", "treze", "quatorze", "quinze", "dezesseis",
		"dezessete", "dezoito", "dezenove"}
	dezenas := []string{"", "", "vinte", "trinta", "quarenta", "cinquenta", "sessenta",
		"setenta", "oitenta", "noventa"}
	centenas := []string{"", "cento", "duzentos", "trezentos", "quatrocentos", "quinhentos",
		"seiscentos", "setecentos", "oitocentos", "novecentos"}

	switch {
	case n < 20:
		return unidades[n]
	case n < 100:
		r := dezenas[n/10]
		if n%10 != 0 {
			r += " e " + unidades[n%10]
		}
		return r
	case n == 100:
		return "cem"
	case n < 1000:
		r := centenas[n/100]
		if n%100 != 0 {
			r += " e " + PorExtenso(n%100)
		}
		return r
	case n < 1000000:
		milhar := n / 1000
		resto := n % 1000
		r := "mil"
		if milhar > 1 {
			r = PorExtenso(milhar) + " mil"
		}
		if resto != 0 {
			if resto < 100 {
				r += " e " + PorExtenso(resto)
			} else {
				r += " " + PorExtenso(resto)
			}
		}
		return r
	default:
		return fmt.Sprintf("%d", n)
	}
}
