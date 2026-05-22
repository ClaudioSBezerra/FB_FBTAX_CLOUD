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
		       COALESCE(cl.fone,''), COALESCE(cl.municipio,''), COALESCE(cl.uf,'')
		FROM financeiro.contratos c
		JOIN financeiro.clientes cl ON cl.id = c.cliente_id
		WHERE c.id = $1`, id,
	).Scan(
		&d.ID, &d.Numero, &d.DataInicio, &d.Periodicidade,
		&d.ValorTotal, &d.Status, &obs, &d.CriadoEm,
		&assinadoEmTime, &assinadoNome,
		&d.Cliente.RazaoSocial, &d.Cliente.CNPJ, &d.Cliente.Email,
		&d.Cliente.Fone, &d.Cliente.Municipio, &d.Cliente.UF,
	)
	if err != nil {
		return nil, fmt.Errorf("contrato não encontrado")
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

func gerarContratoPDF(d *ContratoDetalhe) ([]byte, error) {
	cfg := config.NewBuilder().
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(15).
		Build()

	mrt := maroto.New(cfg)

	addRows := func(rs ...core.Row) { mrt.AddRows(rs...) }

	txtBold := func(content string, size float64) core.Row {
		return row.New(7).Add(col.New(12).Add(
			text.New(content, props.Text{Size: size, Style: fontstyle.Bold, Align: align.Left}),
		))
	}
	txtNorm := func(content string, size float64) core.Row {
		return row.New(6).Add(col.New(12).Add(
			text.New(content, props.Text{Size: size, Align: align.Left}),
		))
	}
	espacamento := func(h float64) core.Row { return row.New(h) }
	separador := func() core.Row { return line.NewRow(3) }

	// ── Cabeçalho ──
	addRows(
		row.New(12).Add(
			col.New(8).Add(text.New("FORTES BEZERRA", props.Text{Size: 18, Style: fontstyle.Bold, Align: align.Left})),
			col.New(4).Add(text.New(fmt.Sprintf("Contrato Nº %s", d.Numero), props.Text{Size: 10, Align: align.Right})),
		),
		txtNorm("FB Tax Cloud — Soluções Fiscais e Financeiras", 9),
		txtNorm(fmt.Sprintf("Emitido em %s", time.Now().Format("02 de January de 2006")), 9),
		separador(),
		espacamento(3),
	)

	// ── Contratante ──
	addRows(
		txtBold("CONTRATANTE", 10),
		espacamento(2),
	)
	if d.Empresa.RazaoSocial != "" {
		addRows(txtNorm(d.Empresa.RazaoSocial, 9))
		if d.Empresa.CNPJ != "" {
			addRows(txtNorm(fmt.Sprintf("CNPJ: %s", d.Empresa.CNPJ), 9))
		}
		if d.Empresa.Logradouro != "" {
			addRows(txtNorm(fmt.Sprintf("%s, %s — %s/%s", d.Empresa.Logradouro, d.Empresa.Numero, d.Empresa.Municipio, d.Empresa.UF), 9))
		}
	} else {
		addRows(txtNorm("Fortes Bezerra Soluções Fiscais", 9))
	}
	addRows(espacamento(4), separador(), espacamento(3))

	// ── Contratado ──
	addRows(txtBold("CONTRATADO", 10), espacamento(2))
	addRows(txtNorm(d.Cliente.RazaoSocial, 9))
	if d.Cliente.CNPJ != "" {
		addRows(txtNorm(fmt.Sprintf("CNPJ: %s", d.Cliente.CNPJ), 9))
	}
	if d.Cliente.Municipio != "" {
		addRows(txtNorm(fmt.Sprintf("%s/%s", d.Cliente.Municipio, d.Cliente.UF), 9))
	}
	if d.Cliente.Email != "" {
		addRows(txtNorm(fmt.Sprintf("E-mail: %s", d.Cliente.Email), 9))
	}
	if d.Cliente.Fone != "" {
		addRows(txtNorm(fmt.Sprintf("Fone: %s", d.Cliente.Fone), 9))
	}
	addRows(espacamento(4), separador(), espacamento(3))

	// ── CNPJs cobertos ──
	if len(d.CNPJs) > 0 {
		addRows(txtBold("CNPJs COBERTOS PELO CONTRATO", 10), espacamento(2))
		for _, c := range d.CNPJs {
			label := c.CNPJ
			if c.Principal {
				label += " (Principal)"
			} else if c.Descricao != "" {
				label += fmt.Sprintf(" — %s", c.Descricao)
			}
			addRows(txtNorm("• "+label, 9))
		}
		addRows(espacamento(4), separador(), espacamento(3))
	}

	// ── Produtos e planos ──
	addRows(txtBold("PRODUTOS E SERVIÇOS CONTRATADOS", 10), espacamento(2))
	addRows(
		row.New(7).Add(
			col.New(5).Add(text.New("Produto", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(4).Add(text.New("Plano", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Valor", props.Text{Size: 9, Style: fontstyle.Bold, Align: align.Right})),
		),
	)
	addRows(separador())
	for _, item := range d.Itens {
		valorStr := "—"
		if item.ValorItem != nil {
			valorStr = fmt.Sprintf("R$ %.2f", *item.ValorItem)
		}
		addRows(row.New(6).Add(
			col.New(5).Add(text.New(item.Produto, props.Text{Size: 9})),
			col.New(4).Add(text.New(item.Plano, props.Text{Size: 9})),
			col.New(3).Add(text.New(valorStr, props.Text{Size: 9, Align: align.Right})),
		))
	}
	addRows(separador(), espacamento(3))

	// ── Condições financeiras ──
	addRows(txtBold("CONDIÇÕES FINANCEIRAS", 10), espacamento(2))
	addRows(
		row.New(6).Add(
			col.New(4).Add(text.New("Valor total:", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(8).Add(text.New(fmt.Sprintf("R$ %.2f", d.ValorTotal), props.Text{Size: 9})),
		),
		row.New(6).Add(
			col.New(4).Add(text.New("Periodicidade:", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(8).Add(text.New(capitalizar(d.Periodicidade), props.Text{Size: 9})),
		),
		row.New(6).Add(
			col.New(4).Add(text.New("Data de início:", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(8).Add(text.New(d.DataInicio, props.Text{Size: 9})),
		),
		row.New(6).Add(
			col.New(4).Add(text.New("Status:", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(8).Add(text.New(capitalizar(d.Status), props.Text{Size: 9})),
		),
	)
	if d.Observacoes != "" {
		addRows(
			espacamento(2),
			row.New(6).Add(
				col.New(4).Add(text.New("Observações:", props.Text{Size: 9, Style: fontstyle.Bold})),
				col.New(8).Add(text.New(d.Observacoes, props.Text{Size: 9})),
			),
		)
	}
	addRows(espacamento(6), separador(), espacamento(8))

	// ── Assinaturas ──
	addRows(txtBold("ASSINATURAS", 10), espacamento(6))
	addRows(
		row.New(6).Add(
			col.New(5).Add(text.New("_________________________________", props.Text{Size: 9, Align: align.Center})),
			col.New(2),
			col.New(5).Add(text.New("_________________________________", props.Text{Size: 9, Align: align.Center})),
		),
		row.New(5).Add(
			col.New(5).Add(text.New("CONTRATANTE", props.Text{Size: 8, Align: align.Center})),
			col.New(2),
			col.New(5).Add(text.New("CONTRATADO", props.Text{Size: 8, Align: align.Center})),
		),
		row.New(5).Add(
			col.New(5).Add(text.New(d.Empresa.RazaoSocial, props.Text{Size: 8, Align: align.Center, Style: fontstyle.Italic})),
			col.New(2),
			col.New(5).Add(text.New(d.Cliente.RazaoSocial, props.Text{Size: 8, Align: align.Center, Style: fontstyle.Italic})),
		),
		espacamento(8),
		row.New(5).Add(
			col.New(5).Add(text.New("_________________________________", props.Text{Size: 9, Align: align.Center})),
			col.New(2),
			col.New(5).Add(text.New("_________________________________", props.Text{Size: 9, Align: align.Center})),
		),
		row.New(5).Add(
			col.New(5).Add(text.New("Testemunha 1", props.Text{Size: 8, Align: align.Center})),
			col.New(2),
			col.New(5).Add(text.New("Testemunha 2", props.Text{Size: 8, Align: align.Center})),
		),
	)

	// ── Rodapé ──
	addRows(
		espacamento(8),
		line.NewRow(2),
		row.New(5).Add(col.New(12).Add(text.New(
			fmt.Sprintf("Gerado em %s · %s", time.Now().Format("02/01/2006 15:04"), d.Numero),
			props.Text{Size: 7, Align: align.Center},
		))),
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
