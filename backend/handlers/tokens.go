package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"fb_cloud/services"
)

type TokenResponse struct {
	ID            string  `json:"id"`
	ContratoID    string  `json:"contrato_id"`
	PlanoID       string  `json:"plano_id"`
	PlanoNome     string  `json:"plano_nome"`
	ProdutoNome   string  `json:"produto_nome"`
	Token         string  `json:"token"`
	Status        string  `json:"status"`
	ValidFrom     string  `json:"valid_from"`
	ValidUntil    string  `json:"valid_until"`
	AlertaEnviado bool    `json:"alerta_enviado"`
	PredecessorID *string `json:"predecessor_id,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

func gerarTokenStr() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "") +
		strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
}

func TokensHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleGetTokens(w, r, db)
		case http.MethodPost:
			handleReativarToken(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleGetTokens(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	contratoID := r.URL.Query().Get("contrato_id")
	if contratoID == "" {
		http.Error(w, "contrato_id required", http.StatusBadRequest)
		return
	}
	rows, err := db.Query(`
		SELECT t.id, t.contrato_id, t.plano_id,
		       pl.nome, p.nome,
		       t.token, t.status,
		       t.valid_from::text, t.valid_until::text,
		       t.alerta_enviado, t.predecessor_id, t.created_at::text
		FROM financeiro.tokens t
		JOIN financeiro.planos pl ON pl.id = t.plano_id
		JOIN financeiro.produtos p ON p.id = pl.produto_id
		WHERE t.contrato_id = $1
		ORDER BY t.created_at DESC
	`, contratoID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tokens := []TokenResponse{}
	for rows.Next() {
		var t TokenResponse
		var predID sql.NullString
		if err := rows.Scan(&t.ID, &t.ContratoID, &t.PlanoID,
			&t.PlanoNome, &t.ProdutoNome,
			&t.Token, &t.Status, &t.ValidFrom, &t.ValidUntil,
			&t.AlertaEnviado, &predID, &t.CreatedAt); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if predID.Valid {
			t.PredecessorID = &predID.String
		}
		tokens = append(tokens, t)
	}
	json.NewEncoder(w).Encode(tokens)
}

func handleReativarToken(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		TokenID string `json:"token_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TokenID == "" {
		http.Error(w, "token_id required", http.StatusBadRequest)
		return
	}

	var contratoID, planoID string
	err := db.QueryRow(
		`SELECT contrato_id, plano_id FROM financeiro.tokens WHERE id = $1`, req.TokenID,
	).Scan(&contratoID, &planoID)
	if err == sql.ErrNoRows {
		http.Error(w, "token not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE financeiro.tokens SET status = 'encerrado', updated_at = NOW() WHERE id = $1`,
		req.TokenID,
	); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	novoToken := gerarTokenStr()
	validFrom := time.Now().Format("2006-01-02")
	validUntil := time.Now().AddDate(0, 0, 45).Format("2006-01-02")
	var newID string
	err = tx.QueryRow(`
		INSERT INTO financeiro.tokens (contrato_id, plano_id, token, status, valid_from, valid_until, predecessor_id)
		VALUES ($1, $2, $3, 'ativo', $4, $5, $6) RETURNING id
	`, contratoID, planoID, novoToken, validFrom, validUntil, req.TokenID).Scan(&newID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("[Tokens] Token %s reativado → novo %s", req.TokenID, newID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": newID, "token": novoToken})
}

// GerarTokensContrato insere tokens para cada plano de um contrato dentro de uma transação.
func GerarTokensContrato(tx *sql.Tx, contratoID, dataInicio string, planoIDs []string) error {
	validFrom, err := time.Parse("2006-01-02", dataInicio)
	if err != nil {
		validFrom = time.Now()
	}
	validUntil := validFrom.AddDate(0, 0, 45)

	for _, planoID := range planoIDs {
		if _, err := tx.Exec(`
			INSERT INTO financeiro.tokens (contrato_id, plano_id, token, valid_from, valid_until)
			VALUES ($1, $2, $3, $4, $5)
		`, contratoID, planoID, gerarTokenStr(),
			validFrom.Format("2006-01-02"),
			validUntil.Format("2006-01-02")); err != nil {
			return fmt.Errorf("token para plano %s: %w", planoID, err)
		}
	}
	return nil
}

// StartTokenStatusUpdater inicia goroutine horária de atualização de status e envio de alertas.
func StartTokenStatusUpdater(db *sql.DB) {
	go func() {
		runTokenUpdate(db)
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			runTokenUpdate(db)
		}
	}()
}

func runTokenUpdate(db *sql.DB) {
	if _, err := db.Exec(`
		UPDATE financeiro.tokens
		SET status = 'em_carencia', updated_at = NOW()
		WHERE status = 'ativo' AND valid_until < CURRENT_DATE
	`); err != nil {
		log.Printf("[Tokens] atualizar em_carencia: %v", err)
	}

	if _, err := db.Exec(`
		UPDATE financeiro.tokens
		SET status = 'suspenso', updated_at = NOW()
		WHERE status = 'em_carencia' AND valid_until + INTERVAL '15 days' < CURRENT_DATE
	`); err != nil {
		log.Printf("[Tokens] atualizar suspenso: %v", err)
	}

	adminEmail := os.Getenv("FINANCEIRO_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = os.Getenv("SMTP_USER")
	}

	rows, err := db.Query(`
		SELECT t.id, t.valid_until::text,
		       c.razao_social, COALESCE(c.email,''),
		       pl.nome, p.nome
		FROM financeiro.tokens t
		JOIN financeiro.contratos ct ON ct.id = t.contrato_id
		JOIN financeiro.clientes c   ON c.id  = ct.cliente_id
		JOIN financeiro.planos pl    ON pl.id = t.plano_id
		JOIN financeiro.produtos p   ON p.id  = pl.produto_id
		WHERE t.status = 'ativo'
		  AND t.alerta_enviado = false
		  AND t.valid_until <= CURRENT_DATE + INTERVAL '15 days'
	`)
	if err != nil {
		log.Printf("[Tokens] buscar alertas: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tokenID, validUntil, razao, clienteEmail, planoNome, prodNome string
		if err := rows.Scan(&tokenID, &validUntil, &razao, &clienteEmail, &planoNome, &prodNome); err != nil {
			continue
		}
		if adminEmail != "" {
			services.SendAlertaToken(adminEmail, razao, prodNome, planoNome, validUntil) //nolint
		}
		if clienteEmail != "" && clienteEmail != adminEmail {
			services.SendAlertaToken(clienteEmail, razao, prodNome, planoNome, validUntil) //nolint
		}
		db.Exec(`UPDATE financeiro.tokens SET alerta_enviado = true WHERE id = $1`, tokenID) //nolint
		log.Printf("[Tokens] Alerta vencimento enviado — token %s (vence %s)", tokenID, validUntil)
	}
}
