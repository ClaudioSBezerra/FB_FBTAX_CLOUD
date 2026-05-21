package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// InterProvider implementa BankProvider para o Banco Inter PJ.
// Credenciais vêm do ProviderConfig por conta — sem variáveis de ambiente globais.
type InterProvider struct{}

const interDefaultBaseURL = "https://cdpj.partners.bancointer.com.br"

// ── Token cache por clientID ──────────────────────────────────────────────────

type interCachedToken struct {
	accessToken string
	expiresAt   time.Time
}

var interTokenCache sync.Map // key: clientID → *interCachedToken

func (p *InterProvider) getToken(cfg ProviderConfig) (string, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = interDefaultBaseURL
	}

	// Verifica cache
	if v, ok := interTokenCache.Load(cfg.ClientID); ok {
		cached := v.(*interCachedToken)
		if time.Now().Before(cached.expiresAt) {
			return cached.accessToken, nil
		}
	}

	client, err := buildMTLSClient(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("scope", "extrato.read saldo.read")

	req, _ := http.NewRequest("POST", baseURL+"/oauth/v2/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("inter token: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("inter token HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", fmt.Errorf("inter token parse: %w", err)
	}
	if tok.ExpiresIn <= 0 {
		tok.ExpiresIn = 3600
	}

	interTokenCache.Store(cfg.ClientID, &interCachedToken{
		accessToken: tok.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second),
	})
	return tok.AccessToken, nil
}

// ── BankProvider ─────────────────────────────────────────────────────────────

func (p *InterProvider) GetSaldo(cfg ProviderConfig) (*BankSaldo, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = interDefaultBaseURL
	}
	token, err := p.getToken(cfg)
	if err != nil {
		return nil, err
	}
	client, err := buildMTLSClient(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", baseURL+"/banking/v3/saldo", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-conta-corrente", cfg.ContaCorrente)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inter saldo: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inter saldo HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var s struct {
		Disponivel float64 `json:"disponivel"`
		Bloqueado  float64 `json:"bloqueado"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("inter saldo parse: %w", err)
	}
	return &BankSaldo{Disponivel: s.Disponivel, Bloqueado: s.Bloqueado}, nil
}

func (p *InterProvider) GetExtrato(cfg ProviderConfig, dataInicio, dataFim string) ([]BankTransacao, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = interDefaultBaseURL
	}
	token, err := p.getToken(cfg)
	if err != nil {
		return nil, err
	}
	client, err := buildMTLSClient(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return nil, err
	}

	u := fmt.Sprintf("%s/banking/v3/extrato?dataInicio=%s&dataFim=%s", baseURL, dataInicio, dataFim)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("x-conta-corrente", cfg.ContaCorrente)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inter extrato: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inter extrato HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var ex struct {
		Transacoes []struct {
			DataTransacao string  `json:"dataTransacao"`
			TipoTransacao string  `json:"tipoTransacao"` // CREDITO | DEBITO
			Valor         float64 `json:"valor"`
			Descricao     string  `json:"descricao"`
			IDTransacao   string  `json:"idTransacao"`
			TipoOperacao  string  `json:"tipoOperacao"`
		} `json:"transacoes"`
	}
	if err := json.Unmarshal(raw, &ex); err != nil {
		return nil, fmt.Errorf("inter extrato parse: %w", err)
	}

	txs := make([]BankTransacao, 0, len(ex.Transacoes))
	for _, t := range ex.Transacoes {
		tipo := strings.ToLower(t.TipoTransacao)
		if tipo != "credito" && tipo != "debito" {
			continue
		}
		txs = append(txs, BankTransacao{
			IDExterno:     t.IDTransacao,
			DataTransacao: t.DataTransacao,
			Descricao:     t.Descricao,
			Valor:         t.Valor,
			Tipo:          tipo,
			Categoria:     t.TipoOperacao,
		})
	}
	return txs, nil
}
