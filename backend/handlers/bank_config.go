package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fb_cloud/services"
)

// GET /api/financeiro/bancos/status
// Retorna todas as contas com provedor configurado e o último log de sync.
func BancosStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		type SyncLog struct {
			Status       string    `json:"status"`
			IniciadoEm   time.Time `json:"iniciado_em"`
			TxImportadas int       `json:"tx_importadas"`
			TxDuplicadas int       `json:"tx_duplicadas"`
			SaldoFinal   *float64  `json:"saldo_final,omitempty"`
			ErrDetalhe   *string   `json:"erro_detalhe,omitempty"`
		}
		type ContaStatus struct {
			ID          string   `json:"id"`
			Apelido     string   `json:"apelido"`
			Banco       string   `json:"banco"`
			Provedor    string   `json:"provedor"`
			Saldo       float64  `json:"saldo"`
			UltimaSync  *string  `json:"ultima_sync,omitempty"`
			Configurado bool     `json:"configurado"`
			UltimoSync  *SyncLog `json:"ultimo_sync,omitempty"`
		}

		rows, err := db.Query(`
			SELECT id, apelido, banco, COALESCE(provedor,''), saldo,
			       ultima_sync, provedor_config IS NOT NULL AND provedor_config <> ''
			FROM financeiro.contas_financeiras
			WHERE ativa = true
			ORDER BY apelido`)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var contas []ContaStatus
		for rows.Next() {
			var c ContaStatus
			var ultimaSync sql.NullTime
			rows.Scan(&c.ID, &c.Apelido, &c.Banco, &c.Provedor, &c.Saldo, &ultimaSync, &c.Configurado)
			if ultimaSync.Valid {
				s := ultimaSync.Time.Format(time.RFC3339)
				c.UltimaSync = &s
			}

			// Último log de sync
			logRow := db.QueryRow(`
				SELECT status, iniciado_em, tx_importadas, tx_duplicadas, saldo_final, erro_detalhe
				FROM financeiro.inter_sync_log
				WHERE conta_id = $1
				ORDER BY iniciado_em DESC LIMIT 1`, c.ID)
			var sl SyncLog
			var sf sql.NullFloat64
			var ed sql.NullString
			if err := logRow.Scan(&sl.Status, &sl.IniciadoEm, &sl.TxImportadas, &sl.TxDuplicadas, &sf, &ed); err == nil {
				if sf.Valid {
					sl.SaldoFinal = &sf.Float64
				}
				if ed.Valid {
					sl.ErrDetalhe = &ed.String
				}
				c.UltimoSync = &sl
			}
			contas = append(contas, c)
		}
		if contas == nil {
			contas = []ContaStatus{}
		}
		json.NewEncoder(w).Encode(contas)
	}
}

// configBancoMeta é o que se mostra da configuração salva: nunca o segredo nem a chave privada.
type configBancoMeta struct {
	Configurado    bool   `json:"configurado"`
	Provedor       string `json:"provedor,omitempty"`
	ClientIDFinal  string `json:"client_id_final,omitempty"` // só os 4 últimos caracteres
	TemSecret      bool   `json:"tem_secret"`
	TemCertificado bool   `json:"tem_certificado"`
	TemChave       bool   `json:"tem_chave"`
	ContaCorrente  string `json:"conta_corrente,omitempty"`
	Scopes         string `json:"scopes,omitempty"`
	CertValidade   string `json:"cert_validade,omitempty"`
	CertDiasRestam int    `json:"cert_dias_restantes,omitempty"`
}

func carregarConfigBanco(db *sql.DB, contaID string) (provedor string, cfg services.ProviderConfig, existe bool, err error) {
	var enc sql.NullString
	var prov sql.NullString
	err = db.QueryRow(`SELECT provedor, provedor_config FROM financeiro.contas_financeiras WHERE id = $1 AND ativa = true`, contaID).Scan(&prov, &enc)
	if err != nil {
		return "", cfg, false, err
	}
	provedor = prov.String
	if !enc.Valid || enc.String == "" {
		return provedor, cfg, false, nil
	}
	js, derr := services.DecryptConfig(enc.String)
	if derr != nil {
		return provedor, cfg, false, derr
	}
	if jerr := json.Unmarshal([]byte(js), &cfg); jerr != nil {
		return provedor, cfg, false, jerr
	}
	return provedor, cfg, true, nil
}

// BancosConfigHandler:
//
//	GET  ?conta_id=  → metadados da configuração (sem segredos)
//	PUT/POST         → salva; campos em branco PRESERVAM o valor já guardado
//	                   Body: { conta_id, provedor, config: { client_id, client_secret, cert_pem, key_pem, conta_corrente, scopes } }
func BancosConfigHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			contaID := r.URL.Query().Get("conta_id")
			prov, cfg, existe, err := carregarConfigBanco(db, contaID)
			if err != nil {
				http.Error(w, "conta não encontrada ou configuração ilegível", http.StatusNotFound)
				return
			}
			m := configBancoMeta{Configurado: existe, Provedor: prov}
			if existe {
				m.ClientIDFinal = ultimos(cfg.ClientID, 4)
				m.TemSecret = cfg.ClientSecret != ""
				m.TemCertificado, m.TemChave = cfg.CertPEM != "", cfg.KeyPEM != ""
				m.ContaCorrente, m.Scopes = cfg.ContaCorrente, cfg.Scopes
				if cfg.CertPEM != "" && cfg.KeyPEM != "" {
					if c, err := services.ValidarParCertificadoInter(cfg.CertPEM, cfg.KeyPEM); err == nil {
						m.CertValidade = c.NotAfter.Format("2006-01-02")
						m.CertDiasRestam = int(time.Until(c.NotAfter).Hours() / 24)
					}
				}
			}
			json.NewEncoder(w).Encode(m)
			return
		}
		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ContaID  string                  `json:"conta_id"`
			Provedor string                  `json:"provedor"`
			Config   services.ProviderConfig `json:"config"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10)).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if req.ContaID == "" || req.Provedor == "" {
			http.Error(w, "conta_id e provedor obrigatórios", http.StatusBadRequest)
			return
		}
		if _, ok := services.GetProvider(req.Provedor); !ok {
			http.Error(w, "provedor não suportado: "+req.Provedor, http.StatusBadRequest)
			return
		}
		if !services.ChaveCriptografiaConfigurada() {
			http.Error(w, "FINANCEIRO_SECRET não configurado: credenciais bancárias não podem ser guardadas sem criptografia", http.StatusUnprocessableEntity)
			return
		}

		// Preserva o que já existe quando o campo vem em branco (o front nunca recebe os segredos de volta).
		_, atual, _, _ := carregarConfigBanco(db, req.ContaID)
		novo := req.Config
		manter := func(n *string, antigo string) {
			if strings.TrimSpace(*n) == "" {
				*n = antigo
			}
		}
		manter(&novo.ClientID, atual.ClientID)
		manter(&novo.ClientSecret, atual.ClientSecret)
		manter(&novo.CertPEM, atual.CertPEM)
		manter(&novo.KeyPEM, atual.KeyPEM)
		manter(&novo.ContaCorrente, atual.ContaCorrente)
		manter(&novo.Scopes, atual.Scopes)
		manter(&novo.Agencia, atual.Agencia)
		manter(&novo.Conta, atual.Conta)
		manter(&novo.BaseURL, atual.BaseURL)

		if req.Provedor == "inter" && novo.CertPEM != "" && novo.KeyPEM != "" {
			cert, err := services.ValidarParCertificadoInter(novo.CertPEM, novo.KeyPEM)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			if time.Now().After(cert.NotAfter) {
				http.Error(w, "o certificado do Inter está vencido ("+cert.NotAfter.Format("02/01/2006")+")", http.StatusUnprocessableEntity)
				return
			}
		}

		configJSON, err := json.Marshal(novo)
		if err != nil {
			http.Error(w, "erro serializando config", http.StatusInternalServerError)
			return
		}
		encrypted, err := services.EncryptConfig(string(configJSON))
		if err != nil {
			http.Error(w, "erro criptografando config", http.StatusInternalServerError)
			return
		}
		if _, err = db.Exec(`UPDATE financeiro.contas_financeiras SET provedor = $1, provedor_config = $2 WHERE id = $3`, req.Provedor, encrypted, req.ContaID); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		log.Printf("[Bancos] credenciais do provedor %s atualizadas (conta %s)", req.Provedor, req.ContaID)
		json.NewEncoder(w).Encode(map[string]string{"mensagem": "Configuração salva com sucesso"})
	}
}

func ultimos(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// BancosTestarHandler roda o diagnóstico da integração — POST /api/financeiro/bancos/testar {conta_id}.
// Verifica certificado, descobre os escopos reais da integração e testa o saldo. Somente leitura.
func BancosTestarHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			ContaID string `json:"conta_id"`
		}
		if json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&req) != nil || req.ContaID == "" {
			http.Error(w, "conta_id obrigatório", http.StatusBadRequest)
			return
		}
		prov, cfg, existe, err := carregarConfigBanco(db, req.ContaID)
		if err != nil || !existe {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"erro": "Conta sem credenciais configuradas."})
			return
		}
		if prov != "inter" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"erro": "o teste detalhado só existe para o Inter"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Second)
		defer cancel()
		done := make(chan *services.InterDiagnostico, 1)
		go func() { done <- services.DiagnosticarInter(cfg) }()
		select {
		case d := <-done:
			json.NewEncoder(w).Encode(d)
		case <-ctx.Done():
			w.WriteHeader(http.StatusFailedDependency) // 424: o Cloudflare mascara 502/504
			json.NewEncoder(w).Encode(map[string]string{"erro": "tempo esgotado ao falar com o Inter"})
		}
	}
}

// POST /api/financeiro/bancos/sync
// Dispara sync manual para uma conta em background.
// Body: { conta_id, data_inicio?, data_fim? } — sem datas usa o período padrão (desde a última sync,
// mínimo 30 dias); com datas, valida o máximo de 90 dias do Inter.
func BancosSyncHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			ContaID    string `json:"conta_id"`
			DataInicio string `json:"data_inicio"`
			DataFim    string `json:"data_fim"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 4<<10)).Decode(&req); err != nil || req.ContaID == "" {
			http.Error(w, "conta_id obrigatório", http.StatusBadRequest)
			return
		}
		periodo := req.DataInicio != "" || req.DataFim != ""
		if periodo {
			if err := services.ValidarPeriodoExtrato(req.DataInicio, req.DataFim); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
				return
			}
		}

		var provedor string
		var configEnc sql.NullString
		err := db.QueryRow(`
			SELECT provedor, provedor_config FROM financeiro.contas_financeiras
			WHERE id = $1 AND ativa = true`, req.ContaID).Scan(&provedor, &configEnc)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"erro": "Conta não encontrada"})
			return
		}
		if !configEnc.Valid || configEnc.String == "" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"erro": "Conta sem credenciais configuradas. Configure primeiro em Bancos → Manutenção do banco."})
			return
		}
		// Uma sincronização por vez por conta.
		var rodando bool
		db.QueryRow(`SELECT EXISTS(SELECT 1 FROM financeiro.inter_sync_log WHERE conta_id = $1 AND status = 'running' AND iniciado_em > NOW() - INTERVAL '10 minutes')`, req.ContaID).Scan(&rodando)
		if rodando {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"erro": "Já existe uma sincronização em andamento para esta conta."})
			return
		}

		var logID string
		var ini, fim any
		if periodo {
			ini, fim = req.DataInicio, req.DataFim
		}
		db.QueryRow(`
			INSERT INTO financeiro.inter_sync_log (conta_id, status, data_inicio, data_fim)
			VALUES ($1, 'running', $2, $3) RETURNING id`, req.ContaID, ini, fim).Scan(&logID)

		go func() {
			var result *services.SyncResult
			var err error
			if periodo {
				result, err = services.SyncAccountFromDBPeriodo(db, req.ContaID, req.DataInicio, req.DataFim)
			} else {
				result, err = services.SyncAccountFromDB(db, req.ContaID)
			}
			if err != nil {
				db.Exec(`UPDATE financeiro.inter_sync_log
					SET encerrado_em = NOW(), status = 'erro', erro_detalhe = $1
					WHERE id = $2`, err.Error(), logID)
				return
			}
			db.Exec(`UPDATE financeiro.inter_sync_log
				SET encerrado_em = NOW(), status = 'ok',
				    tx_importadas = $1, tx_duplicadas = $2, saldo_final = $3,
				    data_inicio = $4::date, data_fim = $5::date
				WHERE id = $6`,
				result.Importadas, result.Duplicadas, result.SaldoFinal, result.DataInicio, result.DataFim, logID)
			// Novos créditos podem ter pago títulos em aberto.
			if n, err := services.ConciliarTitulos(context.Background(), db); err != nil {
				log.Printf("[Titulos] conciliação após sincronização: %v", err)
			} else if n > 0 {
				log.Printf("[Titulos] %d título(s) baixado(s) automaticamente após a sincronização", n)
			}
		}()

		json.NewEncoder(w).Encode(map[string]string{
			"mensagem": "Sincronização iniciada",
			"log_id":   logID,
		})
	}
}

// BancosSyncLogHandler lista o histórico de sincronizações — GET /api/financeiro/bancos/sync-log?conta_id=&limit=
func BancosSyncLogHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limite := 20
		if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n <= 100 {
			limite = n
		}
		contaID := r.URL.Query().Get("conta_id")
		rows, err := db.Query(`
			SELECT id, status, iniciado_em, encerrado_em, data_inicio::text, data_fim::text,
			       COALESCE(tx_importadas,0), COALESCE(tx_duplicadas,0), saldo_final, COALESCE(erro_detalhe,'')
			FROM financeiro.inter_sync_log
			WHERE ($1 = '' OR conta_id::text = $1)
			ORDER BY iniciado_em DESC LIMIT $2`, contaID, limite)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		out := []map[string]any{}
		for rows.Next() {
			var id, status, erro string
			var inicio time.Time
			var fim sql.NullTime
			var di, df sql.NullString
			var imp, dup int
			var saldo sql.NullFloat64
			if rows.Scan(&id, &status, &inicio, &fim, &di, &df, &imp, &dup, &saldo, &erro) != nil {
				continue
			}
			m := map[string]any{"id": id, "status": status, "iniciado_em": inicio, "tx_importadas": imp, "tx_duplicadas": dup, "erro": erro}
			if fim.Valid {
				m["encerrado_em"] = fim.Time
			}
			if di.Valid {
				m["data_inicio"] = di.String
			}
			if df.Valid {
				m["data_fim"] = df.String
			}
			if saldo.Valid {
				m["saldo_final"] = saldo.Float64
			}
			out = append(out, m)
		}
		json.NewEncoder(w).Encode(out)
	}
}

// BancosSaldoHandler consulta o saldo AO VIVO no banco e atualiza o saldo guardado da conta —
// GET /api/financeiro/bancos/saldo?conta_id=. O Inter limita 10 chamadas por minuto.
func BancosSaldoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		contaID := r.URL.Query().Get("conta_id")
		prov, cfg, existe, err := carregarConfigBanco(db, contaID)
		if err != nil || !existe {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"erro": "Conta sem credenciais configuradas."})
			return
		}
		provider, ok := services.GetProvider(prov)
		if !ok {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"erro": "provedor não suportado"})
			return
		}
		saldo, err := provider.GetSaldo(cfg)
		if err != nil {
			log.Printf("[Bancos] saldo da conta %s: %v", contaID, err)
			w.WriteHeader(http.StatusFailedDependency) // 424: o Cloudflare mascara 502/504
			json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
			return
		}
		db.Exec(`UPDATE financeiro.contas_financeiras SET saldo = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`, saldo.Disponivel, contaID)
		json.NewEncoder(w).Encode(map[string]any{"disponivel": saldo.Disponivel, "bloqueado": saldo.Bloqueado, "consultado_em": time.Now()})
	}
}

// GET /api/financeiro/bancos/providers
// Retorna a lista de provedores suportados.
func BancosProvidersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(services.ProvidersDisponiveis())
	}
}

// BancosExtratoAmostraHandler mostra o que o Inter devolve para um período (diagnóstico, somente
// leitura, não grava) — GET /api/financeiro/bancos/extrato-amostra?conta_id=&data_inicio=&data_fim=.
func BancosExtratoAmostraHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		prov, cfg, existe, err := carregarConfigBanco(db, q.Get("conta_id"))
		if err != nil || !existe || prov != "inter" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"erro": "Conta sem credenciais do Inter configuradas."})
			return
		}
		a, err := services.AmostrarExtratoInter(cfg, q.Get("data_inicio"), q.Get("data_fim"))
		if err != nil {
			log.Printf("[Bancos] amostra do extrato: %v", err)
			w.WriteHeader(http.StatusFailedDependency) // 424: o Cloudflare mascara 502/504
			json.NewEncoder(w).Encode(map[string]string{"erro": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(a)
	}
}

// CredenciaisVencimentosHandler lista o vencimento das credenciais (certificado do Inter e A1 da
// NFS-e) — GET /api/financeiro/credenciais/vencimentos. A tela mostra faixa de alerta a partir de 30 dias.
func CredenciaisVencimentosHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		creds, err := services.ListarVencimentosCredenciais(r.Context(), db, time.Now())
		if err != nil {
			log.Printf("[Credenciais] listar: %v", err)
			http.Error(w, "erro ao consultar credenciais", http.StatusInternalServerError)
			return
		}
		if creds == nil {
			creds = []services.CredencialVencimento{}
		}
		json.NewEncoder(w).Encode(creds)
	}
}
