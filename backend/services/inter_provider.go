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

// interEscopoBanking são os escopos pedidos nas chamadas de saldo/extrato. O saldo é coberto
// por extrato.read; um escopo a mais que a integração não tenha faz o Inter recusar o token.
func interEscopoBanking(cfg ProviderConfig) string {
	if e := strings.TrimSpace(cfg.Scopes); e != "" {
		return e
	}
	return "extrato.read"
}

func (p *InterProvider) getToken(cfg ProviderConfig) (string, error) {
	return interObterToken(cfg, interEscopoBanking(cfg))
}

// interObterToken pede (ou reaproveita do cache) um token OAuth com exatamente os escopos dados.
func interObterToken(cfg ProviderConfig, escopo string) (string, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = interDefaultBaseURL
	}

	// Verifica cache
	chaveCache := cfg.ClientID + "|" + escopo
	if v, ok := interTokenCache.Load(chaveCache); ok {
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
	form.Set("scope", escopo)

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

	interTokenCache.Store(chaveCache, &interCachedToken{
		accessToken: tok.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second),
	})
	return tok.AccessToken, nil
}

// ── BankProvider ─────────────────────────────────────────────────────────────

func (p *InterProvider) GetSaldo(cfg ProviderConfig) (*BankSaldo, error) {
	raw, _, err := interBankingGET(cfg, "/saldo", "")
	if err != nil {
		return nil, fmt.Errorf("inter saldo: %w", err)
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

// interBankingGET chama um endpoint do Banking tentando v2 e, se a rota não existir (404), v3.
// Devolve o corpo e a versão que respondeu.
func interBankingGET(cfg ProviderConfig, caminho, query string) ([]byte, string, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = interDefaultBaseURL
	}
	token, err := interObterToken(cfg, interEscopoBanking(cfg))
	if err != nil {
		return nil, "", err
	}
	client, err := buildMTLSClient(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return nil, "", err
	}
	var ultimo error
	for _, versao := range []string{"v2", "v3"} {
		u := baseURL + "/banking/" + versao + caminho
		if query != "" {
			u += "?" + query
		}
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("x-conta-corrente", cfg.ContaCorrente)
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", err
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return raw, versao, nil
		}
		ultimo = fmt.Errorf("HTTP %d (%s): %s", resp.StatusCode, versao, strings.TrimSpace(string(raw)))
		if resp.StatusCode != http.StatusNotFound {
			return nil, versao, ultimo
		}
	}
	return nil, "", ultimo
}

func (p *InterProvider) GetExtrato(cfg ProviderConfig, dataInicio, dataFim string) ([]BankTransacao, error) {
	raw, _, err := interBankingGET(cfg, "/extrato", "dataInicio="+dataInicio+"&dataFim="+dataFim)
	if err != nil {
		return nil, fmt.Errorf("inter extrato: %w", err)
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
