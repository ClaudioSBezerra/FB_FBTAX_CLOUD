package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type ClienteRequest struct {
	ID          string `json:"id,omitempty"`
	RazaoSocial string `json:"razao_social"`
	CNPJ        string `json:"cnpj"`
	Email       string `json:"email,omitempty"`
	Telefone    string `json:"telefone,omitempty"`
	Responsavel string `json:"responsavel,omitempty"`
	Ativo       bool   `json:"ativo"`
	CEP         string `json:"cep,omitempty"`
	Logradouro  string `json:"logradouro,omitempty"`
	Numero      string `json:"numero,omitempty"`
	Complemento string `json:"complemento,omitempty"`
	Bairro      string `json:"bairro,omitempty"`
	Municipio   string `json:"municipio,omitempty"`
	UF          string `json:"uf,omitempty"`
}

type ClienteCNPJResponse struct {
	ID          string `json:"id"`
	ClienteID   string `json:"cliente_id"`
	CNPJ        string `json:"cnpj"`
	Descricao   string `json:"descricao,omitempty"`
	IsPrincipal bool   `json:"is_principal"`
}

type ClienteResponse struct {
	ID          string                `json:"id"`
	RazaoSocial string                `json:"razao_social"`
	CNPJ        string                `json:"cnpj"`
	Email       string                `json:"email,omitempty"`
	Telefone    string                `json:"telefone,omitempty"`
	Responsavel string                `json:"responsavel,omitempty"`
	Ativo       bool                  `json:"ativo"`
	CreatedAt   string                `json:"created_at"`
	CEP         string                `json:"cep,omitempty"`
	Logradouro  string                `json:"logradouro,omitempty"`
	Numero      string                `json:"numero,omitempty"`
	Complemento string                `json:"complemento,omitempty"`
	Bairro      string                `json:"bairro,omitempty"`
	Municipio   string                `json:"municipio,omitempty"`
	UF          string                `json:"uf,omitempty"`
	Cnpjs       []ClienteCNPJResponse `json:"cnpjs,omitempty"`
}

type ClienteCNPJRequest struct {
	ID          string `json:"id,omitempty"`
	ClienteID   string `json:"cliente_id"`
	CNPJ        string `json:"cnpj"`
	Descricao   string `json:"descricao,omitempty"`
	IsPrincipal bool   `json:"is_principal"`
}

func ClientesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		action := r.URL.Query().Get("action")
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("id") != "" {
				handleGetCliente(w, r, db)
			} else {
				handleListClientes(w, r, db)
			}
		case http.MethodPost:
			if action == "add-cnpj" {
				handleAddCNPJ(w, r, db)
			} else {
				handlePostCliente(w, r, db)
			}
		case http.MethodPut:
			handlePutCliente(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleListClientes(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")

	query := `SELECT DISTINCT c.id, c.razao_social, c.cnpj,
		COALESCE(c.email,''), COALESCE(c.telefone,''), COALESCE(c.responsavel,''),
		c.ativo, c.created_at,
		COALESCE(c.cep,''), COALESCE(c.logradouro,''), COALESCE(c.numero,''),
		COALESCE(c.complemento,''), COALESCE(c.bairro,''), COALESCE(c.municipio,''),
		COALESCE(c.uf,'')
	FROM financeiro.clientes c
	LEFT JOIN financeiro.cliente_cnpjs cc ON cc.cliente_id = c.id
	WHERE 1=1`
	args := []interface{}{}
	argN := 1

	if q != "" {
		query += fmt.Sprintf(` AND (c.razao_social ILIKE $%d OR c.cnpj ILIKE $%d OR cc.cnpj ILIKE $%d)`, argN, argN, argN)
		args = append(args, "%"+q+"%")
		argN++
	}
	if status == "ativo" {
		query += fmt.Sprintf(` AND c.ativo = $%d`, argN)
		args = append(args, true)
		argN++
	} else if status == "inativo" {
		query += fmt.Sprintf(` AND c.ativo = $%d`, argN)
		args = append(args, false)
		argN++
	}
	query += ` ORDER BY c.razao_social`

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	clientes := []ClienteResponse{}
	for rows.Next() {
		var c ClienteResponse
		if err := rows.Scan(
			&c.ID, &c.RazaoSocial, &c.CNPJ, &c.Email, &c.Telefone, &c.Responsavel,
			&c.Ativo, &c.CreatedAt,
			&c.CEP, &c.Logradouro, &c.Numero, &c.Complemento, &c.Bairro, &c.Municipio, &c.UF,
		); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		clientes = append(clientes, c)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(clientes)
}

func handleGetCliente(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	id := r.URL.Query().Get("id")
	var c ClienteResponse
	var email, telefone, responsavel, cep, logradouro, numero, complemento, bairro, municipio, uf sql.NullString
	err := db.QueryRow(`
		SELECT id, razao_social, cnpj, email, telefone, responsavel, ativo, created_at,
		       cep, logradouro, numero, complemento, bairro, municipio, uf
		FROM financeiro.clientes WHERE id = $1
	`, id).Scan(
		&c.ID, &c.RazaoSocial, &c.CNPJ, &email, &telefone, &responsavel, &c.Ativo, &c.CreatedAt,
		&cep, &logradouro, &numero, &complemento, &bairro, &municipio, &uf,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "cliente not found"})
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	c.Email = email.String
	c.Telefone = telefone.String
	c.Responsavel = responsavel.String
	c.CEP = cep.String
	c.Logradouro = logradouro.String
	c.Numero = numero.String
	c.Complemento = complemento.String
	c.Bairro = bairro.String
	c.Municipio = municipio.String
	c.UF = uf.String

	rows, err := db.Query(`
		SELECT id, cliente_id, cnpj, COALESCE(descricao,''), is_principal
		FROM financeiro.cliente_cnpjs WHERE cliente_id = $1 ORDER BY is_principal DESC
	`, id)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	c.Cnpjs = []ClienteCNPJResponse{}
	for rows.Next() {
		var cc ClienteCNPJResponse
		if err := rows.Scan(&cc.ID, &cc.ClienteID, &cc.CNPJ, &cc.Descricao, &cc.IsPrincipal); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		c.Cnpjs = append(c.Cnpjs, cc)
	}
	json.NewEncoder(w).Encode(c)
}

func handlePostCliente(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req ClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.RazaoSocial == "" || req.CNPJ == "" {
		http.Error(w, "razao_social and cnpj are required", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRow(`
		INSERT INTO financeiro.clientes
		(razao_social, cnpj, email, telefone, responsavel, ativo,
		 cep, logradouro, numero, complemento, bairro, municipio, uf)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id
	`, req.RazaoSocial, req.CNPJ, req.Email, req.Telefone, req.Responsavel, req.Ativo,
		req.CEP, req.Logradouro, req.Numero, req.Complemento, req.Bairro, req.Municipio, req.UF,
	).Scan(&id)
	if err != nil {
		http.Error(w, "error creating cliente", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
		INSERT INTO financeiro.cliente_cnpjs (cliente_id, cnpj, descricao, is_principal)
		VALUES ($1, $2, $3, $4)
	`, id, req.CNPJ, "CNPJ Principal", true)
	if err != nil {
		http.Error(w, "error creating cliente cnpj", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("[Clientes] Cliente %s criado: %s", id, req.RazaoSocial)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func handlePutCliente(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req ClienteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`
		UPDATE financeiro.clientes
		SET razao_social=$1, email=$2, telefone=$3, responsavel=$4, ativo=$5,
		    cep=$6, logradouro=$7, numero=$8, complemento=$9, bairro=$10, municipio=$11, uf=$12,
		    updated_at=CURRENT_TIMESTAMP
		WHERE id=$13
	`, req.RazaoSocial, req.Email, req.Telefone, req.Responsavel, req.Ativo,
		req.CEP, req.Logradouro, req.Numero, req.Complemento, req.Bairro, req.Municipio, req.UF,
		req.ID,
	)
	if err != nil {
		http.Error(w, "error updating cliente", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func handleAddCNPJ(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req ClienteCNPJRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.ClienteID == "" || req.CNPJ == "" {
		http.Error(w, "cliente_id and cnpj are required", http.StatusBadRequest)
		return
	}
	var id string
	err := db.QueryRow(`
		INSERT INTO financeiro.cliente_cnpjs (cliente_id, cnpj, descricao, is_principal)
		VALUES ($1, $2, $3, $4) RETURNING id
	`, req.ClienteID, req.CNPJ, req.Descricao, false).Scan(&id)
	if err != nil {
		http.Error(w, "error adding cnpj", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}
