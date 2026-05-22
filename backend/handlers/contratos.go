package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type ContratoItemRequest struct {
	PlanoID   string   `json:"plano_id"`
	ValorItem *float64 `json:"valor_item,omitempty"`
}

type ContratoRequest struct {
	ID           string                `json:"id,omitempty"`
	ClienteID    string                `json:"cliente_id"`
	DataInicio   string                `json:"data_inicio"`
	Periodicidade string               `json:"periodicidade"`
	ValorTotal   float64               `json:"valor_total"`
	Status       string                `json:"status,omitempty"`
	Observacoes  string                `json:"observacoes,omitempty"`
	CnpjIDs      []string              `json:"cnpj_ids"`
	Itens        []ContratoItemRequest `json:"itens"`
}

type ContratoItemResponse struct {
	PlanoID      string   `json:"plano_id"`
	ProdutoCodigo string  `json:"produto_codigo"`
	PlanoNome    string   `json:"plano_nome"`
	ValorItem    *float64 `json:"valor_item"`
}

type ContratoResponse struct {
	ID             string                 `json:"id"`
	ClienteID      string                 `json:"cliente_id"`
	DataInicio     string                 `json:"data_inicio"`
	Periodicidade  string                 `json:"periodicidade"`
	ValorTotal     float64                `json:"valor_total"`
	Status         string                 `json:"status"`
	Observacoes    string                 `json:"observacoes,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	Itens          []ContratoItemResponse `json:"itens"`
	TokensAtivos   int                    `json:"tokens_ativos"`
	TokensTotal    int                    `json:"tokens_total"`
}

func ContratosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("id") != "" {
				handleGetContrato(w, r, db)
			} else {
				handleListContratos(w, r, db)
			}
		case http.MethodPost:
			handlePostContrato(w, r, db)
		case http.MethodPut:
			handlePutContrato(w, r, db)
		case http.MethodDelete:
			handleDeleteContrato(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleListContratos(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	clienteID := r.URL.Query().Get("cliente_id")
	query := `
		SELECT c.id, c.cliente_id, c.data_inicio, c.periodicidade,
		       c.valor_total, c.status, COALESCE(c.observacoes,''), c.created_at,
		       COUNT(t.id) FILTER (WHERE t.status = 'ativo')    AS tokens_ativos,
		       COUNT(t.id)                                       AS tokens_total
		FROM financeiro.contratos c
		LEFT JOIN financeiro.tokens t ON t.contrato_id = c.id
		WHERE 1=1`
	args := []interface{}{}
	argN := 1

	if clienteID != "" {
		query += fmt.Sprintf(` AND c.cliente_id = $%d`, argN)
		args = append(args, clienteID)
		argN++
	}
	query += ` GROUP BY c.id ORDER BY c.created_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	contratos := []ContratoResponse{}
	for rows.Next() {
		var c ContratoResponse
		if err := rows.Scan(&c.ID, &c.ClienteID, &c.DataInicio, &c.Periodicidade,
			&c.ValorTotal, &c.Status, &c.Observacoes, &c.CreatedAt,
			&c.TokensAtivos, &c.TokensTotal); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		itens, err := fetchContratoItens(db, c.ID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		c.Itens = itens
		contratos = append(contratos, c)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(contratos)
}

func handleGetContrato(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	var c ContratoResponse
	err := db.QueryRow(`
		SELECT id, cliente_id, data_inicio, periodicidade, valor_total,
		       status, COALESCE(observacoes,''), created_at
		FROM financeiro.contratos WHERE id = $1
	`, id).Scan(&c.ID, &c.ClienteID, &c.DataInicio, &c.Periodicidade,
		&c.ValorTotal, &c.Status, &c.Observacoes, &c.CreatedAt)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "contrato not found"})
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	itens, err := fetchContratoItens(db, c.ID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	c.Itens = itens
	json.NewEncoder(w).Encode(c)
}

func fetchContratoItens(db *sql.DB, contratoID string) ([]ContratoItemResponse, error) {
	rows, err := db.Query(`
		SELECT ci.plano_id, p.codigo, pl.nome, ci.valor_item
		FROM financeiro.contrato_itens ci
		JOIN financeiro.planos pl ON pl.id = ci.plano_id
		JOIN financeiro.produtos p ON p.id = pl.produto_id
		WHERE ci.contrato_id = $1
		ORDER BY p.nome, pl.nome
	`, contratoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itens := []ContratoItemResponse{}
	for rows.Next() {
		var item ContratoItemResponse
		var valorItem sql.NullFloat64
		if err := rows.Scan(&item.PlanoID, &item.ProdutoCodigo, &item.PlanoNome, &valorItem); err != nil {
			return nil, err
		}
		if valorItem.Valid {
			item.ValorItem = &valorItem.Float64
		}
		itens = append(itens, item)
	}
	return itens, rows.Err()
}

func handlePostContrato(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req ContratoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.ClienteID == "" || req.DataInicio == "" || req.Periodicidade == "" || req.ValorTotal == 0 {
		http.Error(w, "cliente_id, data_inicio, periodicidade and valor_total are required", http.StatusBadRequest)
		return
	}
	if len(req.Itens) == 0 {
		http.Error(w, "at least one item is required", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	status := req.Status
	if status == "" {
		status = "ativo"
	}

	var id string
	err = tx.QueryRow(`
		INSERT INTO financeiro.contratos
		(cliente_id, data_inicio, periodicidade, valor_total, status, observacoes)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, req.ClienteID, req.DataInicio, req.Periodicidade, req.ValorTotal, status, req.Observacoes).Scan(&id)
	if err != nil {
		http.Error(w, "error creating contrato", http.StatusInternalServerError)
		return
	}

	for _, cnpjID := range req.CnpjIDs {
		if _, err := tx.Exec(`
			INSERT INTO financeiro.contrato_cnpjs (contrato_id, cnpj_id) VALUES ($1, $2)
		`, id, cnpjID); err != nil {
			http.Error(w, "error adding contrato cnpj", http.StatusInternalServerError)
			return
		}
	}

	var planoIDs []string
	for _, item := range req.Itens {
		if _, err := tx.Exec(`
			INSERT INTO financeiro.contrato_itens (contrato_id, plano_id, valor_item) VALUES ($1, $2, $3)
		`, id, item.PlanoID, item.ValorItem); err != nil {
			http.Error(w, "error adding contrato item", http.StatusInternalServerError)
			return
		}
		planoIDs = append(planoIDs, item.PlanoID)
	}

	if err := GerarTokensContrato(tx, id, req.DataInicio, planoIDs); err != nil {
		log.Printf("[Contratos] erro gerando tokens: %v", err)
		http.Error(w, "error generating tokens", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("[Contratos] Contrato %s criado para cliente %s com %d itens", id, req.ClienteID, len(req.Itens))
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func handlePutContrato(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req ContratoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`
		UPDATE financeiro.contratos
		SET periodicidade=$1, valor_total=$2, status=$3, observacoes=$4, updated_at=CURRENT_TIMESTAMP
		WHERE id=$5
	`, req.Periodicidade, req.ValorTotal, req.Status, req.Observacoes, req.ID)
	if err != nil {
		http.Error(w, "error updating contrato", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func handleDeleteContrato(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id obrigatório", http.StatusBadRequest)
		return
	}

	var tokenAtivo int
	db.QueryRow(`SELECT COUNT(*) FROM financeiro.tokens WHERE contrato_id = $1 AND status = 'ativo'`, id).Scan(&tokenAtivo)
	if tokenAtivo > 0 {
		http.Error(w, "contrato possui tokens ativos — inative todos os tokens antes de excluir", http.StatusConflict)
		return
	}

	res, err := db.Exec(`DELETE FROM financeiro.contratos WHERE id = $1`, id)
	if err != nil {
		http.Error(w, "erro ao excluir contrato", http.StatusInternalServerError)
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		http.Error(w, "contrato não encontrado", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
