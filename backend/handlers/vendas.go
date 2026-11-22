package handlers

// vendas.go — Assistente FbTax (bot de vendas da landing apuracaoassistida.ia.br), issue #1.
//
// POST /api/vendas/chat   (público, chamado pela landing): um turno da conversa.
// GET  /api/vendas/leads  (admin): lista de leads para a equipe.
// PATCH /api/vendas/leads?id= (admin): situação e observação.

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fb_cloud/services"
)

const (
	vendasMaxCaracteres      = 600
	vendasMaxMensagens       = 30 // mensagens do visitante por conversa
	vendasMaxConversasIPHora = 10
)

// VendasRL: mensagens por IP por hora (rota pública que gasta crédito de IA).
var VendasRL = newRateLimiter(40, time.Hour)

var uuidVendas = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ipVisitante: atrás do Cloudflare o IP real vem em CF-Connecting-IP.
func ipVisitante(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	return GetClientIP(r)
}

func hashIP(ip string) string {
	s := sha256.Sum256([]byte("fbtax-vendas:" + ip))
	return hex.EncodeToString(s[:16])
}

func erroVendas(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// verificarTurnstile confere o token anti-robô do Cloudflare quando TURNSTILE_SECRET está configurado.
func verificarTurnstile(ctx context.Context, token, ip string) bool {
	secret := strings.TrimSpace(os.Getenv("TURNSTILE_SECRET"))
	if secret == "" {
		return true
	}
	if token == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	form := url.Values{"secret": {secret}, "response": {token}, "remoteip": {ip}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[Vendas] turnstile: %v", err)
		return false
	}
	defer resp.Body.Close()
	var out struct {
		Success bool `json:"success"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Success
}

type leadVendas struct {
	ID            int64
	Sessao        string
	Nome, Empresa string
	Cargo, Email  string
	Whatsapp      string
	Faixa, ERP    string
	NumEmpresas   string
	Prazo, Resumo string
	Status        string
	Consentimento bool
	Encaminhado   bool
	Mensagens     int
	Conversa      []services.MensagemVendas
}

func corta(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// aplicarAtualizacoes junta o que o modelo extraiu. Contato só entra com consentimento (LGPD).
func aplicarAtualizacoes(l *leadVendas, ups []services.AtualizacaoLead) {
	for _, u := range ups {
		if u.ConsentimentoContato {
			l.Consentimento = true
		}
	}
	set := func(dst *string, v string, n int) {
		if v = corta(v, n); v != "" {
			*dst = v
		}
	}
	for _, u := range ups {
		set(&l.Nome, u.Nome, 120)
		set(&l.Empresa, u.Empresa, 160)
		set(&l.Cargo, u.Cargo, 120)
		set(&l.Faixa, u.FaixaFaturamento, 80)
		set(&l.ERP, u.ERP, 80)
		set(&l.NumEmpresas, u.NumEmpresas, 40)
		set(&l.Prazo, u.Prazo, 80)
		set(&l.Resumo, u.Resumo, 300)
		if l.Consentimento {
			set(&l.Email, strings.ToLower(u.Email), 160)
			set(&l.Whatsapp, u.Whatsapp, 40)
		}
	}
}

// statusLead: quente quando encaminhado; qualificado quando já se sabe quem é e o porte/ERP.
func statusLead(l *leadVendas, encaminhar bool) string {
	if l.Status == "contatado" || l.Status == "descartado" {
		return l.Status
	}
	if encaminhar || l.Encaminhado {
		return "quente"
	}
	if l.Empresa != "" && (l.Faixa != "" || l.ERP != "" || l.Email != "" || l.Whatsapp != "") {
		return "qualificado"
	}
	return "conversa"
}

func camposLead(l *leadVendas) [][2]string {
	return [][2]string{
		{"Nome", l.Nome}, {"Cargo", l.Cargo}, {"Empresa", l.Empresa}, {"E-mail", l.Email}, {"WhatsApp", l.Whatsapp},
		{"Faturamento", l.Faixa}, {"ERP", l.ERP}, {"Empresas no grupo", l.NumEmpresas}, {"Prazo", l.Prazo}, {"Interesse", l.Resumo},
	}
}

func textoTelegramLead(l *leadVendas, motivo string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔥 Lead #%d do Assistente FbTax", l.ID))
	if motivo != "" {
		b.WriteString(" — " + motivo)
	}
	b.WriteString("\n")
	for _, c := range camposLead(l) {
		if c[1] != "" {
			b.WriteString(c[0] + ": " + c[1] + "\n")
		}
	}
	b.WriteString("\nResponder pelo WhatsApp/e-mail informado. Lista: " + strings.TrimRight(appURLVendas(), "/") + "/admin/financeiro/leads")
	return b.String()
}

func appURLVendas() string {
	if u := os.Getenv("APP_URL"); u != "" {
		return u
	}
	return "https://fbtax.cloud"
}

func destinatariosVendas() []string {
	v := strings.TrimSpace(os.Getenv("VENDAS_EMAIL_EQUIPE"))
	if v == "" {
		v = "contato@fbtechia.com"
	}
	var out []string
	for _, e := range strings.Split(v, ",") {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}

// VendasChatHandler: POST {sessao?, mensagem, origem?, turnstile?} -> {sessao, resposta, whatsapp_url?}
func VendasChatHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			erroVendas(w, http.StatusMethodNotAllowed, "método não permitido")
			return
		}
		ip := ipVisitante(r)
		if !VendasRL.Allow(ip) {
			erroVendas(w, http.StatusTooManyRequests, "Muitas mensagens em pouco tempo. Tente de novo mais tarde ou fale com a equipe pelo WhatsApp.")
			return
		}
		var req struct {
			Sessao    string `json:"sessao"`
			Mensagem  string `json:"mensagem"`
			Origem    string `json:"origem"`
			Turnstile string `json:"turnstile"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			erroVendas(w, http.StatusBadRequest, "requisição inválida")
			return
		}
		msg := strings.TrimSpace(req.Mensagem)
		if msg == "" || len([]rune(msg)) > vendasMaxCaracteres {
			erroVendas(w, http.StatusBadRequest, fmt.Sprintf("escreva uma mensagem de até %d caracteres", vendasMaxCaracteres))
			return
		}

		l := leadVendas{}
		var conversaJSON []byte
		existe := false
		if uuidVendas.MatchString(req.Sessao) {
			err := db.QueryRow(`SELECT id, sessao::text, nome, empresa, cargo, email, whatsapp, faixa_faturamento, erp, num_empresas, prazo, resumo,
					status, consentimento_em IS NOT NULL, encaminhado_em IS NOT NULL, mensagens, conversa
				FROM vendas_leads WHERE sessao = $1::uuid`, req.Sessao).Scan(&l.ID, &l.Sessao, &l.Nome, &l.Empresa, &l.Cargo, &l.Email, &l.Whatsapp,
				&l.Faixa, &l.ERP, &l.NumEmpresas, &l.Prazo, &l.Resumo, &l.Status, &l.Consentimento, &l.Encaminhado, &l.Mensagens, &conversaJSON)
			if err == nil {
				existe = true
				json.Unmarshal(conversaJSON, &l.Conversa)
			} else if err != sql.ErrNoRows {
				log.Printf("[Vendas] ler conversa: %v", err)
				erroVendas(w, http.StatusInternalServerError, "erro ao carregar a conversa")
				return
			}
		}
		if !existe {
			var n int
			db.QueryRow(`SELECT COUNT(*) FROM vendas_leads WHERE ip_hash = $1 AND criado_em > NOW() - INTERVAL '1 hour'`, hashIP(ip)).Scan(&n)
			if n >= vendasMaxConversasIPHora {
				erroVendas(w, http.StatusTooManyRequests, "Muitas conversas novas deste endereço. Fale com a equipe pelo WhatsApp.")
				return
			}
			if !verificarTurnstile(r.Context(), req.Turnstile, ip) {
				erroVendas(w, http.StatusForbidden, "Não foi possível confirmar que você não é um robô. Recarregue a página.")
				return
			}
			if err := db.QueryRow(`INSERT INTO vendas_leads (sessao, origem, ip_hash) VALUES (gen_random_uuid(), $1, $2) RETURNING id, sessao::text`,
				corta(req.Origem, 200), hashIP(ip)).Scan(&l.ID, &l.Sessao); err != nil {
				log.Printf("[Vendas] criar conversa: %v", err)
				erroVendas(w, http.StatusInternalServerError, "erro ao iniciar a conversa")
				return
			}
			l.Status = "conversa"
		}
		whatsapp := services.LinkWhatsappLead(l.ID, l.Nome, l.Empresa)
		if l.Mensagens >= vendasMaxMensagens {
			json.NewEncoder(w).Encode(map[string]any{"sessao": l.Sessao, "whatsapp_url": whatsapp,
				"resposta": "Esta conversa ficou longa. Para continuar, fale direto com um especialista da FBTechIA pelo WhatsApp."})
			return
		}

		res, err := services.ConversarVendas(r.Context(), l.Conversa, msg)
		if err != nil {
			log.Printf("[Vendas] lead #%d: %v", l.ID, err)
			erroVendas(w, http.StatusBadGateway, "O assistente não respondeu agora. Tente de novo ou fale com a equipe pelo WhatsApp.")
			return
		}
		aplicarAtualizacoes(&l, res.Atualizacoes)
		encaminharAgora := res.Encaminhar && !l.Encaminhado
		l.Status = statusLead(&l, res.Encaminhar)
		l.Conversa = append(l.Conversa, services.MensagemVendas{Papel: "visitante", Texto: msg}, services.MensagemVendas{Papel: "assistente", Texto: res.Resposta})
		conv, _ := json.Marshal(l.Conversa)
		if _, err := db.Exec(`UPDATE vendas_leads SET nome=$2, empresa=$3, cargo=$4, email=$5, whatsapp=$6, faixa_faturamento=$7, erp=$8,
				num_empresas=$9, prazo=$10, resumo=$11, status=$12, conversa=$13::jsonb, mensagens = mensagens + 1, atualizado_em = NOW(),
				consentimento_em = CASE WHEN $14 THEN COALESCE(consentimento_em, NOW()) ELSE consentimento_em END,
				encaminhado_em = CASE WHEN $15 THEN COALESCE(encaminhado_em, NOW()) ELSE encaminhado_em END
			WHERE id = $1`, l.ID, l.Nome, l.Empresa, l.Cargo, l.Email, l.Whatsapp, l.Faixa, l.ERP, l.NumEmpresas, l.Prazo, l.Resumo,
			l.Status, string(conv), l.Consentimento, encaminharAgora); err != nil {
			log.Printf("[Vendas] gravar lead #%d: %v", l.ID, err)
		}

		if encaminharAgora {
			lead := l
			motivo := res.Motivo
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := services.EnviarTelegramVendas(ctx, textoTelegramLead(&lead, motivo)); err != nil {
					log.Printf("[Vendas] telegram do lead #%d: %v", lead.ID, err)
				}
				if err := services.SendLeadVendasEmail(destinatariosVendas(), lead.ID, camposLead(&lead), lead.Conversa); err != nil {
					log.Printf("[Vendas] e-mail do lead #%d: %v", lead.ID, err)
				}
			}()
			log.Printf("[Vendas] lead #%d encaminhado (%s)", l.ID, res.Motivo)
		}

		out := map[string]any{"sessao": l.Sessao, "resposta": res.Resposta}
		if l.Encaminhado || encaminharAgora {
			out["whatsapp_url"] = services.LinkWhatsappLead(l.ID, l.Nome, l.Empresa)
		}
		json.NewEncoder(w).Encode(out)
	}
}

type LeadVendasLista struct {
	ID           int64                     `json:"id"`
	CriadoEm     time.Time                 `json:"criado_em"`
	AtualizadoEm time.Time                 `json:"atualizado_em"`
	Nome         string                    `json:"nome"`
	Empresa      string                    `json:"empresa"`
	Cargo        string                    `json:"cargo"`
	Email        string                    `json:"email"`
	Whatsapp     string                    `json:"whatsapp"`
	Faixa        string                    `json:"faixa_faturamento"`
	ERP          string                    `json:"erp"`
	NumEmpresas  string                    `json:"num_empresas"`
	Prazo        string                    `json:"prazo"`
	Resumo       string                    `json:"resumo"`
	Origem       string                    `json:"origem"`
	Status       string                    `json:"status"`
	Observacao   string                    `json:"observacao"`
	Mensagens    int                       `json:"mensagens"`
	Conversa     []services.MensagemVendas `json:"conversa"`
}

var statusLeadValidos = map[string]bool{"conversa": true, "qualificado": true, "quente": true, "contatado": true, "descartado": true}

// VendasLeadsHandler: GET ?status= lista; PATCH ?id= {status, observacao}. Só admin (rota com withAuth "admin").
func VendasLeadsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			st := r.URL.Query().Get("status")
			if !statusLeadValidos[st] {
				st = ""
			}
			rows, err := db.Query(`SELECT id, criado_em, atualizado_em, nome, empresa, cargo, email, whatsapp, faixa_faturamento, erp, num_empresas,
					prazo, resumo, origem, status, observacao, mensagens, conversa
				FROM vendas_leads WHERE ($1 = '' OR status = $1) AND mensagens > 0
				ORDER BY CASE status WHEN 'quente' THEN 0 WHEN 'qualificado' THEN 1 WHEN 'conversa' THEN 2 ELSE 3 END, atualizado_em DESC LIMIT 300`, st)
			if err != nil {
				log.Printf("[Vendas] listar: %v", err)
				erroVendas(w, http.StatusInternalServerError, "erro ao listar leads")
				return
			}
			defer rows.Close()
			lista := []LeadVendasLista{}
			for rows.Next() {
				var l LeadVendasLista
				var conv []byte
				if err := rows.Scan(&l.ID, &l.CriadoEm, &l.AtualizadoEm, &l.Nome, &l.Empresa, &l.Cargo, &l.Email, &l.Whatsapp, &l.Faixa, &l.ERP,
					&l.NumEmpresas, &l.Prazo, &l.Resumo, &l.Origem, &l.Status, &l.Observacao, &l.Mensagens, &conv); err != nil {
					log.Printf("[Vendas] listar scan: %v", err)
					erroVendas(w, http.StatusInternalServerError, "erro ao listar leads")
					return
				}
				json.Unmarshal(conv, &l.Conversa)
				if l.Conversa == nil {
					l.Conversa = []services.MensagemVendas{}
				}
				lista = append(lista, l)
			}
			json.NewEncoder(w).Encode(lista)
		case http.MethodPatch:
			id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
			if err != nil || id <= 0 {
				erroVendas(w, http.StatusBadRequest, "lead inválido")
				return
			}
			var req struct {
				Status     string  `json:"status"`
				Observacao *string `json:"observacao"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !statusLeadValidos[req.Status] {
				erroVendas(w, http.StatusBadRequest, "situação inválida")
				return
			}
			obs := ""
			if req.Observacao != nil {
				obs = corta(*req.Observacao, 2000)
			}
			res, err := db.Exec(`UPDATE vendas_leads SET status = $2::text, atualizado_em = NOW(),
					observacao = CASE WHEN $3::boolean THEN $4::text ELSE observacao END WHERE id = $1`,
				id, req.Status, req.Observacao != nil, obs)
			if err != nil {
				log.Printf("[Vendas] atualizar #%d: %v", id, err)
				erroVendas(w, http.StatusInternalServerError, "erro ao atualizar o lead")
				return
			}
			if n, _ := res.RowsAffected(); n == 0 {
				erroVendas(w, http.StatusNotFound, "lead não encontrado")
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"id": id, "status": req.Status})
		default:
			erroVendas(w, http.StatusMethodNotAllowed, "método não permitido")
		}
	}
}
