package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"fb_cloud/services"
)

// GET /api/financeiro/inter/status
// Retorna a conta Inter cadastrada e o último log de sync
func InterStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		type ContaInfo struct {
			ID        string     `json:"id"`
			Apelido   string     `json:"apelido"`
			Banco     string     `json:"banco"`
			Saldo     float64    `json:"saldo"`
			UltimaSync *time.Time `json:"ultima_sync"`
		}
		type LogInfo struct {
			Status      string     `json:"status"`
			IniciadoEm  time.Time  `json:"iniciado_em"`
			TxImportadas int       `json:"tx_importadas"`
			TxDuplicadas int       `json:"tx_duplicadas"`
			SaldoFinal  *float64   `json:"saldo_final"`
			ErrDetalhe  *string    `json:"erro_detalhe"`
		}
		type StatusResp struct {
			Conta      *ContaInfo `json:"conta"`
			UltimoSync *LogInfo   `json:"ultimo_sync"`
			Configurado bool      `json:"configurado"`
		}

		resp := StatusResp{Configurado: isInterConfigured()}

		row := db.QueryRow(`
			SELECT id, apelido, banco, saldo, ultima_sync
			FROM financeiro.contas_financeiras
			WHERE provedor = 'inter' AND ativa = true
			LIMIT 1`)

		var conta ContaInfo
		var ultimaSync sql.NullTime
		if err := row.Scan(&conta.ID, &conta.Apelido, &conta.Banco, &conta.Saldo, &ultimaSync); err == nil {
			if ultimaSync.Valid {
				t := ultimaSync.Time
				conta.UltimaSync = &t
			}
			resp.Conta = &conta

			// Último log de sync
			logRow := db.QueryRow(`
				SELECT status, iniciado_em, tx_importadas, tx_duplicadas, saldo_final, erro_detalhe
				FROM financeiro.inter_sync_log
				WHERE conta_id = $1
				ORDER BY iniciado_em DESC
				LIMIT 1`, conta.ID)

			var logInfo LogInfo
			var sf sql.NullFloat64
			var ed sql.NullString
			if err := logRow.Scan(&logInfo.Status, &logInfo.IniciadoEm, &logInfo.TxImportadas, &logInfo.TxDuplicadas, &sf, &ed); err == nil {
				if sf.Valid {
					logInfo.SaldoFinal = &sf.Float64
				}
				if ed.Valid {
					logInfo.ErrDetalhe = &ed.String
				}
				resp.UltimoSync = &logInfo
			}
		}

		json.NewEncoder(w).Encode(resp)
	}
}

// POST /api/financeiro/inter/sync
// Dispara sync manual em background
func InterSyncHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		if !isInterConfigured() {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"erro": "Inter API não configurada. Defina INTER_CLIENT_ID, INTER_CLIENT_SECRET, INTER_CERT_PEM, INTER_KEY_PEM."})
			return
		}

		row := db.QueryRow(`
			SELECT id FROM financeiro.contas_financeiras
			WHERE provedor = 'inter' AND ativa = true LIMIT 1`)
		var contaID string
		if err := row.Scan(&contaID); err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"erro": "Nenhuma conta Inter ativa encontrada"})
			return
		}

		// Registra log de inicio
		var logID string
		db.QueryRow(`
			INSERT INTO financeiro.inter_sync_log (conta_id, status)
			VALUES ($1, 'running') RETURNING id`, contaID).Scan(&logID)

		// Executa em background
		go func() {
			result, err := services.SyncInterAccount(db, contaID)
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

		json.NewEncoder(w).Encode(map[string]string{"mensagem": "Sincronização iniciada", "log_id": logID})
	}
}

func isInterConfigured() bool {
	return os.Getenv("INTER_CLIENT_ID") != "" &&
		os.Getenv("INTER_CLIENT_SECRET") != "" &&
		os.Getenv("INTER_CERT_PEM") != "" &&
		os.Getenv("INTER_KEY_PEM") != ""
}
