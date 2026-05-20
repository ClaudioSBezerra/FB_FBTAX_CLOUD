package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type ApiKeyResponse struct {
	ID         string `json:"id"`
	ContratoID string `json:"contrato_id"`
	ApiKey     string `json:"api_key"`
	Descricao  string `json:"descricao,omitempty"`
	Ativo      bool   `json:"ativo"`
	CreatedAt  string `json:"created_at"`
}

func ApiKeysHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleGetApiKeys(w, r, db)
		case http.MethodPost:
			handleCreateApiKey(w, r, db)
		case http.MethodPut:
			handleUpdateApiKey(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleGetApiKeys(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	contratoID := r.URL.Query().Get("contrato_id")
	query := `SELECT id, contrato_id, api_key, COALESCE(descricao,''), ativo, created_at::text
	          FROM financeiro.api_keys WHERE 1=1`
	args := []interface{}{}
	if contratoID != "" {
		query += ` AND contrato_id = $1`
		args = append(args, contratoID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	keys := []ApiKeyResponse{}
	for rows.Next() {
		var k ApiKeyResponse
		if err := rows.Scan(&k.ID, &k.ContratoID, &k.ApiKey, &k.Descricao, &k.Ativo, &k.CreatedAt); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		keys = append(keys, k)
	}
	json.NewEncoder(w).Encode(keys)
}

func handleCreateApiKey(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ContratoID string `json:"contrato_id"`
		Descricao  string `json:"descricao,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ContratoID == "" {
		http.Error(w, "contrato_id required", http.StatusBadRequest)
		return
	}

	apiKey := strings.ReplaceAll(uuid.New().String(), "-", "") +
		strings.ReplaceAll(uuid.New().String(), "-", "")[:16]

	var id string
	err := db.QueryRow(`
		INSERT INTO financeiro.api_keys (contrato_id, api_key, descricao)
		VALUES ($1, $2, $3) RETURNING id
	`, req.ContratoID, apiKey, req.Descricao).Scan(&id)
	if err != nil {
		http.Error(w, "error creating api key", http.StatusInternalServerError)
		return
	}

	log.Printf("[ApiKeys] API Key criada para contrato %s", req.ContratoID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "api_key": apiKey})
}

func handleUpdateApiKey(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ID    string `json:"id"`
		Ativo bool   `json:"ativo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if _, err := db.Exec(
		`UPDATE financeiro.api_keys SET ativo = $1 WHERE id = $2`,
		req.Ativo, req.ID,
	); err != nil {
		http.Error(w, "error updating api key", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
