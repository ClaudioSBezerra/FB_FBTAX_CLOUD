package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
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
			Status       string     `json:"status"`
			IniciadoEm   time.Time  `json:"iniciado_em"`
			TxImportadas int        `json:"tx_importadas"`
			TxDuplicadas int        `json:"tx_duplicadas"`
			SaldoFinal   *float64   `json:"saldo_final,omitempty"`
			ErrDetalhe   *string    `json:"erro_detalhe,omitempty"`
		}
		type ContaStatus struct {
			ID             string    `json:"id"`
			Apelido        string    `json:"apelido"`
			Banco          string    `json:"banco"`
			Provedor       string    `json:"provedor"`
			Saldo          float64   `json:"saldo"`
			UltimaSync     *string   `json:"ultima_sync,omitempty"`
			Configurado    bool      `json:"configurado"`
			UltimoSync     *SyncLog  `json:"ultimo_sync,omitempty"`
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
				if sf.Valid { sl.SaldoFinal = &sf.Float64 }
				if ed.Valid { sl.ErrDetalhe = &ed.String }
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

// PUT /api/financeiro/bancos/config
// Salva credenciais do provedor para uma conta (criptografadas).
// Body: { conta_id, provedor, config: { client_id, client_secret, cert_pem, key_pem, ... } }
func BancosConfigHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			ContaID  string                  `json:"conta_id"`
			Provedor string                  `json:"provedor"`
			Config   services.ProviderConfig `json:"config"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			http.Error(w, "erro serializando config", http.StatusInternalServerError)
			return
		}
		encrypted, err := services.EncryptConfig(string(configJSON))
		if err != nil {
			http.Error(w, "erro criptografando config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = db.Exec(`
			UPDATE financeiro.contas_financeiras
			SET provedor = $1, provedor_config = $2
			WHERE id = $3`, req.Provedor, encrypted, req.ContaID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"mensagem": "Configuração salva com sucesso"})
	}
}

// POST /api/financeiro/bancos/sync
// Dispara sync manual para uma conta específica em background.
// Body: { conta_id }
func BancosSyncHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		var req struct {
			ContaID string `json:"conta_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ContaID == "" {
			http.Error(w, "conta_id obrigatório", http.StatusBadRequest)
			return
		}

		// Verifica se a conta existe e tem config
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
			json.NewEncoder(w).Encode(map[string]string{"erro": "Conta sem credenciais configuradas. Configure primeiro em Bancos → Configurar API."})
			return
		}

		var logID string
		db.QueryRow(`
			INSERT INTO financeiro.inter_sync_log (conta_id, status)
			VALUES ($1, 'running') RETURNING id`, req.ContaID).Scan(&logID)

		go func() {
			result, err := services.SyncAccountFromDB(db, req.ContaID)
			if err != nil {
				db.Exec(`UPDATE financeiro.inter_sync_log
					SET encerrado_em = NOW(), status = 'erro', erro_detalhe = $1
					WHERE id = $2`, err.Error(), logID)
				return
			}
			db.Exec(`UPDATE financeiro.inter_sync_log
				SET encerrado_em = NOW(), status = 'ok',
				    tx_importadas = $1, tx_duplicadas = $2, saldo_final = $3,
				    data_inicio = NOW()-INTERVAL '30 days', data_fim = NOW()
				WHERE id = $4`,
				result.Importadas, result.Duplicadas, result.SaldoFinal, logID)
		}()

		json.NewEncoder(w).Encode(map[string]string{
			"mensagem": "Sincronização iniciada",
			"log_id":   logID,
		})
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
