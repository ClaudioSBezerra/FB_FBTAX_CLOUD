package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// ── Contas Financeiras ────────────────────────────────────────────────────────

type ContaFinanceiraRequest struct {
	Apelido    string  `json:"apelido"`
	Banco      string  `json:"banco"`
	Agencia    string  `json:"agencia,omitempty"`
	Conta      string  `json:"conta,omitempty"`
	Tipo       string  `json:"tipo"`
	Provedor   string  `json:"provedor,omitempty"`
	ProvedorID string  `json:"provedor_id,omitempty"`
	Saldo      float64 `json:"saldo"`
}

type ContaFinanceiraResponse struct {
	ID         string   `json:"id"`
	Apelido    string   `json:"apelido"`
	Banco      string   `json:"banco"`
	Agencia    string   `json:"agencia,omitempty"`
	Conta      string   `json:"conta,omitempty"`
	Tipo       string   `json:"tipo"`
	Provedor   string   `json:"provedor,omitempty"`
	ProvedorID string   `json:"provedor_id,omitempty"`
	Saldo      float64  `json:"saldo"`
	UltimaSync *string  `json:"ultima_sync,omitempty"`
	Ativa      bool     `json:"ativa"`
}

func ContasFinanceirasHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			rows, err := db.Query(`
				SELECT id, apelido, banco, COALESCE(agencia,''), COALESCE(conta,''),
				       tipo, COALESCE(provedor,''), COALESCE(provedor_id,''),
				       saldo, ultima_sync, ativa
				FROM financeiro.contas_financeiras
				WHERE ativa = true
				ORDER BY apelido
			`)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			contas := []ContaFinanceiraResponse{}
			for rows.Next() {
				var c ContaFinanceiraResponse
				var sync sql.NullTime
				if err := rows.Scan(&c.ID, &c.Apelido, &c.Banco, &c.Agencia, &c.Conta,
					&c.Tipo, &c.Provedor, &c.ProvedorID, &c.Saldo, &sync, &c.Ativa); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				if sync.Valid {
					s := sync.Time.Format(time.RFC3339)
					c.UltimaSync = &s
				}
				contas = append(contas, c)
			}
			json.NewEncoder(w).Encode(contas)

		case http.MethodPost:
			var req ContaFinanceiraRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if req.Apelido == "" || req.Banco == "" {
				http.Error(w, "apelido and banco are required", http.StatusBadRequest)
				return
			}
			if req.Tipo == "" {
				req.Tipo = "corrente"
			}
			var id string
			err := db.QueryRow(`
				INSERT INTO financeiro.contas_financeiras
				(apelido, banco, agencia, conta, tipo, provedor, provedor_id, saldo)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id
			`, req.Apelido, req.Banco, req.Agencia, req.Conta, req.Tipo,
				req.Provedor, req.ProvedorID, req.Saldo).Scan(&id)
			if err != nil {
				http.Error(w, "error creating conta", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": id})

		case http.MethodDelete:
			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, "id query param required", http.StatusBadRequest)
				return
			}
			res, err := db.Exec(`
				UPDATE financeiro.contas_financeiras
				SET ativa = false
				WHERE id = $1 AND ativa = true
			`, id)
			if err != nil {
				http.Error(w, "error deleting conta", http.StatusInternalServerError)
				return
			}
			n, _ := res.RowsAffected()
			if n == 0 {
				http.Error(w, "conta not found or already inactive", http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// ── Transações ────────────────────────────────────────────────────────────────

type TransacaoRequest struct {
	ContaID       string  `json:"conta_id"`
	DataTransacao string  `json:"data_transacao"`
	Descricao     string  `json:"descricao"`
	Valor         float64 `json:"valor"`
	Tipo          string  `json:"tipo"`
	Categoria     string  `json:"categoria,omitempty"`
	ReferenciaExt string  `json:"referencia_ext,omitempty"`
}

type TransacaoResponse struct {
	ID            string  `json:"id"`
	ContaID       string  `json:"conta_id"`
	ContaApelido  string  `json:"conta_apelido"`
	DataTransacao string  `json:"data_transacao"`
	Descricao     string  `json:"descricao"`
	Valor         float64 `json:"valor"`
	Tipo          string  `json:"tipo"`
	Categoria     string  `json:"categoria,omitempty"`
	Conciliado    bool    `json:"conciliado"`
}

func TransacoesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			limit := 50
			if l := r.URL.Query().Get("limit"); l != "" {
				if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
					limit = n
				}
			}
			contaID := r.URL.Query().Get("conta_id")

			query := `
				SELECT t.id, t.conta_id, cf.apelido, t.data_transacao,
				       t.descricao, t.valor, t.tipo,
				       COALESCE(t.categoria,''), t.conciliado
				FROM financeiro.transacoes t
				JOIN financeiro.contas_financeiras cf ON cf.id = t.conta_id
				WHERE 1=1`
			args := []interface{}{}
			argN := 1
			if contaID != "" {
				query += ` AND t.conta_id = $` + strconv.Itoa(argN)
				args = append(args, contaID)
				argN++
			}
			query += ` ORDER BY t.data_transacao DESC, t.created_at DESC LIMIT $` + strconv.Itoa(argN)
			args = append(args, limit)

			rows, err := db.Query(query, args...)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			defer rows.Close()
			transacoes := []TransacaoResponse{}
			for rows.Next() {
				var t TransacaoResponse
				var data time.Time
				if err := rows.Scan(&t.ID, &t.ContaID, &t.ContaApelido, &data,
					&t.Descricao, &t.Valor, &t.Tipo, &t.Categoria, &t.Conciliado); err != nil {
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				t.DataTransacao = data.Format("2006-01-02")
				transacoes = append(transacoes, t)
			}
			json.NewEncoder(w).Encode(transacoes)

		case http.MethodPost:
			var req TransacaoRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			if req.ContaID == "" || req.Descricao == "" || req.Valor == 0 ||
				(req.Tipo != "credito" && req.Tipo != "debito") {
				http.Error(w, "conta_id, descricao, valor and tipo (credito|debito) are required", http.StatusBadRequest)
				return
			}
			if req.DataTransacao == "" {
				req.DataTransacao = time.Now().Format("2006-01-02")
			}
			var id string
			err := db.QueryRow(`
				INSERT INTO financeiro.transacoes
				(conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext)
				VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id
			`, req.ContaID, req.DataTransacao, req.Descricao, req.Valor,
				req.Tipo, req.Categoria, req.ReferenciaExt).Scan(&id)
			if err != nil {
				http.Error(w, "error creating transacao", http.StatusInternalServerError)
				return
			}
			// Atualiza saldo da conta
			sinalSaldo := 1.0
			if req.Tipo == "debito" {
				sinalSaldo = -1.0
			}
			db.Exec(`
				UPDATE financeiro.contas_financeiras
				SET saldo = saldo + $1, updated_at = CURRENT_TIMESTAMP
				WHERE id = $2
			`, req.Valor*sinalSaldo, req.ContaID)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": id})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// ── Painel — resumo e fluxo de caixa ─────────────────────────────────────────

type FluxoDia struct {
	Data    string  `json:"data"`
	Entrada float64 `json:"entrada"`
	Saida   float64 `json:"saida"`
}

type PainelResponse struct {
	SaldoTotal  float64    `json:"saldo_total"`
	Entradas30d float64    `json:"entradas_30d"`
	Saidas30d   float64    `json:"saidas_30d"`
	Resultado30d float64   `json:"resultado_30d"`
	NumContas   int        `json:"num_contas"`
	Fluxo       []FluxoDia `json:"fluxo"`
}

func PainelFinanceiroHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var resp PainelResponse

		// Saldo total e número de contas
		db.QueryRow(`
			SELECT COALESCE(SUM(saldo),0), COUNT(*)
			FROM financeiro.contas_financeiras WHERE ativa = true
		`).Scan(&resp.SaldoTotal, &resp.NumContas)

		// Entradas e saídas 30 dias
		db.QueryRow(`
			SELECT
				COALESCE(SUM(CASE WHEN tipo='credito' THEN valor ELSE 0 END),0),
				COALESCE(SUM(CASE WHEN tipo='debito'  THEN valor ELSE 0 END),0)
			FROM financeiro.transacoes
			WHERE data_transacao >= CURRENT_DATE - INTERVAL '30 days'
		`).Scan(&resp.Entradas30d, &resp.Saidas30d)
		resp.Resultado30d = resp.Entradas30d - resp.Saidas30d

		// Fluxo diário 30 dias
		rows, err := db.Query(`
			SELECT
				data_transacao::text,
				COALESCE(SUM(CASE WHEN tipo='credito' THEN valor ELSE 0 END),0) AS entrada,
				COALESCE(SUM(CASE WHEN tipo='debito'  THEN valor ELSE 0 END),0) AS saida
			FROM financeiro.transacoes
			WHERE data_transacao >= CURRENT_DATE - INTERVAL '30 days'
			GROUP BY data_transacao
			ORDER BY data_transacao
		`)
		if err == nil {
			defer rows.Close()
			resp.Fluxo = []FluxoDia{}
			for rows.Next() {
				var f FluxoDia
				rows.Scan(&f.Data, &f.Entrada, &f.Saida)
				resp.Fluxo = append(resp.Fluxo, f)
			}
		}
		if resp.Fluxo == nil {
			resp.Fluxo = []FluxoDia{}
		}

		json.NewEncoder(w).Encode(resp)
	}
}
