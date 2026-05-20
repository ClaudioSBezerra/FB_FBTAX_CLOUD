package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type EmpresaRequest struct {
	ID           string `json:"id,omitempty"`
	RazaoSocial  string `json:"razao_social"`
	NomeFantasia string `json:"nome_fantasia,omitempty"`
	CNPJ         string `json:"cnpj"`
	Logradouro   string `json:"logradouro"`
	Numero       string `json:"numero"`
	Complemento  string `json:"complemento,omitempty"`
	Bairro       string `json:"bairro"`
	CEP          string `json:"cep"`
	Municipio    string `json:"municipio"`
	UF           string `json:"uf"`
}

type DadosBancariosRequest struct {
	ID        string `json:"id,omitempty"`
	EmpresaID string `json:"empresa_id"`
	Banco     string `json:"banco"`
	Agencia   string `json:"agencia"`
	Conta     string `json:"conta"`
	TipoConta string `json:"tipo_conta"`
	Titular   string `json:"titular,omitempty"`
}

func EmpresaHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleGetEmpresa(w, r, db)
		case http.MethodPost:
			handlePostEmpresa(w, r, db)
		case http.MethodPut:
			handlePutEmpresa(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleGetEmpresa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var emp EmpresaRequest
	var nomeFantasia, complemento sql.NullString
	err := db.QueryRow(`
		SELECT id, razao_social, nome_fantasia, cnpj, logradouro, numero,
		       complemento, bairro, cep, municipio, uf
		FROM financeiro.empresas
		LIMIT 1
	`).Scan(
		&emp.ID, &emp.RazaoSocial, &nomeFantasia, &emp.CNPJ,
		&emp.Logradouro, &emp.Numero, &complemento, &emp.Bairro,
		&emp.CEP, &emp.Municipio, &emp.UF,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "empresa not found"})
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	emp.NomeFantasia = nomeFantasia.String
	emp.Complemento = complemento.String
	json.NewEncoder(w).Encode(emp)
}

func handlePostEmpresa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req EmpresaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.RazaoSocial == "" || req.CNPJ == "" {
		http.Error(w, "razao_social and cnpj are required", http.StatusBadRequest)
		return
	}
	var id string
	err := db.QueryRow(`
		INSERT INTO financeiro.empresas
		(razao_social, nome_fantasia, cnpj, logradouro, numero, complemento, bairro, cep, municipio, uf)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, req.RazaoSocial, req.NomeFantasia, req.CNPJ, req.Logradouro, req.Numero,
		req.Complemento, req.Bairro, req.CEP, req.Municipio, req.UF,
	).Scan(&id)
	if err != nil {
		http.Error(w, "error creating empresa", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func handlePutEmpresa(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req EmpresaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`
		UPDATE financeiro.empresas
		SET razao_social = $1, nome_fantasia = $2, cnpj = $3, logradouro = $4,
		    numero = $5, complemento = $6, bairro = $7, cep = $8, municipio = $9,
		    uf = $10, updated_at = CURRENT_TIMESTAMP
		WHERE id = $11
	`, req.RazaoSocial, req.NomeFantasia, req.CNPJ, req.Logradouro, req.Numero,
		req.Complemento, req.Bairro, req.CEP, req.Municipio, req.UF, req.ID,
	)
	if err != nil {
		http.Error(w, "error updating empresa", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func DadosBancariosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleGetDadosBancarios(w, r, db)
		case http.MethodPost:
			handlePostDadosBancarios(w, r, db)
		case http.MethodPut:
			handlePutDadosBancarios(w, r, db)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleGetDadosBancarios(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var dados DadosBancariosRequest
	var titular sql.NullString
	err := db.QueryRow(`
		SELECT id, empresa_id, banco, agencia, conta, tipo_conta, titular
		FROM financeiro.dados_bancarios
		LIMIT 1
	`).Scan(
		&dados.ID, &dados.EmpresaID, &dados.Banco, &dados.Agencia,
		&dados.Conta, &dados.TipoConta, &titular,
	)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "dados_bancarios not found"})
		return
	}
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	dados.Titular = titular.String
	json.NewEncoder(w).Encode(dados)
}

func handlePostDadosBancarios(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req DadosBancariosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.EmpresaID == "" || req.Banco == "" || req.Agencia == "" || req.Conta == "" || req.TipoConta == "" {
		http.Error(w, "empresa_id, banco, agencia, conta and tipo_conta are required", http.StatusBadRequest)
		return
	}
	var id string
	err := db.QueryRow(`
		INSERT INTO financeiro.dados_bancarios
		(empresa_id, banco, agencia, conta, tipo_conta, titular)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, req.EmpresaID, req.Banco, req.Agencia, req.Conta, req.TipoConta, req.Titular,
	).Scan(&id)
	if err != nil {
		http.Error(w, "error creating dados_bancarios", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func handlePutDadosBancarios(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req DadosBancariosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	_, err := db.Exec(`
		UPDATE financeiro.dados_bancarios
		SET empresa_id = $1, banco = $2, agencia = $3, conta = $4,
		    tipo_conta = $5, titular = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
	`, req.EmpresaID, req.Banco, req.Agencia, req.Conta, req.TipoConta, req.Titular, req.ID,
	)
	if err != nil {
		http.Error(w, "error updating dados_bancarios", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
