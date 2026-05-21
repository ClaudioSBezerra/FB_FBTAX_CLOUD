package services

import (
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const interBaseURL = "https://cdpj.partners.bancointer.com.br"

// ── mTLS client ──────────────────────────────────────────────────────────────

func buildInterHTTPClient() (*http.Client, error) {
	certPEM := os.Getenv("INTER_CERT_PEM")
	keyPEM := os.Getenv("INTER_KEY_PEM")
	if certPEM == "" || keyPEM == "" {
		return nil, fmt.Errorf("INTER_CERT_PEM ou INTER_KEY_PEM não configurados")
	}

	// Accept base64-encoded or raw PEM
	certBytes := decodePEMEnv(certPEM)
	keyBytes := decodePEMEnv(keyPEM)

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return nil, fmt.Errorf("certificado Inter inválido: %w", err)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	transport := &http.Transport{TLSClientConfig: tlsCfg}
	return &http.Client{Transport: transport, Timeout: 30 * time.Second}, nil
}

func decodePEMEnv(s string) []byte {
	// Se não contém "-----BEGIN", assume base64
	if !strings.Contains(s, "-----BEGIN") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
		if err == nil {
			return decoded
		}
	}
	// Substitui \n literal por quebra de linha real (para env vars em linha única)
	return []byte(strings.ReplaceAll(s, `\n`, "\n"))
}

// ── OAuth2 token ──────────────────────────────────────────────────────────────

type InterToken struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"-"`
}

var interTokenCache *InterToken

func GetInterToken() (*InterToken, error) {
	if interTokenCache != nil && time.Now().Before(interTokenCache.ExpiresAt) {
		return interTokenCache, nil
	}

	clientID := os.Getenv("INTER_CLIENT_ID")
	clientSecret := os.Getenv("INTER_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("INTER_CLIENT_ID ou INTER_CLIENT_SECRET não configurados")
	}

	client, err := buildInterHTTPClient()
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("scope", "extrato.read saldo.read")

	req, _ := http.NewRequest("POST", interBaseURL+"/oauth/v2/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token Inter: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("token Inter HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var tok InterToken
	if err := json.Unmarshal(raw, &tok); err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if tok.ExpiresIn <= 0 {
		tok.ExpiresIn = 3600
	}
	tok.ExpiresAt = time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second)
	interTokenCache = &tok
	return &tok, nil
}

// ── Saldo ─────────────────────────────────────────────────────────────────────

type InterSaldo struct {
	Disponivel float64 `json:"disponivel"`
	Bloqueado  float64 `json:"bloqueado"`
	Total      float64 `json:"total"`
}

func GetInterSaldo() (*InterSaldo, error) {
	tok, err := GetInterToken()
	if err != nil {
		return nil, err
	}
	client, err := buildInterHTTPClient()
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", interBaseURL+"/banking/v3/saldo", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("x-conta-corrente", os.Getenv("INTER_CONTA_CORRENTE"))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("saldo Inter: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("saldo Inter HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var saldo InterSaldo
	if err := json.Unmarshal(raw, &saldo); err != nil {
		return nil, fmt.Errorf("parse saldo: %w", err)
	}
	saldo.Total = saldo.Disponivel + saldo.Bloqueado
	return &saldo, nil
}

// ── Extrato ───────────────────────────────────────────────────────────────────

type InterTransacao struct {
	DataTransacao  string  `json:"dataTransacao"`
	Tipo           string  `json:"tipoTransacao"` // CREDITO | DEBITO
	Valor          float64 `json:"valor"`
	Descricao      string  `json:"descricao"`
	IDTransacao    string  `json:"idTransacao"`
	TipoOperacao   string  `json:"tipoOperacao"`
}

type InterExtratoResp struct {
	Transacoes []InterTransacao `json:"transacoes"`
}

func GetInterExtrato(dataInicio, dataFim string) ([]InterTransacao, error) {
	tok, err := GetInterToken()
	if err != nil {
		return nil, err
	}
	client, err := buildInterHTTPClient()
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/banking/v3/extrato?dataInicio=%s&dataFim=%s", interBaseURL, dataInicio, dataFim)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("x-conta-corrente", os.Getenv("INTER_CONTA_CORRENTE"))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("extrato Inter: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("extrato Inter HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var ex InterExtratoResp
	if err := json.Unmarshal(raw, &ex); err != nil {
		return nil, fmt.Errorf("parse extrato: %w", err)
	}
	return ex.Transacoes, nil
}

// ── Sync pipeline ─────────────────────────────────────────────────────────────

type InterSyncResult struct {
	Importadas  int
	Duplicadas  int
	SaldoFinal  float64
	ErrDetalhe  string
}

func SyncInterAccount(db *sql.DB, contaID string) (*InterSyncResult, error) {
	res := &InterSyncResult{}

	dataFim := time.Now().Format("2006-01-02")
	dataInicio := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	// Busca última sync para saber de quando puxar
	var ultimaSync sql.NullTime
	db.QueryRow(`SELECT ultima_sync FROM financeiro.contas_financeiras WHERE id = $1`, contaID).Scan(&ultimaSync)
	if ultimaSync.Valid {
		dataInicio = ultimaSync.Time.AddDate(0, 0, -1).Format("2006-01-02")
	}

	txs, err := GetInterExtrato(dataInicio, dataFim)
	if err != nil {
		return nil, err
	}

	for _, tx := range txs {
		tipo := strings.ToLower(tx.Tipo)
		if tipo != "credito" && tipo != "debito" {
			continue
		}
		var exists bool
		db.QueryRow(`SELECT EXISTS(SELECT 1 FROM financeiro.transacoes WHERE referencia_ext = $1)`, tx.IDTransacao).Scan(&exists)
		if exists {
			res.Duplicadas++
			continue
		}
		_, err := db.Exec(`
			INSERT INTO financeiro.transacoes (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext, conciliado)
			VALUES ($1, $2, $3, $4, $5, $6, $7, true)`,
			contaID, tx.DataTransacao, tx.Descricao, tx.Valor, tipo, tx.TipoOperacao, tx.IDTransacao,
		)
		if err != nil {
			log.Printf("[inter-sync] erro ao inserir tx %s: %v", tx.IDTransacao, err)
			continue
		}
		res.Importadas++
	}

	// Atualiza saldo e ultima_sync
	saldo, err := GetInterSaldo()
	if err == nil {
		res.SaldoFinal = saldo.Disponivel
		db.Exec(`UPDATE financeiro.contas_financeiras SET saldo = $1, ultima_sync = NOW() WHERE id = $2`,
			saldo.Disponivel, contaID)
	} else {
		// Atualiza apenas ultima_sync
		db.Exec(`UPDATE financeiro.contas_financeiras SET ultima_sync = NOW() WHERE id = $1`, contaID)
	}

	return res, nil
}

// ── Cron diário ───────────────────────────────────────────────────────────────

func StartInterDailySync(db *sql.DB) {
	go func() {
		for {
			now := time.Now()
			// Próxima execução às 06:00
			next := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))

			rows, err := db.Query(`
				SELECT id FROM financeiro.contas_financeiras
				WHERE ativa = true AND provedor = 'inter'`)
			if err != nil {
				log.Printf("[inter-cron] erro ao listar contas: %v", err)
				continue
			}
			var ids []string
			for rows.Next() {
				var id string
				rows.Scan(&id)
				ids = append(ids, id)
			}
			rows.Close()

			for _, id := range ids {
				log.Printf("[inter-cron] sincronizando conta %s", id)
				result, err := SyncInterAccount(db, id)
				if err != nil {
					log.Printf("[inter-cron] erro conta %s: %v", id, err)
					db.Exec(`
						INSERT INTO financeiro.inter_sync_log (conta_id, encerrado_em, status, erro_detalhe)
						VALUES ($1, NOW(), 'erro', $2)`, id, err.Error())
					continue
				}
				db.Exec(`
					INSERT INTO financeiro.inter_sync_log
					    (conta_id, encerrado_em, status, data_inicio, data_fim, tx_importadas, tx_duplicadas, saldo_final)
					VALUES ($1, NOW(), 'ok', NOW()-INTERVAL '30 days', NOW(), $2, $3, $4)`,
					id, result.Importadas, result.Duplicadas, result.SaldoFinal)
				log.Printf("[inter-cron] conta %s: %d importadas, %d duplicadas", id, result.Importadas, result.Duplicadas)
			}
		}
	}()
}
