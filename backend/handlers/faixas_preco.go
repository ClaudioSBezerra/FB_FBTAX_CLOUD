package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
)

type FaixaPreco struct {
	ID        string   `json:"id"`
	ProdutoID *string  `json:"produto_id"`
	MBMin     float64  `json:"mb_min"`
	MBMax     *float64 `json:"mb_max"`
	Preco     *float64 `json:"preco"`
	Descricao string   `json:"descricao"`
	Ativo     bool     `json:"ativo"`
	Ordem     int      `json:"ordem"`
}

// GET  /api/financeiro/faixas-preco              → lista faixas (produto_id opcional via query)
// POST /api/financeiro/faixas-preco              → cria faixa
// PUT  /api/financeiro/faixas-preco              → atualiza faixa (id no body)
// DELETE /api/financeiro/faixas-preco?id=xxx     → remove faixa
func FaixasPrecoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleListFaixas(w, r, db)
		case http.MethodPost:
			handleCreateFaixa(w, r, db)
		case http.MethodPut:
			handleUpdateFaixa(w, r, db)
		case http.MethodDelete:
			handleDeleteFaixa(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleListFaixas(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	produtoID := r.URL.Query().Get("produto_id")

	var rows *sql.Rows
	var err error
	if produtoID != "" {
		rows, err = db.Query(`
			SELECT id, produto_id, mb_min, mb_max, preco, COALESCE(descricao,''), ativo, ordem
			FROM financeiro.faixas_preco
			WHERE (produto_id = $1 OR produto_id IS NULL) AND ativo = true
			ORDER BY ordem, mb_min`, produtoID)
	} else {
		rows, err = db.Query(`
			SELECT id, produto_id, mb_min, mb_max, preco, COALESCE(descricao,''), ativo, ordem
			FROM financeiro.faixas_preco
			WHERE produto_id IS NULL AND ativo = true
			ORDER BY ordem, mb_min`)
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var faixas []FaixaPreco
	for rows.Next() {
		var f FaixaPreco
		var pid sql.NullString
		var mbMax sql.NullFloat64
		var preco sql.NullFloat64
		rows.Scan(&f.ID, &pid, &f.MBMin, &mbMax, &preco, &f.Descricao, &f.Ativo, &f.Ordem)
		if pid.Valid { f.ProdutoID = &pid.String }
		if mbMax.Valid { f.MBMax = &mbMax.Float64 }
		if preco.Valid { f.Preco = &preco.Float64 }
		faixas = append(faixas, f)
	}
	if faixas == nil {
		faixas = []FaixaPreco{}
	}
	json.NewEncoder(w).Encode(faixas)
}

func handleCreateFaixa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var f FaixaPreco
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var id string
	err := db.QueryRow(`
		INSERT INTO financeiro.faixas_preco (produto_id, mb_min, mb_max, preco, descricao, ordem)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		f.ProdutoID, f.MBMin, f.MBMax, f.Preco, f.Descricao, f.Ordem,
	).Scan(&id)
	if err != nil {
		http.Error(w, "db error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func handleUpdateFaixa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var f FaixaPreco
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil || f.ID == "" {
		http.Error(w, "id obrigatório", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`
		UPDATE financeiro.faixas_preco
		SET mb_min = $1, mb_max = $2, preco = $3, descricao = $4, ordem = $5
		WHERE id = $6`,
		f.MBMin, f.MBMax, f.Preco, f.Descricao, f.Ordem, f.ID,
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"mensagem": "ok"})
}

func handleDeleteFaixa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id obrigatório", http.StatusBadRequest)
		return
	}
	db.Exec(`UPDATE financeiro.faixas_preco SET ativo = false WHERE id = $1`, id)
	json.NewEncoder(w).Encode(map[string]string{"mensagem": "ok"})
}

// GET /api/financeiro/faixas-preco/calcular?mb=150
// Retorna a faixa correspondente e o preço calculado.
func FaixasCalcularHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		q := r.URL.Query().Get("mb")
		if q == "" {
			http.Error(w, "mb obrigatório", http.StatusBadRequest)
			return
		}
		mb, err := strconv.ParseFloat(q, 64)
		if err != nil || mb < 0 {
			http.Error(w, "mb inválido", http.StatusBadRequest)
			return
		}

		row := db.QueryRow(`
			SELECT id, mb_min, mb_max, preco, COALESCE(descricao,'')
			FROM financeiro.faixas_preco
			WHERE ativo = true AND produto_id IS NULL
			  AND mb_min <= $1
			  AND (mb_max IS NULL OR mb_max > $1)
			ORDER BY mb_min DESC
			LIMIT 1`, mb)

		var f FaixaPreco
		var mbMax sql.NullFloat64
		var preco sql.NullFloat64
		if err := row.Scan(&f.ID, &f.MBMin, &mbMax, &preco, &f.Descricao); err != nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"faixa":       nil,
				"preco":       nil,
				"sob_consulta": true,
			})
			return
		}
		if mbMax.Valid { f.MBMax = &mbMax.Float64 }
		if preco.Valid { f.Preco = &preco.Float64 }

		json.NewEncoder(w).Encode(map[string]interface{}{
			"faixa":        f,
			"preco":        f.Preco,
			"sob_consulta": f.Preco == nil,
		})
	}
}
