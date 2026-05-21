package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// SyncResult é o resultado de uma sincronização de conta.
type SyncResult struct {
	Importadas int
	Duplicadas int
	SaldoFinal float64
}

// SyncAccount sincroniza uma conta bancária usando o provider adequado.
// configJSON é o JSON descriptografado com as credenciais (ProviderConfig).
func SyncAccount(db *sql.DB, contaID, provedor, configJSON string) (*SyncResult, error) {
	provider, ok := GetProvider(provedor)
	if !ok {
		return nil, fmt.Errorf("provider '%s' não suportado", provedor)
	}

	var cfg ProviderConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("config inválido: %w", err)
	}

	// Período: da última sync até hoje (mínimo 30 dias)
	dataFim := time.Now().Format("2006-01-02")
	dataInicio := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	var ultimaSync sql.NullTime
	db.QueryRow(`SELECT ultima_sync FROM financeiro.contas_financeiras WHERE id = $1`, contaID).Scan(&ultimaSync)
	if ultimaSync.Valid && ultimaSync.Time.After(time.Now().AddDate(0, 0, -60)) {
		dataInicio = ultimaSync.Time.AddDate(0, 0, -1).Format("2006-01-02")
	}

	txs, err := provider.GetExtrato(cfg, dataInicio, dataFim)
	if err != nil {
		return nil, fmt.Errorf("extrato: %w", err)
	}

	res := &SyncResult{}
	for _, tx := range txs {
		if tx.IDExterno == "" {
			continue
		}
		var exists bool
		db.QueryRow(`SELECT EXISTS(SELECT 1 FROM financeiro.transacoes WHERE referencia_ext = $1)`, tx.IDExterno).Scan(&exists)
		if exists {
			res.Duplicadas++
			continue
		}
		_, err := db.Exec(`
			INSERT INTO financeiro.transacoes
			    (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext, conciliado)
			VALUES ($1, $2, $3, $4, $5, $6, $7, true)`,
			contaID, tx.DataTransacao, tx.Descricao, tx.Valor, tx.Tipo, tx.Categoria, tx.IDExterno,
		)
		if err != nil {
			log.Printf("[bank-sync] erro inserindo tx %s conta %s: %v", tx.IDExterno, contaID, err)
			continue
		}
		res.Importadas++
	}

	// Atualiza saldo e ultima_sync
	saldo, err := provider.GetSaldo(cfg)
	if err == nil {
		res.SaldoFinal = saldo.Disponivel
		db.Exec(`UPDATE financeiro.contas_financeiras SET saldo = $1, ultima_sync = NOW() WHERE id = $2`,
			saldo.Disponivel, contaID)
	} else {
		log.Printf("[bank-sync] saldo falhou conta %s: %v", contaID, err)
		db.Exec(`UPDATE financeiro.contas_financeiras SET ultima_sync = NOW() WHERE id = $1`, contaID)
	}

	return res, nil
}

// SyncAccountFromDB carrega as credenciais do banco, descriptografa e sincroniza.
func SyncAccountFromDB(db *sql.DB, contaID string) (*SyncResult, error) {
	var provedor string
	var configEnc sql.NullString
	err := db.QueryRow(`
		SELECT provedor, provedor_config
		FROM financeiro.contas_financeiras
		WHERE id = $1 AND ativa = true`, contaID).Scan(&provedor, &configEnc)
	if err != nil {
		return nil, fmt.Errorf("conta não encontrada: %w", err)
	}
	if !configEnc.Valid || configEnc.String == "" {
		return nil, fmt.Errorf("conta %s não tem provedor_config configurado", contaID)
	}
	configJSON, err := DecryptConfig(configEnc.String)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}
	return SyncAccount(db, contaID, provedor, configJSON)
}

// StartBankDailySync inicia o cron diário que sincroniza todas as contas
// com provedor e provedor_config configurados. Executa às 06:00.
func StartBankDailySync(db *sql.DB) {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))

			rows, err := db.Query(`
				SELECT id, provedor FROM financeiro.contas_financeiras
				WHERE ativa = true
				  AND provedor IS NOT NULL
				  AND provedor_config IS NOT NULL`)
			if err != nil {
				log.Printf("[bank-cron] erro listar contas: %v", err)
				continue
			}
			type contaJob struct{ id, provedor string }
			var jobs []contaJob
			for rows.Next() {
				var j contaJob
				rows.Scan(&j.id, &j.provedor)
				jobs = append(jobs, j)
			}
			rows.Close()

			for _, j := range jobs {
				log.Printf("[bank-cron] sincronizando %s (%s)", j.id, j.provedor)

				var logID string
				db.QueryRow(`
					INSERT INTO financeiro.inter_sync_log (conta_id, status)
					VALUES ($1, 'running') RETURNING id`, j.id).Scan(&logID)

				result, err := SyncAccountFromDB(db, j.id)
				if err != nil {
					log.Printf("[bank-cron] erro conta %s: %v", j.id, err)
					db.Exec(`UPDATE financeiro.inter_sync_log
						SET encerrado_em = NOW(), status = 'erro', erro_detalhe = $1
						WHERE id = $2`, err.Error(), logID)
					continue
				}
				db.Exec(`UPDATE financeiro.inter_sync_log
					SET encerrado_em = NOW(), status = 'ok',
					    tx_importadas = $1, tx_duplicadas = $2, saldo_final = $3,
					    data_inicio = NOW()-INTERVAL '30 days', data_fim = NOW()
					WHERE id = $4`,
					result.Importadas, result.Duplicadas, result.SaldoFinal, logID)
				log.Printf("[bank-cron] %s: %d importadas, %d duplicadas", j.id, result.Importadas, result.Duplicadas)
			}
		}
	}()
}
