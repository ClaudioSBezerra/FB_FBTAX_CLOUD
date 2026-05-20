package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

type Banco struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

var (
	bancosCache   []Banco
	bancosCachedAt time.Time
	bancosMu      sync.Mutex
)

func BancosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		bancosMu.Lock()
		defer bancosMu.Unlock()

		if bancosCache != nil && time.Since(bancosCachedAt) < 24*time.Hour {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(bancosCache)
			return
		}

		resp, err := http.Get("https://brasilapi.com.br/api/banks/v1")
		if err != nil {
			http.Error(w, "error fetching banks", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		var raw []struct {
			Code *int   `json:"code"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			http.Error(w, "error parsing banks", http.StatusBadGateway)
			return
		}

		bancos := make([]Banco, 0, len(raw))
		for _, b := range raw {
			if b.Code != nil && *b.Code > 0 && b.Name != "" {
				bancos = append(bancos, Banco{
					Code: *b.Code,
					Name: b.Name,
				})
			}
		}
		sort.Slice(bancos, func(i, j int) bool { return bancos[i].Code < bancos[j].Code })

		bancosCache = bancos
		bancosCachedAt = time.Now()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bancos)
	}
}

func FormatBanco(code int, name string) string {
	return fmt.Sprintf("%03d - %s", code, name)
}
