package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ── Auth portal cliente ───────────────────────────────────────────────────────

func PortalLoginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
			http.Error(w, "email e senha obrigatórios", http.StatusBadRequest)
			return
		}

		var id, clienteID, hash string
		var ativo bool
		err := db.QueryRow(`
			SELECT id, cliente_id, password_hash, ativo
			FROM financeiro.portal_clientes WHERE email = $1
		`, req.Email).Scan(&id, &clienteID, &hash, &ativo)
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "credenciais inválidas"})
			return
		}
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		if !ativo {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "acesso inativo"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "credenciais inválidas"})
			return
		}

		token, err := GeneratePortalToken(id, clienteID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("[Portal] Login cliente %s (id=%s)", req.Email, clienteID)
		json.NewEncoder(w).Encode(map[string]string{"token": token, "cliente_id": clienteID})
	}
}

// ── Gerenciamento de usuários do portal (admin) ───────────────────────────────

func PortalClientesAdminHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleListPortalClientes(w, r, db)
		case http.MethodPost:
			handleCreatePortalCliente(w, r, db)
		case http.MethodPut:
			handleUpdatePortalCliente(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleListPortalClientes(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	rows, err := db.Query(`
		SELECT pc.id, pc.cliente_id, pc.email, pc.ativo, pc.created_at::text,
		       c.razao_social
		FROM financeiro.portal_clientes pc
		JOIN financeiro.clientes c ON c.id = pc.cliente_id
		ORDER BY c.razao_social
	`)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type PortalClienteRow struct {
		ID          string `json:"id"`
		ClienteID   string `json:"cliente_id"`
		Email       string `json:"email"`
		Ativo       bool   `json:"ativo"`
		CreatedAt   string `json:"created_at"`
		RazaoSocial string `json:"razao_social"`
	}
	list := []PortalClienteRow{}
	for rows.Next() {
		var p PortalClienteRow
		if err := rows.Scan(&p.ID, &p.ClienteID, &p.Email, &p.Ativo, &p.CreatedAt, &p.RazaoSocial); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		list = append(list, p)
	}
	json.NewEncoder(w).Encode(list)
}

func handleCreatePortalCliente(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ClienteID string `json:"cliente_id"`
		Email     string `json:"email"`
		Password  string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil ||
		req.ClienteID == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "cliente_id, email e password obrigatórios", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var id string
	err = db.QueryRow(`
		INSERT INTO financeiro.portal_clientes (cliente_id, email, password_hash)
		VALUES ($1, $2, $3) RETURNING id
	`, req.ClienteID, req.Email, string(hash)).Scan(&id)
	if err != nil {
		http.Error(w, "error creating portal cliente", http.StatusInternalServerError)
		return
	}

	log.Printf("[Portal] Acesso criado para cliente %s email %s", req.ClienteID, req.Email)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func handleUpdatePortalCliente(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		ID       string  `json:"id"`
		Ativo    bool    `json:"ativo"`
		Password *string `json:"password,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		db.Exec(`UPDATE financeiro.portal_clientes SET password_hash = $1, updated_at = NOW() WHERE id = $2`, string(hash), req.ID) //nolint
	}
	db.Exec(`UPDATE financeiro.portal_clientes SET ativo = $1, updated_at = NOW() WHERE id = $2`, req.Ativo, req.ID) //nolint
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// ── Endpoints do portal do cliente (JWT role=portal_cliente) ─────────────────

func PortalMeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clienteID, ok := portalClienteIDFromCtx(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var c struct {
			ID          string `json:"id"`
			RazaoSocial string `json:"razao_social"`
			CNPJ        string `json:"cnpj"`
			Email       string `json:"email"`
		}
		err := db.QueryRow(`
			SELECT id, razao_social, cnpj, COALESCE(email,'')
			FROM financeiro.clientes WHERE id = $1
		`, clienteID).Scan(&c.ID, &c.RazaoSocial, &c.CNPJ, &c.Email)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(c)
	}
}

func PortalContratosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clienteID, ok := portalClienteIDFromCtx(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			SELECT id, data_inicio::text, periodicidade, valor_total, status, COALESCE(observacoes,''), created_at::text
			FROM financeiro.contratos
			WHERE cliente_id = $1
			ORDER BY created_at DESC
		`, clienteID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		contratos := []ContratoResponse{}
		for rows.Next() {
			var c ContratoResponse
			if err := rows.Scan(&c.ID, &c.DataInicio, &c.Periodicidade,
				&c.ValorTotal, &c.Status, &c.Observacoes, &c.CreatedAt); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			c.ClienteID = clienteID
			itens, _ := fetchContratoItens(db, c.ID)
			c.Itens = itens
			contratos = append(contratos, c)
		}
		json.NewEncoder(w).Encode(contratos)
	}
}

func PortalTokensHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		clienteID, ok := portalClienteIDFromCtx(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
			SELECT t.id, t.contrato_id, t.token, t.status,
			       t.valid_from::text, t.valid_until::text,
			       pl.nome, p.nome, p.codigo
			FROM financeiro.tokens t
			JOIN financeiro.contratos ct ON ct.id = t.contrato_id
			JOIN financeiro.planos pl    ON pl.id = t.plano_id
			JOIN financeiro.produtos p   ON p.id  = pl.produto_id
			WHERE ct.cliente_id = $1
			  AND t.status IN ('ativo','em_carencia','suspenso')
			ORDER BY t.status, t.valid_until
		`, clienteID)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type PortalToken struct {
			ID           string `json:"id"`
			ContratoID   string `json:"contrato_id"`
			Token        string `json:"token"`
			Status       string `json:"status"`
			ValidFrom    string `json:"valid_from"`
			ValidUntil   string `json:"valid_until"`
			PlanoNome    string `json:"plano_nome"`
			ProdutoNome  string `json:"produto_nome"`
			ProdutoCodigo string `json:"produto_codigo"`
		}
		tokens := []PortalToken{}
		for rows.Next() {
			var t PortalToken
			if err := rows.Scan(&t.ID, &t.ContratoID, &t.Token, &t.Status,
				&t.ValidFrom, &t.ValidUntil, &t.PlanoNome, &t.ProdutoNome, &t.ProdutoCodigo); err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			tokens = append(tokens, t)
		}
		json.NewEncoder(w).Encode(tokens)
	}
}

// ── JWT portal (role = portal_cliente) ───────────────────────────────────────

func GeneratePortalToken(portalClienteID, clienteID string) (string, error) {
	return generateJWTWithClaims(map[string]interface{}{
		"sub":               portalClienteID,
		"role":              "portal_cliente",
		"cliente_id":        clienteID,
		"exp":               time.Now().Add(24 * time.Hour).Unix(),
	})
}

func portalClienteIDFromCtx(r *http.Request) (string, bool) {
	claims, ok := getPortalClaims(r)
	if !ok {
		return "", false
	}
	clienteID, ok := claims["cliente_id"].(string)
	return clienteID, ok && clienteID != ""
}
