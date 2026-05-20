package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type ValidacaoTokenResponse struct {
	Status     string                `json:"status"`
	ContratoID string                `json:"contrato_id"`
	Tokens     []ValidacaoTokenItem  `json:"tokens"`
}

type ValidacaoTokenItem struct {
	PlanoID      string  `json:"plano_id"`
	PlanoNome    string  `json:"plano_nome"`
	ProdutoCodigo string `json:"produto_codigo"`
	Token        string  `json:"token"`
	Status       string  `json:"status"`
	ValidUntil   string  `json:"valid_until"`
}

// ValidacaoHandler é público — autenticado apenas pela API Key no header X-API-Key.
func ValidacaoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "X-API-Key required"})
			return
		}

		var contratoID string
		var ativo bool
		err := db.QueryRow(
			`SELECT contrato_id, ativo FROM financeiro.api_keys WHERE api_key = $1`, apiKey,
		).Scan(&contratoID, &ativo)
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid api key"})
			return
		}
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if !ativo {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "api key inativa"})
			return
		}

		produtoCodigo := r.URL.Query().Get("produto")

		query := `
			SELECT t.plano_id, pl.nome, p.codigo, t.token, t.status, t.valid_until::text
			FROM financeiro.tokens t
			JOIN financeiro.planos pl   ON pl.id = t.plano_id
			JOIN financeiro.produtos p  ON p.id  = pl.produto_id
			WHERE t.contrato_id = $1
			  AND t.status IN ('ativo','em_carencia')
		`
		args := []interface{}{contratoID}
		if produtoCodigo != "" {
			query += ` AND p.codigo = $2`
			args = append(args, produtoCodigo)
		}
		query += ` ORDER BY p.nome, pl.nome`

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		tokens := []ValidacaoTokenItem{}
		overallStatus := "sem_tokens"
		for rows.Next() {
			var item ValidacaoTokenItem
			if err := rows.Scan(&item.PlanoID, &item.PlanoNome, &item.ProdutoCodigo,
				&item.Token, &item.Status, &item.ValidUntil); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			tokens = append(tokens, item)
			if overallStatus == "sem_tokens" || (item.Status == "ativo" && overallStatus != "ativo") {
				overallStatus = item.Status
			}
		}

		json.NewEncoder(w).Encode(ValidacaoTokenResponse{
			Status:     overallStatus,
			ContratoID: contratoID,
			Tokens:     tokens,
		})
	}
}
