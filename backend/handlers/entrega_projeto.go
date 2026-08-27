package handlers

// entrega_projeto.go — Termo de Entrega e Aceite de Projeto.
//
// Documento assinado pelo cliente ao final da implantação de um módulo. É
// gerado no Padrão de Documentos FBTECH (services/fbdoc.go) e todo campo que o
// usuário deixar em branco na tela é impresso como linha pautada, para
// preenchimento à mão no ato da assinatura.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fb_cloud/services"
)

// ── Modelo ───────────────────────────────────────────────────────────────────

type EntregaItem struct {
	ID        string `json:"id,omitempty"`
	Ordem     int    `json:"ordem"`
	Descricao string `json:"descricao"`
	Detalhe   string `json:"detalhe"`
	Situacao  string `json:"situacao"`
}

type EntregaPendencia struct {
	ID          string `json:"id,omitempty"`
	Ordem       int    `json:"ordem"`
	Descricao   string `json:"descricao"`
	Responsavel string `json:"responsavel"`
	Prazo       string `json:"prazo"`
}

type EntregaSignatario struct {
	ID          string `json:"id,omitempty"`
	Ordem       int    `json:"ordem"`
	Papel       string `json:"papel"`
	Nome        string `json:"nome"`
	Cargo       string `json:"cargo"`
	Organizacao string `json:"organizacao"`
	Documento   string `json:"documento"`
}

type EntregaProjeto struct {
	ID           string `json:"id,omitempty"`
	Numero       string `json:"numero"`
	ClienteID    string `json:"cliente_id"`
	ContratoID   string `json:"contrato_id"`
	ProdutoID    string `json:"produto_id"`
	Versao       string `json:"versao"`
	Titulo       string `json:"titulo"`
	Projeto      string `json:"projeto"`
	EscopoResumo string `json:"escopo_resumo"`

	DataInicio    string `json:"data_inicio"`
	DataGolive    string `json:"data_golive"`
	DataEntrega   string `json:"data_entrega"`
	GarantiaDias  int    `json:"garantia_dias"`
	AssistidaDias int    `json:"assistida_dias"`

	Ambiente             string `json:"ambiente"`
	ResponsavelTecnico   string `json:"responsavel_tecnico"`
	ResponsavelComercial string `json:"responsavel_comercial"`

	Observacoes  string `json:"observacoes"`
	LinhasLivres int    `json:"linhas_livres"`
	Status       string `json:"status"`

	// Preenchidos na leitura, para exibição.
	ClienteNome string `json:"cliente_nome,omitempty"`
	ClienteCNPJ string `json:"cliente_cnpj,omitempty"`
	ProdutoNome string `json:"produto_nome,omitempty"`
	ProdutoCod  string `json:"produto_codigo,omitempty"`
	AssinadoEm  string `json:"assinado_em,omitempty"`
	CriadoEm    string `json:"criado_em,omitempty"`

	Itens       []EntregaItem       `json:"itens"`
	Pendencias  []EntregaPendencia  `json:"pendencias"`
	Signatarios []EntregaSignatario `json:"signatarios"`
}

// ── GET/POST /api/financeiro/entregas ────────────────────────────────────────

func EntregasHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			listarEntregas(db, w, r)
		case http.MethodPost:
			criarEntrega(db, w, r)
		case http.MethodPut:
			atualizarEntrega(db, w, r)
		case http.MethodDelete:
			excluirEntrega(db, w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func listarEntregas(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT e.id, e.numero, COALESCE(e.projeto,''), e.data_entrega, e.status,
		       COALESCE(cl.razao_social,''), COALESCE(cl.cnpj,''),
		       COALESCE(p.nome,''), COALESCE(p.codigo,''),
		       e.assinado_em, e.created_at
		FROM financeiro.entregas_projeto e
		LEFT JOIN financeiro.clientes  cl ON cl.id = e.cliente_id
		LEFT JOIN financeiro.produtos  p  ON p.id  = e.produto_id
		ORDER BY e.created_at DESC`)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	lista := []EntregaProjeto{}
	for rows.Next() {
		var e EntregaProjeto
		var dataEntrega time.Time
		var criadoEm time.Time
		var assinadoEm sql.NullTime
		if err := rows.Scan(&e.ID, &e.Numero, &e.Projeto, &dataEntrega, &e.Status,
			&e.ClienteNome, &e.ClienteCNPJ, &e.ProdutoNome, &e.ProdutoCod,
			&assinadoEm, &criadoEm); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		e.DataEntrega = dataEntrega.Format("2006-01-02")
		e.CriadoEm = criadoEm.Format(time.RFC3339)
		if assinadoEm.Valid {
			e.AssinadoEm = assinadoEm.Time.Format(time.RFC3339)
		}
		lista = append(lista, e)
	}
	json.NewEncoder(w).Encode(lista)
}

// nulo converte string vazia em NULL para colunas opcionais.
func nulo(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// nuloData valida e converte uma data ISO opcional.
func nuloData(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return nil
	}
	return s
}

func criarEntrega(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var e EntregaProjeto
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(e.ClienteID) == "" {
		http.Error(w, "cliente_id obrigatório", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var numero string
	if err := tx.QueryRow(`
		SELECT 'TE-' || TO_CHAR(CURRENT_DATE,'YYYY') || '-' ||
		       LPAD(NEXTVAL('financeiro.entrega_seq')::TEXT, 3, '0')`).Scan(&numero); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	aplicarPadroes(&e)

	var id string
	err = tx.QueryRow(`
		INSERT INTO financeiro.entregas_projeto
		    (numero, cliente_id, contrato_id, produto_id, versao, titulo, projeto,
		     escopo_resumo, data_inicio, data_golive, data_entrega, garantia_dias,
		     assistida_dias, ambiente, responsavel_tecnico, responsavel_comercial,
		     observacoes, linhas_livres, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,COALESCE($11::date, CURRENT_DATE),
		        $12,$13,$14,$15,$16,$17,$18,$19)
		RETURNING id`,
		numero, e.ClienteID, nulo(e.ContratoID), nulo(e.ProdutoID), e.Versao, e.Titulo,
		nulo(e.Projeto), nulo(e.EscopoResumo), nuloData(e.DataInicio), nuloData(e.DataGolive),
		nuloData(e.DataEntrega), e.GarantiaDias, e.AssistidaDias, nulo(e.Ambiente),
		nulo(e.ResponsavelTecnico), nulo(e.ResponsavelComercial), nulo(e.Observacoes),
		e.LinhasLivres, e.Status,
	).Scan(&id)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := gravarFilhos(tx, id, &e); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	e.ID, e.Numero = id, numero
	json.NewEncoder(w).Encode(e)
}

func atualizarEntrega(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var e EntregaProjeto
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		http.Error(w, "json inválido", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(e.ID) == "" {
		http.Error(w, "id obrigatório", http.StatusBadRequest)
		return
	}
	aplicarPadroes(&e)

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE financeiro.entregas_projeto SET
		    cliente_id = $2, contrato_id = $3, produto_id = $4, versao = $5,
		    titulo = $6, projeto = $7, escopo_resumo = $8, data_inicio = $9,
		    data_golive = $10, data_entrega = COALESCE($11::date, data_entrega),
		    garantia_dias = $12, assistida_dias = $13, ambiente = $14,
		    responsavel_tecnico = $15, responsavel_comercial = $16,
		    observacoes = $17, linhas_livres = $18, status = $19,
		    updated_at = NOW()
		WHERE id = $1`,
		e.ID, nulo(e.ClienteID), nulo(e.ContratoID), nulo(e.ProdutoID), e.Versao,
		e.Titulo, nulo(e.Projeto), nulo(e.EscopoResumo), nuloData(e.DataInicio),
		nuloData(e.DataGolive), nuloData(e.DataEntrega), e.GarantiaDias,
		e.AssistidaDias, nulo(e.Ambiente), nulo(e.ResponsavelTecnico),
		nulo(e.ResponsavelComercial), nulo(e.Observacoes), e.LinhasLivres, e.Status,
	)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "termo não encontrado", http.StatusNotFound)
		return
	}

	// As linhas filhas são recriadas a cada gravação — a tela envia sempre o
	// conjunto completo de itens, pendências e signatários.
	for _, t := range []string{"entrega_itens", "entrega_pendencias", "entrega_signatarios"} {
		if _, err := tx.Exec("DELETE FROM financeiro."+t+" WHERE entrega_id = $1", e.ID); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := gravarFilhos(tx, e.ID, &e); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(e)
}

func excluirEntrega(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id obrigatório", http.StatusBadRequest)
		return
	}
	if _, err := db.Exec(`DELETE FROM financeiro.entregas_projeto WHERE id = $1`, id); err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"mensagem": "Termo excluído"})
}

// aplicarPadroes preenche os defaults do padrão FBTECH para os campos que a
// tela pode deixar em branco.
func aplicarPadroes(e *EntregaProjeto) {
	if strings.TrimSpace(e.Versao) == "" {
		e.Versao = "1.0"
	}
	if strings.TrimSpace(e.Titulo) == "" {
		e.Titulo = "Termo de Entrega e Aceite de Projeto"
	}
	if strings.TrimSpace(e.Status) == "" {
		e.Status = "rascunho"
	}
	if e.GarantiaDias <= 0 {
		e.GarantiaDias = 30
	}
	if e.AssistidaDias <= 0 {
		e.AssistidaDias = 14
	}
	if e.LinhasLivres < 0 {
		e.LinhasLivres = 0
	}
}

func gravarFilhos(tx *sql.Tx, id string, e *EntregaProjeto) error {
	for i, it := range e.Itens {
		if strings.TrimSpace(it.Descricao) == "" {
			continue
		}
		situacao := it.Situacao
		if situacao == "" {
			situacao = "entregue"
		}
		if _, err := tx.Exec(`
			INSERT INTO financeiro.entrega_itens (entrega_id, ordem, descricao, detalhe, situacao)
			VALUES ($1,$2,$3,$4,$5)`, id, i, it.Descricao, nulo(it.Detalhe), situacao); err != nil {
			return err
		}
	}
	for i, p := range e.Pendencias {
		if strings.TrimSpace(p.Descricao) == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO financeiro.entrega_pendencias (entrega_id, ordem, descricao, responsavel, prazo)
			VALUES ($1,$2,$3,$4,$5)`, id, i, p.Descricao, nulo(p.Responsavel), nuloData(p.Prazo)); err != nil {
			return err
		}
	}
	for i, s := range e.Signatarios {
		if strings.TrimSpace(s.Papel) == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO financeiro.entrega_signatarios
			    (entrega_id, ordem, papel, nome, cargo, organizacao, documento)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			id, i, s.Papel, nulo(s.Nome), nulo(s.Cargo), nulo(s.Organizacao), nulo(s.Documento)); err != nil {
			return err
		}
	}
	return nil
}

// ── GET /api/financeiro/entregas/detalhe?id=xxx ──────────────────────────────

func EntregaDetalheHandler(db *sql.DB) http.HandlerFunc {
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

		e, err := carregarEntrega(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(e)
	}
}

func carregarEntrega(db *sql.DB, id string) (*EntregaProjeto, error) {
	var e EntregaProjeto
	var clienteID, contratoID, produtoID sql.NullString
	var dataInicio, dataGolive sql.NullTime
	var dataEntrega, criadoEm time.Time
	var assinadoEm sql.NullTime

	err := db.QueryRow(`
		SELECT e.id, e.numero, e.cliente_id, e.contrato_id, e.produto_id,
		       e.versao, e.titulo, COALESCE(e.projeto,''), COALESCE(e.escopo_resumo,''),
		       e.data_inicio, e.data_golive, e.data_entrega,
		       e.garantia_dias, e.assistida_dias, COALESCE(e.ambiente,''),
		       COALESCE(e.responsavel_tecnico,''), COALESCE(e.responsavel_comercial,''),
		       COALESCE(e.observacoes,''), e.linhas_livres, e.status,
		       e.assinado_em, e.created_at,
		       COALESCE(cl.razao_social,''), COALESCE(cl.cnpj,''),
		       COALESCE(p.nome,''), COALESCE(p.codigo,'')
		FROM financeiro.entregas_projeto e
		LEFT JOIN financeiro.clientes cl ON cl.id = e.cliente_id
		LEFT JOIN financeiro.produtos p  ON p.id  = e.produto_id
		WHERE e.id = $1`, id).
		Scan(&e.ID, &e.Numero, &clienteID, &contratoID, &produtoID,
			&e.Versao, &e.Titulo, &e.Projeto, &e.EscopoResumo,
			&dataInicio, &dataGolive, &dataEntrega,
			&e.GarantiaDias, &e.AssistidaDias, &e.Ambiente,
			&e.ResponsavelTecnico, &e.ResponsavelComercial,
			&e.Observacoes, &e.LinhasLivres, &e.Status,
			&assinadoEm, &criadoEm,
			&e.ClienteNome, &e.ClienteCNPJ, &e.ProdutoNome, &e.ProdutoCod)
	if err != nil {
		return nil, fmt.Errorf("termo não encontrado: %w", err)
	}

	if clienteID.Valid {
		e.ClienteID = clienteID.String
	}
	if contratoID.Valid {
		e.ContratoID = contratoID.String
	}
	if produtoID.Valid {
		e.ProdutoID = produtoID.String
	}
	if dataInicio.Valid {
		e.DataInicio = dataInicio.Time.Format("2006-01-02")
	}
	if dataGolive.Valid {
		e.DataGolive = dataGolive.Time.Format("2006-01-02")
	}
	e.DataEntrega = dataEntrega.Format("2006-01-02")
	e.CriadoEm = criadoEm.Format(time.RFC3339)
	if assinadoEm.Valid {
		e.AssinadoEm = assinadoEm.Time.Format(time.RFC3339)
	}

	e.Itens = []EntregaItem{}
	rows, err := db.Query(`
		SELECT id, ordem, descricao, COALESCE(detalhe,''), situacao
		FROM financeiro.entrega_itens WHERE entrega_id = $1 ORDER BY ordem`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var it EntregaItem
		if err := rows.Scan(&it.ID, &it.Ordem, &it.Descricao, &it.Detalhe, &it.Situacao); err != nil {
			rows.Close()
			return nil, err
		}
		e.Itens = append(e.Itens, it)
	}
	rows.Close()

	e.Pendencias = []EntregaPendencia{}
	rows, err = db.Query(`
		SELECT id, ordem, descricao, COALESCE(responsavel,''), prazo
		FROM financeiro.entrega_pendencias WHERE entrega_id = $1 ORDER BY ordem`, id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p EntregaPendencia
		var prazo sql.NullTime
		if err := rows.Scan(&p.ID, &p.Ordem, &p.Descricao, &p.Responsavel, &prazo); err != nil {
			rows.Close()
			return nil, err
		}
		if prazo.Valid {
			p.Prazo = prazo.Time.Format("2006-01-02")
		}
		e.Pendencias = append(e.Pendencias, p)
	}
	rows.Close()

	e.Signatarios = []EntregaSignatario{}
	rows, err = db.Query(`
		SELECT id, ordem, papel, COALESCE(nome,''), COALESCE(cargo,''),
		       COALESCE(organizacao,''), COALESCE(documento,'')
		FROM financeiro.entrega_signatarios WHERE entrega_id = $1 ORDER BY ordem`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s EntregaSignatario
		if err := rows.Scan(&s.ID, &s.Ordem, &s.Papel, &s.Nome, &s.Cargo,
			&s.Organizacao, &s.Documento); err != nil {
			return nil, err
		}
		e.Signatarios = append(e.Signatarios, s)
	}
	return &e, nil
}

// ── GET /api/financeiro/entregas/pdf?id=xxx ──────────────────────────────────

func EntregaPDFHandler(db *sql.DB) http.HandlerFunc {
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

		e, err := carregarEntrega(db, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		emp := carregarEmpresaEmitente(db)

		pdf, err := gerarTermoEntregaPDF(e, emp)
		if err != nil {
			http.Error(w, "erro gerando PDF: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`inline; filename="termo-entrega-%s.pdf"`, e.Numero))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdf)))
		w.Write(pdf)
	}
}

// emitente reúne os dados da FBTECH impressos no documento.
type emitente struct {
	RazaoSocial string
	CNPJ        string
	Municipio   string
	UF          string
}

// carregarEmpresaEmitente lê a empresa cadastrada; se não houver, usa os dados
// institucionais da FBTECH como padrão.
func carregarEmpresaEmitente(db *sql.DB) emitente {
	e := emitente{
		RazaoSocial: "FORTES BEZERRA TECNOLOGIA E CONSULTORIA LTDA",
		CNPJ:        "38.149.716/0001-28",
		Municipio:   "Aparecida de Goiânia",
		UF:          "GO",
	}
	var razao, cnpj, mun, uf sql.NullString
	err := db.QueryRow(`
		SELECT razao_social, cnpj, municipio, uf
		FROM financeiro.empresas ORDER BY created_at LIMIT 1`).
		Scan(&razao, &cnpj, &mun, &uf)
	if err != nil {
		return e
	}
	if razao.Valid && razao.String != "" {
		e.RazaoSocial = razao.String
	}
	if cnpj.Valid && cnpj.String != "" {
		e.CNPJ = formatarCNPJ(cnpj.String)
	}
	if mun.Valid && mun.String != "" {
		e.Municipio = mun.String
	}
	if uf.Valid && uf.String != "" {
		e.UF = uf.String
	}
	return e
}

// rotuloSituacao traduz o enum de situação do item para o texto impresso.
func rotuloSituacao(s string) string {
	switch s {
	case "entregue":
		return "Entregue"
	case "parcial":
		return "Entregue parcialmente"
	case "pendente":
		return "Pendente"
	case "nao_aplicavel":
		return "Não aplicável"
	default:
		return s
	}
}

// dataOuLinha devolve a data formatada ou um traço de preenchimento quando o
// campo não foi informado na tela.
func dataOuLinha(iso string) string {
	if strings.TrimSpace(iso) == "" {
		return "______/______/__________"
	}
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("02/01/2006")
}

// gerarTermoEntregaPDF monta o Termo de Entrega e Aceite no Padrão FBTECH.
func gerarTermoEntregaPDF(e *EntregaProjeto, emp emitente) ([]byte, error) {
	dataEntrega, err := time.Parse("2006-01-02", e.DataEntrega)
	if err != nil {
		dataEntrega = time.Now()
	}

	produto := e.ProdutoNome
	if e.ProdutoCod != "" {
		produto = fmt.Sprintf("%s (%s)", e.ProdutoNome, e.ProdutoCod)
	}
	subtitulo := e.Projeto
	if subtitulo == "" {
		subtitulo = produto
	}

	doc := services.NovoDoc(services.DocMeta{
		Sigla:        "TE",
		Numero:       strings.TrimPrefix(e.Numero, "TE-"),
		Versao:       e.Versao,
		Titulo:       e.Titulo,
		Subtitulo:    subtitulo,
		Cliente:      e.ClienteNome,
		Cidade:       emp.Municipio + " – " + emp.UF,
		Data:         dataEntrega,
		Emitente:     emp.RazaoSocial,
		EmitCNPJ:     emp.CNPJ,
		Confidencial: true,
	})

	doc.Capa()

	// ── 1. Identificação ─────────────────────────────────────────────────────
	doc.Secao("Identificação")
	doc.Paragrafo(
		"Este Termo formaliza a entrega das funcionalidades descritas abaixo e registra o " +
			"aceite do CLIENTE, dando início aos prazos de operação assistida e de garantia " +
			"previstos neste documento.")
	doc.Esp(2)

	doc.Tabela(
		[]services.Coluna{{Titulo: "Campo", Largura: 4}, {Titulo: "Informação", Largura: 8}},
		[][]string{
			{"Cliente", ouLinha(e.ClienteNome)},
			{"CNPJ", ouLinha(formatarCNPJ(e.ClienteCNPJ))},
			{"Produto / Módulo", ouLinha(produto)},
			{"Projeto", ouLinha(e.Projeto)},
			{"Ambiente", ouLinha(e.Ambiente)},
			{"Início da implantação", dataOuLinha(e.DataInicio)},
			{"Go-live", dataOuLinha(e.DataGolive)},
			{"Data da entrega", dataOuLinha(e.DataEntrega)},
			{"Documento", e.Numero + " v." + e.Versao},
		})

	if e.EscopoResumo != "" {
		doc.Subsecao("Resumo do escopo entregue")
		doc.Paragrafo(e.EscopoResumo)
	}

	// ── 2. Escopo entregue ───────────────────────────────────────────────────
	doc.Secao("Escopo Entregue")
	if len(e.Itens) > 0 {
		doc.Paragrafo(
			"As funcionalidades relacionadas a seguir foram implantadas, testadas e " +
				"disponibilizadas ao CLIENTE no ambiente de produção:")
		doc.Esp(2)

		linhas := make([][]string, 0, len(e.Itens))
		for i, it := range e.Itens {
			linhas = append(linhas, []string{
				fmt.Sprintf("%d", i+1),
				it.Descricao,
				it.Detalhe,
				rotuloSituacao(it.Situacao),
			})
		}
		doc.Tabela([]services.Coluna{
			{Titulo: "#", Largura: 1},
			{Titulo: "Entregável", Largura: 5},
			{Titulo: "Detalhamento", Largura: 4},
			{Titulo: "Situação", Largura: 2},
		}, linhas)
	} else {
		// Sem itens informados na tela, o termo é impresso com linhas pautadas
		// para que o escopo seja relacionado à mão na reunião de aceite.
		doc.Paragrafo("Relacione abaixo as funcionalidades entregues:")
		doc.Esp(2)
		doc.LinhasEmBranco(8)
	}

	// ── 3. Pendências e ressalvas ────────────────────────────────────────────
	doc.Secao("Pendências e Ressalvas")
	if len(e.Pendencias) > 0 {
		linhas := make([][]string, 0, len(e.Pendencias))
		for i, p := range e.Pendencias {
			linhas = append(linhas, []string{
				fmt.Sprintf("%d", i+1), p.Descricao,
				ouLinha(p.Responsavel), dataOuLinha(p.Prazo),
			})
		}
		doc.Tabela([]services.Coluna{
			{Titulo: "#", Largura: 1},
			{Titulo: "Pendência", Largura: 6},
			{Titulo: "Responsável", Largura: 3},
			{Titulo: "Prazo", Largura: 2},
		}, linhas)
	} else {
		doc.Paragrafo(
			"Não há pendências registradas até a data deste Termo. Ressalvas apontadas no " +
				"ato da assinatura devem ser anotadas nas linhas abaixo:")
		doc.Esp(2)
	}
	if e.LinhasLivres > 0 {
		doc.LinhasEmBranco(e.LinhasLivres)
	}

	// ── 4. Operação assistida e garantia ─────────────────────────────────────
	doc.Secao("Operação Assistida e Garantia")
	doc.Marcador(fmt.Sprintf(
		"Operação assistida de %d (%s) dias corridos a contar do go-live, com suporte "+
			"remoto da equipe do projeto em horário comercial.",
		e.AssistidaDias, services.PorExtenso(e.AssistidaDias)))
	doc.Marcador(fmt.Sprintf(
		"Garantia de %d (%s) dias corridos após o encerramento da operação assistida, "+
			"limitada à correção de defeitos das funcionalidades relacionadas na seção 2.",
		e.GarantiaDias, services.PorExtenso(e.GarantiaDias)))
	doc.Marcador(
		"Considera-se defeito o comportamento divergente do especificado e homologado. " +
			"Novos requisitos, alterações de regra de negócio e integrações não previstas " +
			"são tratados como solicitação de mudança, em proposta específica.")
	doc.Marcador(
		"A garantia perde validade caso as configurações, parametrizações ou objetos " +
			"entregues sejam alterados por terceiros não autorizados pela CONTRATADA.")
	doc.Esp(3)

	// ── 5. Declaração de aceite ──────────────────────────────────────────────
	doc.Secao("Declaração de Aceite")
	doc.Paragrafo(
		"O CLIENTE declara que as funcionalidades descritas neste Termo foram apresentadas, " +
			"testadas e homologadas, encontrando-se em conformidade com o escopo contratado, " +
			"e que a solução está em operação produtiva. As pendências eventualmente " +
			"registradas na seção 3 não impedem o aceite e serão tratadas nos prazos ali " +
			"acordados.")

	if e.Observacoes != "" {
		doc.Subsecao("Observações")
		doc.Paragrafo(e.Observacoes)
	}

	doc.Subsecao("Responsáveis pelo projeto")
	doc.LinhaPreenchivel("Responsável técnico", e.ResponsavelTecnico, 4)
	doc.LinhaPreenchivel("Responsável comercial", e.ResponsavelComercial, 4)
	doc.LinhaPreenchivel("Gestor pelo CLIENTE", "", 4)

	// ── 6. Assinaturas ───────────────────────────────────────────────────────
	doc.Secao("Assinaturas")
	doc.LocalEData(emp.Municipio + " – " + emp.UF)

	signatarios := make([]services.Signatario, 0, len(e.Signatarios)+2)
	for _, s := range e.Signatarios {
		signatarios = append(signatarios, services.Signatario{
			Papel: s.Papel, Nome: s.Nome, Cargo: s.Cargo,
			Org: s.Organizacao, Doc: s.Documento,
		})
	}
	if len(signatarios) == 0 {
		// Padrão do termo: uma via para cada parte.
		signatarios = []services.Signatario{
			{Papel: "PELA CONTRATADA", Nome: e.ResponsavelTecnico, Org: emp.RazaoSocial},
			{Papel: "PELO CLIENTE", Org: e.ClienteNome},
		}
	}
	doc.BlocoAssinaturas(signatarios)

	doc.Esp(6)
	doc.Nota(fmt.Sprintf(
		"Emitido em 2 (duas) vias de igual teor. Documento %s v.%s gerado pelo FBTax Cloud em %s.",
		e.Numero, e.Versao, time.Now().Format("02/01/2006 às 15:04")))

	return doc.Bytes()
}

// ouLinha devolve o valor ou um traço de preenchimento quando vazio.
func ouLinha(s string) string {
	if strings.TrimSpace(s) == "" {
		return "________________________"
	}
	return s
}

// ── POST /api/financeiro/entregas/upload-assinado ────────────────────────────

func EntregaUploadAssinadoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		r.ParseMultipartForm(20 << 20)

		id := r.FormValue("entrega_id")
		if id == "" {
			http.Error(w, "entrega_id obrigatório", http.StatusBadRequest)
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

		if _, err := db.Exec(`
			UPDATE financeiro.entregas_projeto
			SET assinado_data = $1, assinado_nome = $2, assinado_em = NOW(),
			    status = 'assinado', updated_at = NOW()
			WHERE id = $3`, data, header.Filename, id); err != nil {
			http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"mensagem": "Termo assinado enviado com sucesso"})
	}
}

// ── GET /api/financeiro/entregas/download-assinado?id=xxx ────────────────────

func EntregaDownloadAssinadoHandler(db *sql.DB) http.HandlerFunc {
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
		err := db.QueryRow(`
			SELECT assinado_data, assinado_nome
			FROM financeiro.entregas_projeto WHERE id = $1`, id).Scan(&data, &nome)
		if err != nil || data == nil {
			http.Error(w, "arquivo não encontrado", http.StatusNotFound)
			return
		}

		filename := "termo-entrega-assinado.pdf"
		if nome.Valid && nome.String != "" {
			filename = nome.String
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.Write(data)
	}
}
