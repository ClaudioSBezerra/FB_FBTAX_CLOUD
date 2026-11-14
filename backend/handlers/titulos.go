package handlers

// Contas a Receber — /api/financeiro/titulos
//   GET                      lista (filtros status, cliente_id, q) com dias de atraso
//   POST                     cria título
//   PUT                      edita título em aberto
//   GET  ?action=candidatos&id=  créditos do extrato que podem ter pago o título
//   POST ?action=baixar      baixa manual {id, data, valor, obs}
//   POST ?action=vincular    baixa ligando a um lançamento {id, transacao_id}
//   POST ?action=cancelar    cancela título em aberto {id, obs}
//   POST ?action=conciliar   roda a conciliação automática agora

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fb_cloud/services"
)

type tituloReq struct {
	ID          string  `json:"id,omitempty"`
	ClienteID   string  `json:"cliente_id"`
	ContratoID  string  `json:"contrato_id,omitempty"`
	Tipo        string  `json:"tipo"`
	Competencia string  `json:"competencia,omitempty"` // AAAA-MM-DD (usa o mês)
	Descricao   string  `json:"descricao"`
	Valor       float64 `json:"valor"`
	Vencimento  string  `json:"vencimento"`
	NFSeID      string  `json:"nfse_id,omitempty"`
}

func validarTitulo(db *sql.DB, req *tituloReq) string {
	if !reUUIDNFS.MatchString(req.ClienteID) {
		return "cliente obrigatório"
	}
	switch req.Tipo {
	case "mensalidade", "adesao", "projeto", "outro":
	default:
		return "tipo deve ser mensalidade, adesao, projeto ou outro"
	}
	if req.Valor <= 0 {
		return "valor deve ser maior que zero"
	}
	if strings.TrimSpace(req.Descricao) == "" {
		return "descrição obrigatória"
	}
	if _, err := time.Parse("2006-01-02", req.Vencimento); err != nil {
		return "vencimento obrigatório (AAAA-MM-DD)"
	}
	if req.Tipo == "mensalidade" {
		c, err := time.Parse("2006-01-02", req.Competencia)
		if err != nil {
			return "competência obrigatória para mensalidade (AAAA-MM-DD)"
		}
		req.Competencia = time.Date(c.Year(), c.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		if req.ContratoID == "" {
			return "mensalidade exige o contrato (é ele que recebe a renovação do token)"
		}
	} else {
		req.Competencia = ""
	}
	if req.ContratoID != "" {
		if !reUUIDNFS.MatchString(req.ContratoID) {
			return "contrato inválido"
		}
		var ok bool
		db.QueryRow(`SELECT EXISTS(SELECT 1 FROM financeiro.contratos WHERE id = $1 AND cliente_id = $2)`, req.ContratoID, req.ClienteID).Scan(&ok)
		if !ok {
			return "o contrato não pertence a este cliente"
		}
	}
	return ""
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// TitulosHandler trata todas as rotas do Contas a Receber.
func TitulosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		switch {
		case r.Method == http.MethodGet && action == "candidatos":
			titulosCandidatos(w, r, db)
		case r.Method == http.MethodGet:
			titulosListar(w, r, db)
		case r.Method == http.MethodPost && action == "baixar":
			titulosBaixar(w, r, db)
		case r.Method == http.MethodPost && action == "vincular":
			titulosVincular(w, r, db)
		case r.Method == http.MethodPost && action == "cancelar":
			titulosCancelar(w, r, db)
		case r.Method == http.MethodPost && action == "conciliar":
			ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
			defer cancel()
			n, err := services.ConciliarTitulos(ctx, db)
			if err != nil {
				nfseErro(w, http.StatusInternalServerError, "erro na conciliação")
				return
			}
			nfseJSON(w, http.StatusOK, map[string]int{"baixados": n})
		case r.Method == http.MethodPost:
			titulosCriar(w, r, db)
		case r.Method == http.MethodPut:
			titulosEditar(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func titulosListar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	q := r.URL.Query()
	limite := 300
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 && n <= 1000 {
		limite = n
	}
	rows, err := db.Query(`
		SELECT t.id, t.cliente_id, c.razao_social, COALESCE(t.contrato_id::text,''), t.tipo, COALESCE(t.competencia::text,''),
		       t.descricao, t.valor, t.vencimento::text, t.status,
		       CASE WHEN t.status = 'aberto' AND t.vencimento < CURRENT_DATE THEN (CURRENT_DATE - t.vencimento) ELSE 0 END,
		       COALESCE(t.pago_em::text,''), t.valor_pago, COALESCE(t.baixa_origem,''), COALESCE(t.baixa_obs,''),
		       COALESCE(t.nfse_id::text,''), COALESCE(n.n_nfse::text,''), COALESCE(t.cobranca_status,'')
		FROM financeiro.titulos_receber t
		JOIN financeiro.clientes c ON c.id = t.cliente_id
		LEFT JOIN financeiro.nfse_emitidas n ON n.id = t.nfse_id
		WHERE ($1 = '' OR t.status = $1) AND ($2 = '' OR t.cliente_id::text = $2)
		  AND ($3 = '' OR c.razao_social ILIKE '%'||$3||'%' OR t.descricao ILIKE '%'||$3||'%')
		ORDER BY (t.status = 'aberto') DESC, t.vencimento, t.created_at
		LIMIT $4`, q.Get("status"), q.Get("cliente_id"), strings.TrimSpace(q.Get("q")), limite)
	if err != nil {
		log.Printf("[Titulos] listar: %v", err)
		nfseErro(w, http.StatusInternalServerError, "erro ao listar títulos")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, cli, razao, contrato, tipo, comp, desc, venc, status, pagoEm, origem, obs, nfse, nNFSe, cobr string
		var valor float64
		var valorPago sql.NullFloat64
		var atraso int
		if err := rows.Scan(&id, &cli, &razao, &contrato, &tipo, &comp, &desc, &valor, &venc, &status, &atraso, &pagoEm, &valorPago, &origem, &obs, &nfse, &nNFSe, &cobr); err != nil {
			continue
		}
		m := map[string]any{"id": id, "cliente_id": cli, "cliente": razao, "contrato_id": contrato, "tipo": tipo, "competencia": comp,
			"descricao": desc, "valor": valor, "vencimento": venc, "status": status, "dias_atraso": atraso, "pago_em": pagoEm,
			"baixa_origem": origem, "baixa_obs": obs, "nfse_id": nfse, "n_nfse": nNFSe, "cobranca_status": cobr}
		if valorPago.Valid {
			m["valor_pago"] = valorPago.Float64
		}
		out = append(out, m)
	}
	nfseJSON(w, http.StatusOK, out)
}

func titulosCriar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req tituloReq
	if json.NewDecoder(io.LimitReader(r.Body, 32<<10)).Decode(&req) != nil {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	if msg := validarTitulo(db, &req); msg != "" {
		nfseErro(w, http.StatusBadRequest, msg)
		return
	}
	var id string
	err := db.QueryRow(`
		INSERT INTO financeiro.titulos_receber (cliente_id, contrato_id, tipo, competencia, descricao, valor, vencimento, nfse_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		req.ClienteID, nullStr(req.ContratoID), req.Tipo, nullStr(req.Competencia), strings.Join(strings.Fields(req.Descricao), " "),
		req.Valor, req.Vencimento, nullStr(req.NFSeID)).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "uq_titulos_mensalidade") {
			nfseErro(w, http.StatusConflict, "já existe uma mensalidade desse contrato para essa competência")
			return
		}
		log.Printf("[Titulos] criar: %v", err)
		nfseErro(w, http.StatusInternalServerError, "erro ao criar título")
		return
	}
	nfseJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func titulosEditar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req tituloReq
	if json.NewDecoder(io.LimitReader(r.Body, 32<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ID) {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	if msg := validarTitulo(db, &req); msg != "" {
		nfseErro(w, http.StatusBadRequest, msg)
		return
	}
	res, err := db.Exec(`
		UPDATE financeiro.titulos_receber
		SET cliente_id=$2, contrato_id=$3, tipo=$4, competencia=$5, descricao=$6, valor=$7, vencimento=$8, nfse_id=$9, updated_at=NOW()
		WHERE id=$1 AND status='aberto' AND cobranca_codigo IS NULL`,
		req.ID, req.ClienteID, nullStr(req.ContratoID), req.Tipo, nullStr(req.Competencia), strings.Join(strings.Fields(req.Descricao), " "),
		req.Valor, req.Vencimento, nullStr(req.NFSeID))
	if err != nil {
		nfseErro(w, http.StatusConflict, "não foi possível editar (mensalidade duplicada?)")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		nfseErro(w, http.StatusConflict, "só é possível editar título em aberto e sem cobrança emitida no banco")
		return
	}
	nfseJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func titulosCancelar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ID  string `json:"id"`
		Obs string `json:"obs"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ID) {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	res, err := db.Exec(`UPDATE financeiro.titulos_receber SET status='cancelado', baixa_obs=NULLIF($2,''), updated_at=NOW() WHERE id=$1 AND status='aberto'`, req.ID, strings.TrimSpace(req.Obs))
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao cancelar")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		nfseErro(w, http.StatusConflict, "só é possível cancelar título em aberto")
		return
	}
	nfseJSON(w, http.StatusOK, map[string]string{"status": "cancelado"})
}

func baixarComTransacao(w http.ResponseWriter, db *sql.DB, id string, transacaoID *string, data time.Time, valor float64, origem, obs string) {
	tx, err := db.Begin()
	if err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro interno")
		return
	}
	defer tx.Rollback()
	renovados, err := services.BaixarTitulo(tx, id, transacaoID, data, valor, origem, obs)
	if err != nil {
		if strings.Contains(err.Error(), "uq_titulos_transacao") {
			nfseErro(w, http.StatusConflict, "este lançamento já baixou outro título")
			return
		}
		nfseErro(w, http.StatusConflict, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		nfseErro(w, http.StatusInternalServerError, "erro ao confirmar a baixa")
		return
	}
	log.Printf("[Titulos] título %s baixado (%s); %d token(s) renovado(s)", id, origem, renovados)
	nfseJSON(w, http.StatusOK, map[string]any{"status": "pago", "tokens_renovados": renovados})
}

func titulosBaixar(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ID    string  `json:"id"`
		Data  string  `json:"data"`
		Valor float64 `json:"valor"`
		Obs   string  `json:"obs"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ID) {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	data, err := time.Parse("2006-01-02", req.Data)
	if err != nil {
		nfseErro(w, http.StatusBadRequest, "data do pagamento obrigatória (AAAA-MM-DD)")
		return
	}
	baixarComTransacao(w, db, req.ID, nil, data, req.Valor, "manual", strings.TrimSpace(req.Obs))
}

func titulosVincular(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ID          string `json:"id"`
		TransacaoID string `json:"transacao_id"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&req) != nil || !reUUIDNFS.MatchString(req.ID) || !reUUIDNFS.MatchString(req.TransacaoID) {
		nfseErro(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	var data time.Time
	var valor float64
	var tipo string
	if err := db.QueryRow(`SELECT data_transacao, valor, tipo FROM financeiro.transacoes WHERE id = $1`, req.TransacaoID).Scan(&data, &valor, &tipo); err != nil {
		nfseErro(w, http.StatusNotFound, "lançamento não encontrado")
		return
	}
	if tipo != "credito" {
		nfseErro(w, http.StatusBadRequest, "só um crédito (entrada) pode baixar um título")
		return
	}
	baixarComTransacao(w, db, req.ID, &req.TransacaoID, data, valor, "manual", "vinculado manualmente a um lançamento do extrato")
}

func titulosCandidatos(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if !reUUIDNFS.MatchString(id) {
		nfseErro(w, http.StatusBadRequest, "id inválido")
		return
	}
	rows, err := db.Query(`
		SELECT t.id, t.data_transacao::text, t.descricao, t.valor, COALESCE(t.categoria,''),
		       COALESCE(t.detalhes->'detalhes'->>'nomePagador',''), COALESCE(t.detalhes->'detalhes'->>'cpfCnpjPagador',''),
		       (v.valor = t.valor) AS valor_igual
		FROM financeiro.titulos_receber v
		JOIN financeiro.transacoes t ON t.tipo = 'credito' AND t.data_transacao >= v.vencimento - 90
		WHERE v.id = $1 AND v.status = 'aberto'
		  AND NOT EXISTS (SELECT 1 FROM financeiro.titulos_receber x WHERE x.transacao_id = t.id)
		ORDER BY (v.valor = t.valor) DESC, ABS(t.data_transacao - v.vencimento), t.data_transacao DESC
		LIMIT 30`, id)
	if err != nil {
		log.Printf("[Titulos] candidatos: %v", err)
		nfseErro(w, http.StatusInternalServerError, "erro ao buscar lançamentos")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var tid, data, desc, cat, nome, cnpj string
		var valor float64
		var igual bool
		if rows.Scan(&tid, &data, &desc, &valor, &cat, &nome, &cnpj, &igual) == nil {
			out = append(out, map[string]any{"transacao_id": tid, "data": data, "descricao": desc, "valor": valor, "categoria": cat, "pagador": nome, "cnpj_pagador": cnpj, "valor_igual": igual})
		}
	}
	nfseJSON(w, http.StatusOK, out)
}
