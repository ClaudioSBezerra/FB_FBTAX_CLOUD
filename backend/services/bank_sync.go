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
	DataInicio string // período efetivamente consultado (AAAA-MM-DD)
	DataFim    string
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

	// Período padrão: da última sync até hoje (mínimo 30 dias)
	dataFim := time.Now().Format("2006-01-02")
	dataInicio := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	var ultimaSync sql.NullTime
	db.QueryRow(`SELECT ultima_sync FROM financeiro.contas_financeiras WHERE id = $1`, contaID).Scan(&ultimaSync)
	if ultimaSync.Valid && ultimaSync.Time.After(time.Now().AddDate(0, 0, -60)) {
		dataInicio = ultimaSync.Time.AddDate(0, 0, -1).Format("2006-01-02")
	}
	return syncAccountPeriodo(db, contaID, provider, cfg, dataInicio, dataFim)
}

// MaxPeriodoExtratoDias é o período máximo aceito pelo Inter por consulta de extrato.
const MaxPeriodoExtratoDias = 90

// ValidarPeriodoExtrato confere formato, ordem, que o fim não seja futuro e o limite de 90 dias.
func ValidarPeriodoExtrato(inicio, fim string) error {
	i, err1 := time.Parse("2006-01-02", inicio)
	f, err2 := time.Parse("2006-01-02", fim)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("datas devem estar no formato AAAA-MM-DD")
	}
	if f.Before(i) {
		return fmt.Errorf("a data final é anterior à inicial")
	}
	if f.After(time.Now().Add(24 * time.Hour)) {
		return fmt.Errorf("a data final não pode estar no futuro")
	}
	if f.Sub(i) > MaxPeriodoExtratoDias*24*time.Hour {
		return fmt.Errorf("o período máximo é de %d dias", MaxPeriodoExtratoDias)
	}
	return nil
}

// SyncAccountFromDBPeriodo sincroniza um período explícito (validado por ValidarPeriodoExtrato).
func SyncAccountFromDBPeriodo(db *sql.DB, contaID, dataInicio, dataFim string) (*SyncResult, error) {
	if err := ValidarPeriodoExtrato(dataInicio, dataFim); err != nil {
		return nil, err
	}
	var provedor string
	var configEnc sql.NullString
	if err := db.QueryRow(`SELECT provedor, provedor_config FROM financeiro.contas_financeiras WHERE id = $1 AND ativa = true`, contaID).Scan(&provedor, &configEnc); err != nil {
		return nil, fmt.Errorf("conta não encontrada: %w", err)
	}
	if !configEnc.Valid || configEnc.String == "" {
		return nil, fmt.Errorf("conta %s não tem provedor_config configurado", contaID)
	}
	configJSON, err := DecryptConfig(configEnc.String)
	if err != nil {
		return nil, fmt.Errorf("decrypt config: %w", err)
	}
	provider, ok := GetProvider(provedor)
	if !ok {
		return nil, fmt.Errorf("provider '%s' não suportado", provedor)
	}
	var cfg ProviderConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return nil, fmt.Errorf("config inválido: %w", err)
	}
	return syncAccountPeriodo(db, contaID, provider, cfg, dataInicio, dataFim)
}

func syncAccountPeriodo(db *sql.DB, contaID string, provider BankProvider, cfg ProviderConfig, dataInicio, dataFim string) (*SyncResult, error) {
	txs, err := provider.GetExtrato(cfg, dataInicio, dataFim)
	if err != nil {
		return nil, fmt.Errorf("extrato: %w", err)
	}

	res := &SyncResult{DataInicio: dataInicio, DataFim: dataFim}
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
			    (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext, conciliado, detalhes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, true, NULLIF($8,'')::jsonb)`,
			contaID, tx.DataTransacao, tx.Descricao, tx.Valor, tx.Tipo, tx.Categoria, tx.IDExterno, tx.Detalhes,
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
// HorariosSyncBancaria são as horas do dia (horário local do servidor) em que o cron sincroniza.
var HorariosSyncBancaria = []int{6, 12, 18}

// ProximoHorarioSync devolve o próximo horário de sincronização estritamente depois de agora.
func ProximoHorarioSync(agora time.Time, horas []int) time.Time {
	var melhor time.Time
	for dia := 0; dia <= 1; dia++ {
		for _, h := range horas {
			c := time.Date(agora.Year(), agora.Month(), agora.Day()+dia, h, 0, 0, 0, agora.Location())
			if c.After(agora) && (melhor.IsZero() || c.Before(melhor)) {
				melhor = c
			}
		}
	}
	return melhor
}

func StartBankDailySync(db *sql.DB) {
	go func() {
		for {
			next := ProximoHorarioSync(time.Now(), HorariosSyncBancaria)
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
					    data_inicio = $4::date, data_fim = $5::date
					WHERE id = $6`,
					result.Importadas, result.Duplicadas, result.SaldoFinal, result.DataInicio, result.DataFim, logID)
				log.Printf("[bank-cron] %s: %d importadas, %d duplicadas", j.id, result.Importadas, result.Duplicadas)
			}
		}
	}()
}
