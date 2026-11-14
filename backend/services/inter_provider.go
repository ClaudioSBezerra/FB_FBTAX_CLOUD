package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	scope       string
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

// interTokenMu serializa os pedidos de token e impõe um intervalo mínimo entre eles: o endpoint
// de token do Inter responde 429 a rajadas.
var (
	interTokenMu      sync.Mutex
	interTokenUltimo  time.Time
	interTokenEspaco  = 1200 * time.Millisecond
	interTokenTentati = 4
)

// ErrInterLimite indica que o Inter recusou por excesso de requisições (HTTP 429), e não por
// falta de permissão.
var ErrInterLimite = fmt.Errorf("inter: limite de requisições (HTTP 429)")

// interObterToken pede (ou reaproveita do cache) um token OAuth com exatamente os escopos dados.
func interObterToken(cfg ProviderConfig, escopo string) (string, error) {
	tok, _, err := interObterTokenEscopos(cfg, escopo)
	return tok, err
}

// interObterTokenEscopos devolve o token e os escopos que o Inter diz ter concedido (campo
// "scope" da resposta; vazio se o Inter não informar).
func interObterTokenEscopos(cfg ProviderConfig, escopo string) (token, concedidos string, err error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = interDefaultBaseURL
	}
	chaveCache := cfg.ClientID + "|" + escopo
	if v, ok := interTokenCache.Load(chaveCache); ok {
		cached := v.(*interCachedToken)
		if time.Now().Before(cached.expiresAt) {
			return cached.accessToken, cached.scope, nil
		}
	}
	client, err := buildMTLSClient(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return "", "", err
	}

	for tentativa := 1; ; tentativa++ {
		interTokenMu.Lock()
		if espera := interTokenEspaco - time.Since(interTokenUltimo); espera > 0 {
			time.Sleep(espera)
		}
		interTokenUltimo = time.Now()
		interTokenMu.Unlock()

		form := url.Values{}
		form.Set("grant_type", "client_credentials")
		form.Set("client_id", cfg.ClientID)
		form.Set("client_secret", cfg.ClientSecret)
		form.Set("scope", escopo)
		req, _ := http.NewRequest("POST", baseURL+"/oauth/v2/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := client.Do(req)
		if err != nil {
			return "", "", fmt.Errorf("inter token: %w", err)
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			if tentativa >= interTokenTentati {
				return "", "", fmt.Errorf("%w: %s", ErrInterLimite, strings.TrimSpace(string(raw)))
			}
			espera := time.Duration(tentativa) * 2 * time.Second
			if ra, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && ra > 0 && ra <= 30 {
				espera = time.Duration(ra) * time.Second
			}
			time.Sleep(espera)
			continue
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return "", "", fmt.Errorf("inter token HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		}

		var tok struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
			Scope       string `json:"scope"`
		}
		if err := json.Unmarshal(raw, &tok); err != nil {
			return "", "", fmt.Errorf("inter token parse: %w", err)
		}
		if tok.ExpiresIn <= 0 {
			tok.ExpiresIn = 3600
		}
		interTokenCache.Store(chaveCache, &interCachedToken{
			accessToken: tok.AccessToken, scope: tok.Scope,
			expiresAt: time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second),
		})
		return tok.AccessToken, tok.Scope, nil
	}
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

// interBankingGET chama um endpoint do Banking v2 (a v3 não existe para extrato/saldo, conforme a referência do Inter).
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
	for _, versao := range []string{"v2"} {
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
			DataInclusao  string  `json:"dataInclusao"`
			TipoTransacao string  `json:"tipoTransacao"` // PIX, BOLETO_COBRANCA, TRANSFERENCIA, PAGAMENTO, ...
			TipoOperacao  string  `json:"tipoOperacao"`  // C (entrada) | D (saída)
			Valor         float64 `json:"valor"`
			Titulo        string  `json:"titulo"`
			Descricao     string  `json:"descricao"`
			IDTransacao   string  `json:"idTransacao"`
		} `json:"transacoes"`
	}
	if err := json.Unmarshal(raw, &ex); err != nil {
		return nil, fmt.Errorf("inter extrato parse: %w", err)
	}

	txs := make([]BankTransacao, 0, len(ex.Transacoes))
	for _, t := range ex.Transacoes {
		tipo, categoria := interSentido(t.TipoOperacao, t.TipoTransacao)
		if tipo == "" {
			continue // sentido desconhecido: não arrisca lançar crédito/débito errado
		}
		data := strings.TrimSpace(t.DataTransacao)
		if data == "" {
			data = strings.TrimSpace(t.DataInclusao)
		}
		if len(data) > 10 {
			data = data[:10]
		}
		desc := strings.TrimSpace(t.Descricao)
		if desc == "" {
			desc = strings.TrimSpace(t.Titulo)
		}
		valor := t.Valor
		if valor < 0 {
			valor = -valor
		}
		id := strings.TrimSpace(t.IDTransacao)
		if id == "" { // sem id do banco: identificador estável para não duplicar a cada sincronização
			soma := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%.2f|%s|%s", data, tipo, valor, desc, categoria)))
			id = "h-" + hex.EncodeToString(soma[:8])
		}
		txs = append(txs, BankTransacao{IDExterno: id, DataTransacao: data, Descricao: desc, Valor: valor, Tipo: tipo, Categoria: categoria})
	}
	return txs, nil
}

// interSentido descobre se o lançamento é crédito ou débito. O Inter informa o sentido em
// tipoOperacao ("C"/"D"); aceita também CREDITO/DEBITO em qualquer dos dois campos. Devolve a
// categoria (o outro campo, ex.: PIX).
func interSentido(operacao, transacao string) (tipo, categoria string) {
	classifica := func(v string) string {
		switch strings.ToUpper(strings.TrimSpace(v)) {
		case "C", "CREDITO", "CRÉDITO", "ENTRADA":
			return "credito"
		case "D", "DEBITO", "DÉBITO", "SAIDA", "SAÍDA":
			return "debito"
		}
		return ""
	}
	if t := classifica(operacao); t != "" {
		return t, transacao
	}
	if t := classifica(transacao); t != "" {
		return t, operacao
	}
	return "", ""
}
