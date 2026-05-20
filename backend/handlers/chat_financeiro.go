package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"fb_cloud/services"
)

type chatFinRequest struct {
	Pergunta  string                     `json:"pergunta"`
	Historico []services.FinanceiroChatMsg `json:"historico"`
}

func ChatFinanceiroHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var req chatFinRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if req.Pergunta == "" {
			http.Error(w, "pergunta is required", http.StatusBadRequest)
			return
		}

		result, err := services.ResponderPerguntaFinanceiro(db, req.Pergunta, req.Historico)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(result)
	}
}
