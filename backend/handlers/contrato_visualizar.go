package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

// ── Estruturas de detalhe do contrato ────────────────────────────────────────

type ContratoDetalhe struct {
	ID            string             `json:"id"`
	Numero        string             `json:"numero"`
	DataInicio    string             `json:"data_inicio"`
	Periodicidade string             `json:"periodicidade"`
	ValorTotal    float64            `json:"valor_total"`
	Status        string             `json:"status"`
	Observacoes   string             `json:"observacoes"`
	CriadoEm      string             `json:"criado_em"`
	AssinadoEm    *string            `json:"assinado_em"`
	AssinadoNome  *string            `json:"assinado_nome"`
	Cliente       ContratoCliente    `json:"cliente"`
	Empresa       ContratoEmpresa    `json:"empresa"`
	CNPJs         []ContratoCNPJ     `json:"cnpjs"`
	Itens         []ContratoItem     `json:"itens"`
}

type ContratoCliente struct {
	RazaoSocial string `json:"razao_social"`
	CNPJ        string `json:"cnpj"`
	Email       string `json:"email"`
	Fone        string `json:"fone"`
	Municipio   string `json:"municipio"`
	UF          string `json:"uf"`
	Responsavel string `json:"responsavel"`
	Logradouro  string `json:"logradouro"`
	NumeroEnd   string `json:"numero_end"`
}

type ContratoEmpresa struct {
	RazaoSocial   string `json:"razao_social"`
	NomeFantasia  string `json:"nome_fantasia"`
	CNPJ          string `json:"cnpj"`
	Logradouro    string `json:"logradouro"`
	Numero        string `json:"numero"`
	Municipio     string `json:"municipio"`
	UF            string `json:"uf"`
}

type ContratoCNPJ struct {
	CNPJ      string `json:"cnpj"`
	Descricao string `json:"descricao"`
	Principal bool   `json:"principal"`
}

type ContratoItem struct {
	Produto   string   `json:"produto"`
	Plano     string   `json:"plano"`
	ValorItem *float64 `json:"valor_item"`
}

// ── GET /api/financeiro/contratos/detalhe?id=xxx ─────────────────────────────

func ContratoDetalheHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id obrigatório", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		det, err := carregarDetalheContrato(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(det)
	}
}

func carregarDetalheContrato(db *sql.DB, id string) (*ContratoDetalhe, error) {
	var d ContratoDetalhe
	var obs, assinadoNome sql.NullString
	var assinadoEmTime sql.NullTime

	err := db.QueryRow(`
		SELECT c.id, COALESCE(c.numero,''), c.data_inicio, c.periodicidade,
		       c.valor_total, c.status, COALESCE(c.observacoes,''), c.created_at,
		       c.assinado_em, c.assinado_nome,
		       cl.razao_social, COALESCE(cl.cnpj,''), COALESCE(cl.email,''),
		       COALESCE(cl.telefone,''), COALESCE(cl.municipio,''), COALESCE(cl.uf,''),
		       COALESCE(cl.responsavel,''), COALESCE(cl.logradouro,''), COALESCE(cl.numero,'')
		FROM financeiro.contratos c
		JOIN financeiro.clientes cl ON cl.id = c.cliente_id
		WHERE c.id = $1`, id,
	).Scan(
		&d.ID, &d.Numero, &d.DataInicio, &d.Periodicidade,
		&d.ValorTotal, &d.Status, &obs, &d.CriadoEm,
		&assinadoEmTime, &assinadoNome,
		&d.Cliente.RazaoSocial, &d.Cliente.CNPJ, &d.Cliente.Email,
		&d.Cliente.Fone, &d.Cliente.Municipio, &d.Cliente.UF,
		&d.Cliente.Responsavel, &d.Cliente.Logradouro, &d.Cliente.NumeroEnd,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("contrato não encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar contrato: %w", err)
	}
	d.Observacoes = obs.String
	if assinadoEmTime.Valid {
		s := assinadoEmTime.Time.Format("02/01/2006 15:04")
		d.AssinadoEm = &s
	}
	if assinadoNome.Valid {
		d.AssinadoNome = &assinadoNome.String
	}

	// Empresa contratante
	db.QueryRow(`
		SELECT COALESCE(razao_social,''), COALESCE(nome_fantasia,''), COALESCE(cnpj,''),
		       COALESCE(logradouro,''), COALESCE(numero,''), COALESCE(municipio,''), COALESCE(uf,'')
		FROM financeiro.empresas LIMIT 1`,
	).Scan(
		&d.Empresa.RazaoSocial, &d.Empresa.NomeFantasia, &d.Empresa.CNPJ,
		&d.Empresa.Logradouro, &d.Empresa.Numero, &d.Empresa.Municipio, &d.Empresa.UF,
	)

	// CNPJs
	cnpjRows, _ := db.Query(`
		SELECT cc.cnpj, COALESCE(cc.descricao,''), cc.is_principal
		FROM financeiro.contrato_cnpjs ccj
		JOIN financeiro.cliente_cnpjs cc ON cc.id = ccj.cnpj_id
		WHERE ccj.contrato_id = $1
		ORDER BY cc.is_principal DESC, cc.cnpj`, id)
	if cnpjRows != nil {
		defer cnpjRows.Close()
		for cnpjRows.Next() {
			var c ContratoCNPJ
			cnpjRows.Scan(&c.CNPJ, &c.Descricao, &c.Principal)
			d.CNPJs = append(d.CNPJs, c)
		}
	}

	// Itens
	itemRows, _ := db.Query(`
		SELECT p.nome AS produto, pl.nome AS plano, ci.valor_item
		FROM financeiro.contrato_itens ci
		JOIN financeiro.planos pl ON pl.id = ci.plano_id
		JOIN financeiro.produtos p ON p.id = pl.produto_id
		WHERE ci.contrato_id = $1
		ORDER BY p.nome, pl.nome`, id)
	if itemRows != nil {
		defer itemRows.Close()
		for itemRows.Next() {
			var item ContratoItem
			var vi sql.NullFloat64
			itemRows.Scan(&item.Produto, &item.Plano, &vi)
			if vi.Valid {
				item.ValorItem = &vi.Float64
			}
			d.Itens = append(d.Itens, item)
		}
	}

	return &d, nil
}

// ── GET /api/financeiro/contratos/pdf?id=xxx ─────────────────────────────────

func ContratoPDFHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id obrigatório", http.StatusBadRequest)
			return
		}

		det, err := carregarDetalheContrato(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		pdfBytes, err := gerarContratoPDF(det)
		if err != nil {
			http.Error(w, "erro gerando PDF: "+err.Error(), http.StatusInternalServerError)
			return
		}

		nome := fmt.Sprintf("Contrato_%s.pdf", det.Numero)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, nome))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
		w.Write(pdfBytes)
	}
}

func formatarCNPJ(cnpj string) string {
	d := []rune{}
	for _, c := range cnpj {
		if c >= '0' && c <= '9' {
			d = append(d, c)
		}
	}
	if len(d) == 14 {
		return fmt.Sprintf("%s.%s.%s/%s-%s",
			string(d[0:2]), string(d[2:5]), string(d[5:8]), string(d[8:12]), string(d[12:14]))
	}
	return cnpj
}

func formatarData(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	meses := []string{"", "janeiro", "fevereiro", "março", "abril", "maio", "junho",
		"julho", "agosto", "setembro", "outubro", "novembro", "dezembro"}
	return fmt.Sprintf("%d de %s de %d", t.Day(), meses[int(t.Month())], t.Year())
}

func gerarContratoPDF(d *ContratoDetalhe) ([]byte, error) {
	cfg := config.NewBuilder().
		WithLeftMargin(18).
		WithRightMargin(18).
		WithTopMargin(12).
		Build()

	mrt := maroto.New(cfg)
	add := func(rs ...core.Row) { mrt.AddRows(rs...) }

	// ── helpers ────────────────────────────────────────────────────────────────
	esp := func(h float64) core.Row { return row.New(h) }
	sep := func() core.Row { return line.NewRow(2) }

	titulo := func(s string) core.Row {
		return row.New(9).Add(col.New(12).Add(
			text.New(s, props.Text{Size: 11, Style: fontstyle.Bold, Align: align.Center}),
		))
	}
	clausulaTitulo := func(s string) core.Row {
		return row.New(7).Add(col.New(12).Add(
			text.New(s, props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Left}),
		))
	}
	para := func(s string, h float64) core.Row {
		return row.New(h).Add(col.New(12).Add(
			text.New(s, props.Text{Size: 8.5, Align: align.Left}),
		))
	}
	campo := func(label, valor string) core.Row {
		return row.New(5.5).Add(
			col.New(4).Add(text.New(label, props.Text{Size: 8.5, Style: fontstyle.Bold})),
			col.New(8).Add(text.New(valor, props.Text{Size: 8.5})),
		)
	}
	rodapeTexto := func(s string) core.Row {
		return row.New(5).Add(col.New(12).Add(
			text.New(s, props.Text{Size: 7.5, Align: align.Center}),
		))
	}

	// ── dados da empresa (fallback) ───────────────────────────────────────────
	nomeEmpresa := d.Empresa.RazaoSocial
	if nomeEmpresa == "" {
		nomeEmpresa = "FORTES BEZERRA TECNOLOGIA LTDA"
	}
	cnpjEmpresa := formatarCNPJ(d.Empresa.CNPJ)
	if cnpjEmpresa == "" {
		cnpjEmpresa = "38.149.716/0001-28"
	}
	endEmpresa := ""
	if d.Empresa.Logradouro != "" {
		endEmpresa = fmt.Sprintf("%s, nº %s, %s/%s", d.Empresa.Logradouro, d.Empresa.Numero, d.Empresa.Municipio, d.Empresa.UF)
	} else {
		endEmpresa = "Aparecida de Goiania - GO"
	}
	municipioEmpresa := d.Empresa.Municipio
	if municipioEmpresa == "" {
		municipioEmpresa = "Aparecida de Goiania"
	}
	ufEmpresa := d.Empresa.UF
	if ufEmpresa == "" {
		ufEmpresa = "GO"
	}

	// ── dados do cliente ──────────────────────────────────────────────────────
	cnpjCliente := formatarCNPJ(d.Cliente.CNPJ)
	endCliente := ""
	if d.Cliente.Logradouro != "" {
		endCliente = d.Cliente.Logradouro
		if d.Cliente.NumeroEnd != "" {
			endCliente += ", nº " + d.Cliente.NumeroEnd
		}
		if d.Cliente.Municipio != "" {
			endCliente += fmt.Sprintf(", %s/%s", d.Cliente.Municipio, d.Cliente.UF)
		}
	} else if d.Cliente.Municipio != "" {
		endCliente = fmt.Sprintf("%s/%s", d.Cliente.Municipio, d.Cliente.UF)
	}

	// ── CABEÇALHO ─────────────────────────────────────────────────────────────
	add(
		row.New(11).Add(
			col.New(8).Add(text.New(nomeEmpresa, props.Text{Size: 14, Style: fontstyle.Bold, Align: align.Left})),
			col.New(4).Add(text.New(fmt.Sprintf("Nº %s", d.Numero), props.Text{Size: 9, Align: align.Right, Style: fontstyle.Bold})),
		),
		row.New(5).Add(col.New(12).Add(
			text.New("FBTax Cloud — Soluções Fiscais e Tributárias", props.Text{Size: 8, Align: align.Left}),
		)),
		esp(3),
		sep(),
		esp(4),
	)

	// ── TÍTULO ────────────────────────────────────────────────────────────────
	add(
		titulo("CONTRATO DE PRESTAÇÃO DE SERVIÇOS DE TECNOLOGIA"),
		esp(6),
	)

	// ── PREÂMBULO ─────────────────────────────────────────────────────────────
	add(
		para(fmt.Sprintf(
			"%s, pessoa jurídica de direito privado, inscrita no CNPJ sob o nº %s, com sede na %s, doravante denominada simplesmente CONTRATANTE;",
			nomeEmpresa, cnpjEmpresa, endEmpresa,
		), 18),
		esp(2),
		para("E", 4),
		esp(2),
	)

	clienteDesc := d.Cliente.RazaoSocial
	if cnpjCliente != "" {
		clienteDesc += fmt.Sprintf(", pessoa jurídica de direito privado, inscrita no CNPJ sob o nº %s", cnpjCliente)
	}
	if endCliente != "" {
		clienteDesc += fmt.Sprintf(", com sede na %s", endCliente)
	}
	clienteDesc += ", doravante denominada simplesmente CONTRATADA;"

	add(
		para(clienteDesc, 22),
		esp(3),
		para("As partes acima qualificadas, em conjunto denominadas PARTES, têm entre si justo e acordado o presente Contrato de Prestação de Serviços de Tecnologia, que se regerá pelas cláusulas e condições a seguir estipuladas.", 14),
		esp(6),
		sep(),
		esp(5),
	)

	// ── CLÁUSULA 1ª ───────────────────────────────────────────────────────────
	add(
		clausulaTitulo("CLÁUSULA 1ª – DO OBJETO"),
		esp(2),
		para("O presente Contrato tem por objeto a prestação de serviços de tecnologia pela CONTRATANTE à CONTRATADA, consistindo no acesso e utilização dos sistemas e plataformas digitais integrantes do ecossistema FBTax Cloud e Portal Fortes Bezerra, incluindo as soluções de gestão fiscal, tributária e financeira, nos termos e condições estabelecidos neste instrumento.", 22),
		esp(5),
	)

	// ── CLÁUSULA 2ª ───────────────────────────────────────────────────────────
	add(
		clausulaTitulo("CLÁUSULA 2ª – DOS SERVIÇOS CONTRATADOS"),
		esp(2),
		para("Os serviços objeto deste Contrato compreendem o acesso às seguintes soluções e respectivos planos:", 10),
		esp(2),
	)

	// tabela de produtos
	add(row.New(6).Add(
		col.New(5).Add(text.New("Produto / Solução", props.Text{Size: 8.5, Style: fontstyle.Bold})),
		col.New(4).Add(text.New("Plano", props.Text{Size: 8.5, Style: fontstyle.Bold})),
		col.New(3).Add(text.New("Valor Mensal (R$)", props.Text{Size: 8.5, Style: fontstyle.Bold, Align: align.Right})),
	))
	add(line.NewRow(1))
	for _, item := range d.Itens {
		valorStr := "Sob consulta"
		if item.ValorItem != nil {
			valorStr = fmt.Sprintf("%.2f", *item.ValorItem)
		}
		add(row.New(5.5).Add(
			col.New(5).Add(text.New(item.Produto, props.Text{Size: 8.5})),
			col.New(4).Add(text.New(item.Plano, props.Text{Size: 8.5})),
			col.New(3).Add(text.New(valorStr, props.Text{Size: 8.5, Align: align.Right})),
		))
	}
	add(line.NewRow(1), esp(5))

	if len(d.CNPJs) > 0 {
		add(para("Parágrafo único. Os serviços acima abrangem os seguintes CNPJs da CONTRATADA:", 8), esp(2))
		for _, c := range d.CNPJs {
			label := "• " + formatarCNPJ(c.CNPJ)
			if c.Principal {
				label += " (Estabelecimento Principal)"
			} else if c.Descricao != "" {
				label += " — " + c.Descricao
			}
			add(para(label, 5.5))
		}
		add(esp(5))
	}

	// ── CLÁUSULA 3ª ───────────────────────────────────────────────────────────
	add(
		clausulaTitulo("CLÁUSULA 3ª – DO VALOR E DA FORMA DE PAGAMENTO"),
		esp(2),
		campo("Valor total:", fmt.Sprintf("R$ %.2f (%s)", d.ValorTotal, valorPorExtenso(d.ValorTotal))),
		campo("Periodicidade:", capitalizar(d.Periodicidade)),
		campo("Vigência a partir de:", formatarData(d.DataInicio)),
		esp(2),
		para("3.1. O pagamento deverá ser realizado na data de vencimento acordada entre as PARTES, mediante boleto bancário, transferência bancária (TED/PIX) ou outra forma previamente convencionada.", 14),
		esp(2),
		para("3.2. O inadimplemento por prazo superior a 15 (quinze) dias corridos implicará a suspensão automática do acesso aos serviços contratados, sem prejuízo da cobrança de multa de 2% (dois por cento) sobre o valor em atraso, acrescida de juros de mora de 1% (um por cento) ao mês e correção monetária pelo IGPM/FGV.", 22),
		esp(5),
	)

	// ── CLÁUSULA 4ª ───────────────────────────────────────────────────────────
	add(
		clausulaTitulo("CLÁUSULA 4ª – DAS OBRIGAÇÕES DAS PARTES"),
		esp(2),
		para("4.1. Compete à CONTRATANTE: (i) disponibilizar o acesso às plataformas contratadas em ambiente operacional; (ii) prestar suporte técnico nos termos do plano contratado; (iii) manter os sistemas atualizados conforme a legislação fiscal e tributária vigente; (iv) garantir a segurança e disponibilidade dos dados no ambiente de nuvem.", 22),
		esp(2),
		para("4.2. Compete à CONTRATADA: (i) efetuar os pagamentos nas datas acordadas; (ii) utilizar os serviços de forma lícita e de acordo com a legislação vigente; (iii) manter atualizados seus dados cadastrais junto à CONTRATANTE; (iv) não ceder, sublicenciar ou compartilhar credenciais de acesso a terceiros.", 22),
		esp(5),
	)

	// ── CLÁUSULA 5ª ───────────────────────────────────────────────────────────
	add(
		clausulaTitulo("CLÁUSULA 5ª – DA VIGÊNCIA E DA RESCISÃO"),
		esp(2),
		para(fmt.Sprintf("5.1. O presente Contrato entra em vigor na data de sua assinatura, com início da prestação dos serviços em %s, e permanecerá em vigor por prazo indeterminado, renovando-se automaticamente a cada período de cobrança.", formatarData(d.DataInicio)), 16),
		esp(2),
		para("5.2. Qualquer das PARTES poderá rescindir o presente Contrato mediante notificação prévia por escrito com antecedência mínima de 30 (trinta) dias, sem a incidência de multas rescisórias, desde que não haja débitos pendentes.", 16),
		esp(2),
		para("5.3. A rescisão imotivada pela CONTRATADA dentro dos primeiros 12 (doze) meses de vigência implicará o pagamento de multa equivalente a 2 (duas) mensalidades, a título de compensação pelos investimentos realizados pela CONTRATANTE.", 16),
		esp(5),
	)

	// ── CLÁUSULA 6ª ───────────────────────────────────────────────────────────
	add(
		clausulaTitulo("CLÁUSULA 6ª – DA PROPRIEDADE INTELECTUAL E CONFIDENCIALIDADE"),
		esp(2),
		para("6.1. Todos os sistemas, softwares, algoritmos, bases de dados, interfaces, documentações e demais ativos tecnológicos disponibilizados pela CONTRATANTE são de sua exclusiva propriedade intelectual, protegidos pela Lei nº 9.279/1996 e pela Lei nº 9.609/1998 (Lei do Software), sendo vedada qualquer reprodução, cópia, engenharia reversa ou uso não autorizado.", 22),
		esp(2),
		para("6.2. As PARTES se comprometem a manter sigilo sobre todas as informações técnicas, comerciais e operacionais a que tiverem acesso em virtude deste Contrato, durante sua vigência e pelo prazo de 5 (cinco) anos após seu término.", 16),
		esp(5),
	)

	// ── CLÁUSULA 7ª ───────────────────────────────────────────────────────────
	add(
		clausulaTitulo("CLÁUSULA 7ª – DA PROTEÇÃO DE DADOS PESSOAIS (LGPD)"),
		esp(2),
		para("As PARTES declaram estar cientes e em conformidade com a Lei Geral de Proteção de Dados Pessoais — LGPD (Lei nº 13.709/2018). A CONTRATANTE atuará como Operadora dos dados pessoais eventualmente processados nos sistemas em nome da CONTRATADA (Controladora), comprometendo-se a adotar medidas técnicas e administrativas adequadas para garantir a segurança, confidencialidade e integridade das informações tratadas.", 28),
		esp(5),
	)

	// ── CLÁUSULA 8ª ───────────────────────────────────────────────────────────
	add(
		clausulaTitulo("CLÁUSULA 8ª – DO FORO"),
		esp(2),
		para(fmt.Sprintf("Fica eleito o foro da Comarca de %s, Estado de %s, com exclusão de qualquer outro, por mais privilegiado que seja, para dirimir quaisquer litígios decorrentes deste Contrato.", municipioEmpresa, ufEmpresa), 12),
		esp(5),
		sep(),
		esp(5),
	)

	if d.Observacoes != "" {
		add(
			clausulaTitulo("DISPOSIÇÕES ADICIONAIS"),
			esp(2),
			para(d.Observacoes, 14),
			esp(5),
			sep(),
			esp(5),
		)
	}

	// ── ENCERRAMENTO ──────────────────────────────────────────────────────────
	cidade := municipioEmpresa + " – " + ufEmpresa
	add(
		para(fmt.Sprintf("E, por estarem assim justas e acordadas, as PARTES assinam o presente Contrato em 2 (duas) vias de igual teor e forma, na presença das testemunhas abaixo identificadas.", 14), 12),
		esp(4),
		para(fmt.Sprintf("%s, _____ de ________________ de ______.", cidade), 6),
		esp(10),
		sep(),
		esp(10),
	)

	// ── ASSINATURAS ───────────────────────────────────────────────────────────
	add(
		row.New(6).Add(
			col.New(5).Add(text.New("_________________________________", props.Text{Size: 9, Align: align.Center})),
			col.New(2),
			col.New(5).Add(text.New("_________________________________", props.Text{Size: 9, Align: align.Center})),
		),
		row.New(5).Add(
			col.New(5).Add(text.New("CONTRATANTE", props.Text{Size: 8.5, Style: fontstyle.Bold, Align: align.Center})),
			col.New(2),
			col.New(5).Add(text.New("CONTRATADA", props.Text{Size: 8.5, Style: fontstyle.Bold, Align: align.Center})),
		),
		row.New(5).Add(
			col.New(5).Add(text.New(nomeEmpresa, props.Text{Size: 8, Align: align.Center, Style: fontstyle.Italic})),
			col.New(2),
			col.New(5).Add(text.New(d.Cliente.RazaoSocial, props.Text{Size: 8, Align: align.Center, Style: fontstyle.Italic})),
		),
		row.New(5).Add(
			col.New(5).Add(text.New("CNPJ: "+cnpjEmpresa, props.Text{Size: 7.5, Align: align.Center})),
			col.New(2),
			col.New(5).Add(text.New("CNPJ: "+cnpjCliente, props.Text{Size: 7.5, Align: align.Center})),
		),
		esp(10),
	)

	// ── TESTEMUNHAS ───────────────────────────────────────────────────────────
	add(
		para("TESTEMUNHAS:", 6),
		esp(8),
		row.New(5).Add(
			col.New(5).Add(text.New("1. _________________________________", props.Text{Size: 9})),
			col.New(2),
			col.New(5).Add(text.New("2. _________________________________", props.Text{Size: 9})),
		),
		row.New(5).Add(
			col.New(5).Add(text.New("   Nome: ___________________________", props.Text{Size: 8.5})),
			col.New(2),
			col.New(5).Add(text.New("   Nome: ___________________________", props.Text{Size: 8.5})),
		),
		row.New(5).Add(
			col.New(5).Add(text.New("   CPF:  ___________________________", props.Text{Size: 8.5})),
			col.New(2),
			col.New(5).Add(text.New("   CPF:  ___________________________", props.Text{Size: 8.5})),
		),
		esp(8),
		sep(),
	)

	// ── RODAPÉ ────────────────────────────────────────────────────────────────
	add(
		esp(2),
		rodapeTexto(fmt.Sprintf("Gerado automaticamente em %s  ·  %s  ·  FBTax Cloud", time.Now().Format("02/01/2006 às 15:04"), d.Numero)),
	)

	doc, err := mrt.Generate()
	if err != nil {
		return nil, err
	}
	return doc.GetBytes(), nil
}

func capitalizar(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

// ── POST /api/financeiro/contratos/upload-assinado ───────────────────────────

func ContratoUploadAssinadoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		// Limita a 20MB
		r.ParseMultipartForm(20 << 20)

		contratoID := r.FormValue("contrato_id")
		if contratoID == "" {
			http.Error(w, "contrato_id obrigatório", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("arquivo")
		if err != nil {
			http.Error(w, "arquivo obrigatório", http.StatusBadRequest)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "erro lendo arquivo", http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(`
			UPDATE financeiro.contratos
			SET assinado_data = $1, assinado_nome = $2, assinado_em = NOW()
			WHERE id = $3`,
			data, header.Filename, contratoID,
		)
		if err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"mensagem": "Contrato assinado enviado com sucesso"})
	}
}

// ── GET /api/financeiro/contratos/download-assinado?id=xxx ──────────────────

func ContratoDownloadAssinadoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id obrigatório", http.StatusBadRequest)
			return
		}

		var data []byte
		var nome sql.NullString
		err := db.QueryRow(`SELECT assinado_data, assinado_nome FROM financeiro.contratos WHERE id = $1`, id).
			Scan(&data, &nome)
		if err != nil || data == nil {
			http.Error(w, "arquivo não encontrado", http.StatusNotFound)
			return
		}

		filename := "contrato_assinado.pdf"
		if nome.Valid && nome.String != "" {
			filename = nome.String
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	}
}

func valorPorExtenso(v float64) string {
	inteiro := int64(v)
	centavos := int64((v - float64(inteiro)) * 100)

	unidades := []string{"", "um", "dois", "três", "quatro", "cinco", "seis", "sete", "oito", "nove",
		"dez", "onze", "doze", "treze", "quatorze", "quinze", "dezesseis", "dezessete", "dezoito", "dezenove"}
	dezenas := []string{"", "", "vinte", "trinta", "quarenta", "cinquenta", "sessenta", "setenta", "oitenta", "noventa"}
	centenas := []string{"", "cem", "duzentos", "trezentos", "quatrocentos", "quinhentos",
		"seiscentos", "setecentos", "oitocentos", "novecentos"}

	var escrever func(n int64) string
	escrever = func(n int64) string {
		switch {
		case n == 0:
			return "zero"
		case n < 20:
			return unidades[n]
		case n < 100:
			r := dezenas[n/10]
			if n%10 != 0 {
				r += " e " + unidades[n%10]
			}
			return r
		case n < 1000:
			if n == 100 {
				return "cem"
			}
			r := centenas[n/100]
			if n%100 != 0 {
				r += " e " + escrever(n%100)
			}
			return r
		case n < 1000000:
			mil := n / 1000
			resto := n % 1000
			r := escrever(mil) + " mil"
			if resto != 0 {
				r += " e " + escrever(resto)
			}
			return r
		default:
			return fmt.Sprintf("%d", n)
		}
	}

	resultado := escrever(inteiro) + " reais"
	if centavos > 0 {
		resultado += " e " + escrever(centavos) + " centavos"
	}
	return resultado
}
