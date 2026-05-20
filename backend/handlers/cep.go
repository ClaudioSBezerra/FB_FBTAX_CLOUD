package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

var reCEP = regexp.MustCompile(`^\d{8}$`)

func CEPProxyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cep := r.URL.Query().Get("cep")
		if !reCEP.MatchString(cep) {
			http.Error(w, "cep must be 8 digits", http.StatusBadRequest)
			return
		}

		resp, err := http.Get(fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep))
		if err != nil {
			http.Error(w, "error contacting viacep", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "error reading viacep response", http.StatusBadGateway)
			return
		}

		// ViaCEP retorna {"erro": true} para CEPs não encontrados
		var check map[string]interface{}
		if err := json.Unmarshal(body, &check); err != nil {
			http.Error(w, "invalid viacep response", http.StatusBadGateway)
			return
		}
		if check["erro"] != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"cep not found"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}
}
