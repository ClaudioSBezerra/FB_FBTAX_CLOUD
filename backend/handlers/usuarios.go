package handlers

// usuarios.go — administração de usuários do painel (/admin/financeiro/usuarios).
//
// Escopo: um administrador só vê e mexe em usuários dos MESMOS ambientes que ele
// (user_environments). Usuário novo entra no ambiente de quem o criou — nunca ganha
// ambiente/empresa próprios (o auto-provisionamento do login só roda para quem não tem vínculo).
//
// Perfil: todas as telas do painel exigem role 'admin' (main.go), então o usuário criado aqui é
// administrador. Não há perfil restrito enquanto não existir tela que um não-admin possa usar.

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"fb_cloud/services"
)

// Validade do link de convite (criar a primeira senha). O "esqueci minha senha" segue com 1 hora.
const conviteValidade = 72 * time.Hour

type UsuarioPainel struct {
	ID       string    `json:"id"`
	Nome     string    `json:"nome"`
	Email    string    `json:"email"`
	Perfil   string    `json:"perfil"`
	CriadoEm time.Time `json:"criado_em"`
	Voce     bool      `json:"voce"`
}

type novoUsuarioReq struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Senha string `json:"senha"` // opcional: vazio = envia convite por e-mail
}

// UsuariosHandler: GET lista, POST cria, DELETE ?id= remove.
func UsuariosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eu := GetUserIDFromContext(r)
		if eu == "" {
			erroJSONUsuarios(w, http.StatusUnauthorized, "não autenticado")
			return
		}
		switch r.Method {
		case http.MethodGet:
			listarUsuarios(db, w, eu)
		case http.MethodPost:
			criarUsuario(db, w, r, eu)
		case http.MethodDelete:
			removerUsuario(db, w, r, eu)
		default:
			erroJSONUsuarios(w, http.StatusMethodNotAllowed, "método não permitido")
		}
	}
}

// UsuarioConviteHandler: POST ?id= reenvia o link para o usuário criar/trocar a senha.
func UsuarioConviteHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			erroJSONUsuarios(w, http.StatusMethodNotAllowed, "método não permitido")
			return
		}
		eu := GetUserIDFromContext(r)
		alvo := r.URL.Query().Get("id")
		if eu == "" || alvo == "" {
			erroJSONUsuarios(w, http.StatusBadRequest, "usuário não informado")
			return
		}
		var nome, email string
		if err := db.QueryRow(`SELECT u.full_name, u.email FROM users u WHERE u.id = $1 AND `+mesmoAmbiente("u.id", "$2"), alvo, eu).
			Scan(&nome, &email); err == sql.ErrNoRows {
			erroJSONUsuarios(w, http.StatusNotFound, "usuário não encontrado")
			return
		} else if err != nil {
			log.Printf("[Usuarios] convite: %v", err)
			erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao buscar usuário")
			return
		}
		if err := enviarConvite(db, alvo, nome, email); err != nil {
			log.Printf("[Usuarios] convite para %s: %v", email, err)
			erroJSONUsuarios(w, http.StatusBadGateway, "não foi possível enviar o e-mail de convite")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "convite enviado para " + email})
	}
}

// mesmoAmbiente: condição SQL "o usuário da coluna col compartilha algum ambiente com o usuário $param".
func mesmoAmbiente(col, param string) string {
	return `EXISTS (SELECT 1 FROM user_environments a JOIN user_environments b ON b.environment_id = a.environment_id
		WHERE a.user_id = ` + col + ` AND b.user_id = ` + param + `)`
}

func listarUsuarios(db *sql.DB, w http.ResponseWriter, eu string) {
	rows, err := db.Query(`
		SELECT u.id, COALESCE(u.full_name, ''), u.email, COALESCE(u.role, 'user'), u.created_at
		FROM users u
		WHERE u.id = $1 OR `+mesmoAmbiente("u.id", "$1")+`
		ORDER BY u.created_at`, eu)
	if err != nil {
		log.Printf("[Usuarios] listar: %v", err)
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao listar usuários")
		return
	}
	defer rows.Close()
	lista := []UsuarioPainel{}
	for rows.Next() {
		var u UsuarioPainel
		if err := rows.Scan(&u.ID, &u.Nome, &u.Email, &u.Perfil, &u.CriadoEm); err != nil {
			log.Printf("[Usuarios] listar scan: %v", err)
			erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao listar usuários")
			return
		}
		u.Voce = u.ID == eu
		lista = append(lista, u)
	}
	if err := rows.Err(); err != nil {
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao listar usuários")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lista)
}

// validarNovoUsuario normaliza e valida o pedido; devolve a mensagem de erro para o usuário.
func validarNovoUsuario(req *novoUsuarioReq) string {
	req.Nome = strings.TrimSpace(req.Nome)
	req.Email = normalizarEmail(req.Email)
	if req.Nome == "" {
		return "informe o nome"
	}
	if a, err := mail.ParseAddress(req.Email); err != nil || a.Address != req.Email || !strings.Contains(req.Email, ".") {
		return "e-mail inválido"
	}
	if req.Senha != "" && len(req.Senha) < 8 {
		return "a senha inicial precisa ter pelo menos 8 caracteres"
	}
	return ""
}

func criarUsuario(db *sql.DB, w http.ResponseWriter, r *http.Request, eu string) {
	var req novoUsuarioReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		erroJSONUsuarios(w, http.StatusBadRequest, "requisição inválida")
		return
	}
	if msg := validarNovoUsuario(&req); msg != "" {
		erroJSONUsuarios(w, http.StatusBadRequest, msg)
		return
	}

	// Ambiente de quem está criando: o novo usuário entra nele (o mais antigo, se houver mais de um).
	var ambiente string
	if err := db.QueryRow(`SELECT environment_id FROM user_environments WHERE user_id = $1 ORDER BY created_at LIMIT 1`, eu).
		Scan(&ambiente); err == sql.ErrNoRows {
		erroJSONUsuarios(w, http.StatusConflict, "seu usuário não está vinculado a um ambiente")
		return
	} else if err != nil {
		log.Printf("[Usuarios] ambiente: %v", err)
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}

	// Sem senha informada, grava uma aleatória que ninguém conhece: o acesso só nasce pelo convite.
	senha := req.Senha
	if senha == "" {
		b := make([]byte, 24)
		if _, err := rand.Read(b); err != nil {
			erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao criar usuário")
			return
		}
		senha = hex.EncodeToString(b)
	}
	hash, err := HashPassword(senha)
	if err != nil {
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}
	defer tx.Rollback()

	var existe bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM users WHERE LOWER(email) = $1)`, req.Email).Scan(&existe); err != nil {
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}
	if existe {
		erroJSONUsuarios(w, http.StatusConflict, "já existe um usuário com este e-mail")
		return
	}
	var id string
	// trial_ends_at distante: o login barra trial vencido para quem não é admin; aqui é acesso definitivo.
	if err := tx.QueryRow(`
		INSERT INTO users (email, password_hash, full_name, trial_ends_at, is_verified, role)
		VALUES ($1, $2, $3, '2099-12-31 23:59:59+00', true, 'admin') RETURNING id`,
		req.Email, hash, req.Nome).Scan(&id); err != nil {
		log.Printf("[Usuarios] criar %s: %v", req.Email, err)
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}
	if _, err := tx.Exec(`INSERT INTO user_environments (user_id, environment_id, role) VALUES ($1, $2, 'admin')`, id, ambiente); err != nil {
		log.Printf("[Usuarios] vincular %s ao ambiente: %v", req.Email, err)
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}
	if err := tx.Commit(); err != nil {
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao criar usuário")
		return
	}
	log.Printf("[Usuarios] %s criou o usuário %s (%s)", eu, req.Email, id)

	resp := map[string]any{"id": id, "convite_enviado": false}
	if req.Senha == "" {
		if err := enviarConvite(db, id, req.Nome, req.Email); err != nil {
			// O usuário já existe: a tela avisa e oferece "Reenviar convite".
			log.Printf("[Usuarios] convite para %s: %v", req.Email, err)
			resp["aviso"] = "usuário criado, mas o e-mail de convite não foi enviado — use \"Reenviar convite\""
		} else {
			resp["convite_enviado"] = true
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

func removerUsuario(db *sql.DB, w http.ResponseWriter, r *http.Request, eu string) {
	alvo := r.URL.Query().Get("id")
	if alvo == "" {
		erroJSONUsuarios(w, http.StatusBadRequest, "usuário não informado")
		return
	}
	if alvo == eu {
		erroJSONUsuarios(w, http.StatusBadRequest, "você não pode remover o seu próprio usuário")
		return
	}
	var donoDeEmpresa bool
	err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM companies WHERE owner_id = u.id) FROM users u WHERE u.id = $1 AND `+mesmoAmbiente("u.id", "$2"), alvo, eu).
		Scan(&donoDeEmpresa)
	if err == sql.ErrNoRows {
		erroJSONUsuarios(w, http.StatusNotFound, "usuário não encontrado")
		return
	} else if err != nil {
		log.Printf("[Usuarios] remover: %v", err)
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao remover usuário")
		return
	}
	if donoDeEmpresa {
		erroJSONUsuarios(w, http.StatusConflict, "este usuário é o titular de uma empresa e não pode ser removido por aqui")
		return
	}
	if _, err := db.Exec(`DELETE FROM users WHERE id = $1`, alvo); err != nil {
		log.Printf("[Usuarios] remover %s: %v", alvo, err)
		erroJSONUsuarios(w, http.StatusInternalServerError, "erro ao remover usuário")
		return
	}
	log.Printf("[Usuarios] %s removeu o usuário %s", eu, alvo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "usuário removido"})
}

// enviarConvite gera um token de redefinição com validade de convite e manda o e-mail.
func enviarConvite(db *sql.DB, userID, nome, email string) error {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := hex.EncodeToString(b)
	if _, err := db.Exec(`INSERT INTO verification_tokens (user_id, token, type, expires_at) VALUES ($1, $2, 'password_reset', $3)`,
		userID, token, time.Now().Add(conviteValidade)); err != nil {
		return err
	}
	return services.SendConviteEmail(email, nome, token, int(conviteValidade.Hours()))
}

// erroJSONUsuarios responde {"error": msg} com o status dado (a tela mostra o texto como está).
func erroJSONUsuarios(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
