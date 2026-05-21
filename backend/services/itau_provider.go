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

// ItauProvider implementa BankProvider para o Itaú PJ.
// Usa OAuth2 client_credentials. mTLS é opcional: se CertPEM/KeyPEM
// estiverem presentes no ProviderConfig, o client TLS os usa.
type ItauProvider struct{}

const itauDefaultBaseURL    = "https://api.itau.com.br"
const itauDefaultTokenURL   = "https://sts.itau.com.br/api/oauth/token"

// ── Token cache por clientID ──────────────────────────────────────────────────

type itauCachedToken struct {
	accessToken string
	expiresAt   time.Time
}

var itauTokenCache sync.Map // key: clientID → *itauCachedToken

func (p *ItauProvider) getToken(cfg ProviderConfig) (string, error) {
	tokenURL := itauDefaultTokenURL
	if cfg.BaseURL != "" {
		tokenURL = cfg.BaseURL + "/oauth/token"
	}

	if v, ok := itauTokenCache.Load(cfg.ClientID); ok {
		cached := v.(*itauCachedToken)
		if time.Now().Before(cached.expiresAt) {
			return cached.accessToken, nil
		}
	}

	// Itaú: mTLS é opcional dependendo do produto contratado
	client, err := buildMTLSClient(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)

	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("itau token: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("itau token HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &tok); err != nil {
		return "", fmt.Errorf("itau token parse: %w", err)
	}
	if tok.ExpiresIn <= 0 {
		tok.ExpiresIn = 3600
	}

	itauTokenCache.Store(cfg.ClientID, &itauCachedToken{
		accessToken: tok.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second),
	})
	return tok.AccessToken, nil
}

// ── BankProvider ─────────────────────────────────────────────────────────────

func (p *ItauProvider) GetSaldo(cfg ProviderConfig) (*BankSaldo, error) {
	baseURL := itauDefaultBaseURL
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	token, err := p.getToken(cfg)
	if err != nil {
		return nil, err
	}
	client, err := buildMTLSClient(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return nil, err
	}

	// Itaú Cash Management PJ — saldo
	u := fmt.Sprintf("%s/cash_management/v2/saldo?agencia=%s&conta=%s", baseURL, cfg.Agencia, cfg.Conta)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("itau saldo: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("itau saldo HTTP %d: %s", resp.StatusCode, string(raw))
	}

	// Itaú retorna saldo em formato nested
	var s struct {
		Saldo struct {
			Disponivel float64 `json:"disponivel"`
			Bloqueado  float64 `json:"bloqueado"`
		} `json:"saldo"`
		// Alguns endpoints retornam direto na raiz
		Disponivel float64 `json:"disponivel"`
		Bloqueado  float64 `json:"bloqueado"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("itau saldo parse: %w", err)
	}

	disp := s.Saldo.Disponivel
	if disp == 0 {
		disp = s.Disponivel
	}
	bloc := s.Saldo.Bloqueado
	if bloc == 0 {
		bloc = s.Bloqueado
	}
	return &BankSaldo{Disponivel: disp, Bloqueado: bloc}, nil
}

func (p *ItauProvider) GetExtrato(cfg ProviderConfig, dataInicio, dataFim string) ([]BankTransacao, error) {
	baseURL := itauDefaultBaseURL
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	token, err := p.getToken(cfg)
	if err != nil {
		return nil, err
	}
	client, err := buildMTLSClient(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return nil, err
	}

	// Itaú Cash Management PJ — extrato conta corrente
	u := fmt.Sprintf(
		"%s/cash_management/v2/extrato/conta-corrente?agencia=%s&conta=%s&dataInicio=%s&dataFim=%s",
		baseURL, cfg.Agencia, cfg.Conta, dataInicio, dataFim,
	)
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("itau extrato: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("itau extrato HTTP %d: %s", resp.StatusCode, string(raw))
	}

	// Itaú retorna lista em campo "data" ou direto como array
	var wrapper struct {
		Data []struct {
			Data      string  `json:"data"`       // YYYY-MM-DD
			Historico string  `json:"historico"`  // descrição
			Valor     float64 `json:"valor"`
			Tipo      string  `json:"tipo"`       // D=Débito C=Crédito
			Documento string  `json:"documento"`  // ID único
			TipoLancamento string `json:"tipo_lancamento"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, fmt.Errorf("itau extrato parse: %w", err)
	}

	txs := make([]BankTransacao, 0, len(wrapper.Data))
	for _, t := range wrapper.Data {
		tipo := "credito"
		if strings.ToUpper(t.Tipo) == "D" {
			tipo = "debito"
		}
		txs = append(txs, BankTransacao{
			IDExterno:     t.Documento,
			DataTransacao: t.Data,
			Descricao:     t.Historico,
			Valor:         t.Valor,
			Tipo:          tipo,
			Categoria:     t.TipoLancamento,
		})
	}
	return txs, nil
}
