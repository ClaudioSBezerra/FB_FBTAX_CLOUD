package services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
		if conta := interContaCorrente(cfg.ContaCorrente); conta != "" {
			req.Header.Set("x-conta-corrente", conta)
		}
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

// interExtratoPorPagina é o tamanho de página do extrato enriquecido (o Inter aceita até 10.000).
const interExtratoPorPagina = 1000

// interNum lê um valor que o Inter pode enviar como número (2500.5) ou como texto ("2500.50").
type interNum float64

func (n *interNum) UnmarshalJSON(b []byte) error {
	t := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if t == "" || t == "null" {
		*n = 0
		return nil
	}
	// Formato brasileiro ("4.500,00"): o ponto é milhar e a vírgula é decimal. Só vírgula ("4500,5")
	// também é decimal; só ponto ("4500.50") já está no formato do Go.
	if strings.Contains(t, ",") {
		t = strings.ReplaceAll(t, ".", "")
		t = strings.ReplaceAll(t, ",", ".")
	}
	f, err := strconv.ParseFloat(t, 64)
	if err != nil {
		return fmt.Errorf("valor inválido %q", t)
	}
	*n = interNum(f)
	return nil
}

// interTransacaoBruta são os campos usados de cada lançamento; o JSON completo é guardado à parte.
type interTransacaoBruta struct {
	DataTransacao string   `json:"dataTransacao"`
	DataInclusao  string   `json:"dataInclusao"`
	DataEntrada   string   `json:"dataEntrada"`
	TipoTransacao string   `json:"tipoTransacao"` // PIX, BOLETO_COBRANCA, TRANSFERENCIA, PAGAMENTO, ...
	TipoOperacao  string   `json:"tipoOperacao"`  // C (entrada) | D (saída)
	Valor         interNum `json:"valor"`
	Titulo        string   `json:"titulo"`
	Descricao     string   `json:"descricao"`
	IDTransacao   string   `json:"idTransacao"`
}

// InterExtratoBruto é o extrato como o Inter devolveu, antes de qualquer interpretação.
type InterExtratoBruto struct {
	Modo   string            // "completo" (enriquecido) ou "simples"
	Brutos []json.RawMessage // um JSON por lançamento
}

// ExtratoBruto busca o extrato (enriquecido, com recuo para o simples) sem interpretar os lançamentos.
func (p *InterProvider) ExtratoBruto(cfg ProviderConfig, dataInicio, dataFim string) (*InterExtratoBruto, error) {
	var brutos []json.RawMessage
	completo := true
	for pagina := 0; pagina < 50; pagina++ {
		q := fmt.Sprintf("dataInicio=%s&dataFim=%s&pagina=%d&tamanhoPagina=%d", dataInicio, dataFim, pagina, interExtratoPorPagina)
		raw, _, err := interBankingGET(cfg, "/extrato/completo", q)
		if err != nil {
			if pagina == 0 {
				completo = false // sem o enriquecido: usa o simples
				break
			}
			return nil, fmt.Errorf("inter extrato (página %d): %w", pagina, err)
		}
		var pg struct {
			UltimaPagina bool              `json:"ultimaPagina"`
			TotalPaginas int               `json:"totalPaginas"`
			Transacoes   []json.RawMessage `json:"transacoes"`
		}
		if err := json.Unmarshal(raw, &pg); err != nil {
			return nil, fmt.Errorf("inter extrato parse: %w", err)
		}
		brutos = append(brutos, pg.Transacoes...)
		if pg.UltimaPagina || len(pg.Transacoes) == 0 || (pg.TotalPaginas > 0 && pagina+1 >= pg.TotalPaginas) {
			break
		}
	}
	if !completo {
		raw, _, err := interBankingGET(cfg, "/extrato", "dataInicio="+dataInicio+"&dataFim="+dataFim)
		if err != nil {
			return nil, fmt.Errorf("inter extrato: %w", err)
		}
		var ex struct {
			Transacoes []json.RawMessage `json:"transacoes"`
		}
		if err := json.Unmarshal(raw, &ex); err != nil {
			return nil, fmt.Errorf("inter extrato parse: %w", err)
		}
		brutos = ex.Transacoes
	}
	modo := "simples"
	if completo {
		modo = "completo"
	}
	return &InterExtratoBruto{Modo: modo, Brutos: brutos}, nil
}

// interConverter transforma um lançamento bruto em BankTransacao. ok=false quando o sentido
// (crédito/débito) não pôde ser determinado; motivo explica o descarte.
func interConverter(bruto json.RawMessage, completo bool) (tx BankTransacao, ok bool, motivo string) {
	var t interTransacaoBruta
	if err := json.Unmarshal(bruto, &t); err != nil {
		return tx, false, "formato ilegível: " + err.Error()
	}
	tipo, categoria := interSentido(t.TipoOperacao, t.TipoTransacao)
	if tipo == "" {
		return tx, false, fmt.Sprintf("sentido desconhecido (tipoOperacao=%q tipoTransacao=%q)", t.TipoOperacao, t.TipoTransacao)
	}
	data := strings.TrimSpace(t.DataTransacao)
	if data == "" {
		data = strings.TrimSpace(t.DataInclusao)
	}
	if data == "" {
		data = strings.TrimSpace(t.DataEntrada)
	}
	if len(data) > 10 {
		data = data[:10]
	}
	desc := strings.TrimSpace(t.Descricao)
	if desc == "" {
		desc = strings.TrimSpace(t.Titulo)
	}
	valor := float64(t.Valor)
	if valor < 0 {
		valor = -valor
	}
	id := strings.TrimSpace(t.IDTransacao)
	if id == "" { // sem id do banco: identificador estável para não duplicar a cada sincronização
		soma := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%.2f|%s|%s", data, tipo, valor, desc, categoria)))
		id = "h-" + hex.EncodeToString(soma[:8])
	}
	det := ""
	if completo {
		det = string(bruto)
	}
	return BankTransacao{IDExterno: id, DataTransacao: data, Descricao: desc, Valor: valor, Tipo: tipo, Categoria: categoria, Detalhes: det}, true, ""
}

// GetExtrato busca o extrato ENRIQUECIDO (/extrato/completo, paginado) e guarda o JSON de cada
// lançamento em Detalhes. Se o Inter não o oferecer para a integração, cai para o extrato simples.
// Lançamentos que não puderam ser interpretados são contados e registrados no log (nunca em silêncio).
func (p *InterProvider) GetExtrato(cfg ProviderConfig, dataInicio, dataFim string) ([]BankTransacao, error) {
	ex, err := p.ExtratoBruto(cfg, dataInicio, dataFim)
	if err != nil {
		return nil, err
	}
	txs := make([]BankTransacao, 0, len(ex.Brutos))
	descartados := 0
	for _, bruto := range ex.Brutos {
		tx, ok, motivo := interConverter(bruto, ex.Modo == "completo")
		if !ok {
			descartados++
			if descartados <= 3 {
				log.Printf("[Inter] lançamento descartado: %s | %s", motivo, limitarTexto(string(bruto), 300))
			}
			continue
		}
		txs = append(txs, tx)
	}
	if descartados > 0 {
		log.Printf("[Inter] extrato %s a %s (%s): %d de %d lançamentos descartados", dataInicio, dataFim, ex.Modo, descartados, len(ex.Brutos))
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

// interContaCorrente normaliza o cabeçalho x-conta-corrente: só dígitos (inclusive o dígito
// verificador) e sem zeros à esquerda, como a referência do Inter exige. Vazio = não enviar o
// cabeçalho (só é necessário quando a integração tem mais de uma conta).
func interContaCorrente(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return strings.TrimLeft(b.String(), "0")
}
