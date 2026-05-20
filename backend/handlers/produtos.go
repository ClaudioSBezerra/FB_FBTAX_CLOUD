package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type PlanoResponse struct {
	ID        string   `json:"id"`
	ProdutoID string   `json:"produto_id"`
	Nome      string   `json:"nome"`
	Preco     *float64 `json:"preco"`
	Ativo     bool     `json:"ativo"`
}

type ProdutoResponse struct {
	ID       string          `json:"id"`
	Codigo   string          `json:"codigo"`
	Nome     string          `json:"nome"`
	Descricao string         `json:"descricao,omitempty"`
	Ativo    bool            `json:"ativo"`
	Planos   []PlanoResponse `json:"planos"`
}

type PlanoUpdateRequest struct {
	ID    string   `json:"id"`
	Preco *float64 `json:"preco"`
}

func ProdutosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleListProdutos(w, r, db)
	}
}

func handleListProdutos(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.Query(`
		SELECT id, codigo, nome, COALESCE(descricao,''), ativo
		FROM financeiro.produtos
		ORDER BY nome
	`)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	produtos := []ProdutoResponse{}
	for rows.Next() {
		var p ProdutoResponse
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Nome, &p.Descricao, &p.Ativo); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		planos, err := fetchPlanos(db, p.ID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		p.Planos = planos
		produtos = append(produtos, p)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(produtos)
}

func fetchPlanos(db *sql.DB, produtoID string) ([]PlanoResponse, error) {
	rows, err := db.Query(`
		SELECT id, produto_id, nome, preco, ativo
		FROM financeiro.planos
		WHERE produto_id = $1
		ORDER BY CASE nome
			WHEN 'Lite'        THEN 1
			WHEN 'Standard'    THEN 2
			WHEN 'Premium'     THEN 3
			WHEN 'Enterprise'  THEN 4
			WHEN 'Sob Demanda' THEN 5
			ELSE 6
		END
	`, produtoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	planos := []PlanoResponse{}
	for rows.Next() {
		var p PlanoResponse
		var preco sql.NullFloat64
		if err := rows.Scan(&p.ID, &p.ProdutoID, &p.Nome, &preco, &p.Ativo); err != nil {
			return nil, err
		}
		if preco.Valid {
			p.Preco = &preco.Float64
		}
		planos = append(planos, p)
	}
	return planos, rows.Err()
}

func PlanosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPut {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req PlanoUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if req.ID == "" {
			http.Error(w, "id required", http.StatusBadRequest)
			return
		}
		_, err := db.Exec(`
			UPDATE financeiro.planos
			SET preco=$1, updated_at=CURRENT_TIMESTAMP
			WHERE id=$2
		`, req.Preco, req.ID)
		if err != nil {
			http.Error(w, "error updating plano", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}
