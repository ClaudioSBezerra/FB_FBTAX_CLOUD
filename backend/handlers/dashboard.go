package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type DashboardFinanceiroResponse struct {
	TotalClientes              int `json:"total_clientes"`
	TotalContratos             int `json:"total_contratos"`
	ContratosAtivos            int `json:"contratos_ativos"`
	TokensAtivos               int `json:"tokens_ativos"`
	TokensEmCarencia           int `json:"tokens_em_carencia"`
	TokensSuspensos            int `json:"tokens_suspensos"`
	TokensProximosVencimento   int `json:"tokens_proximos_vencimento"`
}

func DashboardFinanceiroHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var d DashboardFinanceiroResponse

		db.QueryRow(`SELECT COUNT(*) FROM financeiro.clientes WHERE ativo = true`).Scan(&d.TotalClientes)           //nolint
		db.QueryRow(`SELECT COUNT(*) FROM financeiro.contratos`).Scan(&d.TotalContratos)                           //nolint
		db.QueryRow(`SELECT COUNT(*) FROM financeiro.contratos WHERE status = 'ativo'`).Scan(&d.ContratosAtivos)   //nolint
		db.QueryRow(`SELECT COUNT(*) FROM financeiro.tokens WHERE status = 'ativo'`).Scan(&d.TokensAtivos)         //nolint
		db.QueryRow(`SELECT COUNT(*) FROM financeiro.tokens WHERE status = 'em_carencia'`).Scan(&d.TokensEmCarencia) //nolint
		db.QueryRow(`SELECT COUNT(*) FROM financeiro.tokens WHERE status = 'suspenso'`).Scan(&d.TokensSuspensos)   //nolint
		db.QueryRow(`
			SELECT COUNT(*) FROM financeiro.tokens
			WHERE status = 'ativo' AND valid_until <= CURRENT_DATE + INTERVAL '15 days'
		`).Scan(&d.TokensProximosVencimento) //nolint

		json.NewEncoder(w).Encode(d)
	}
}
