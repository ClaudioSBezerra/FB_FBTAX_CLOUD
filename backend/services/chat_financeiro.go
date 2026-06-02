package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// ── System prompts ────────────────────────────────────────────────────────────

const financeiroSQLPrompt = `Você é um analista financeiro que converte perguntas em português para SQL PostgreSQL.

REGRAS ABSOLUTAS:
- Responda APENAS com a query SQL final, dentro de um único bloco ` + "```sql" + `…` + "```" + `.
- NÃO adicione explicação, texto ou comentário fora do bloco SQL.
- Use APENAS as tabelas listadas abaixo.
- A query deve começar com SELECT ou WITH.
- NUNCA use INSERT, UPDATE, DELETE, ALTER, DROP, CREATE, GRANT, REVOKE, TRUNCATE ou qualquer DDL/DML.
- Não inclua ponto-e-vírgula no meio da query.
- Use sempre LIMIT (máximo 200).
- Quando o usuário disser "este ano" → EXTRACT(YEAR FROM data_transacao) = EXTRACT(YEAR FROM CURRENT_DATE).
- "este mês" → DATE_TRUNC('month', data_transacao) = DATE_TRUNC('month', CURRENT_DATE).
- "últimos 30 dias" → data_transacao >= CURRENT_DATE - INTERVAL '30 days'.
- Para nomes de categorias/descrições use ILIKE '%termo%' (case-insensitive).
- Para "maior/menor X" (singular): use ORDER BY valor DESC/ASC LIMIT 1 — NÃO use MAX/MIN nem traga o oposto na mesma query.
- Para "top N maiores/menores": use ORDER BY valor DESC/ASC LIMIT N.
- "Maior gasto/saída" → tipo='debito' ORDER BY valor DESC LIMIT 1. "Maior entrada/recebimento" → tipo='credito' ORDER BY valor DESC LIMIT 1. "Maior valor" sem qualificar → ignora tipo, ORDER BY valor DESC LIMIT 1.
- Valor é SEMPRE positivo; a direção (entrada/saída) está em tipo. NUNCA escreva "valor < 0".
- Para contagem ("quantos/quantas"): use COUNT(*). Para média: AVG(valor).

TABELAS DISPONÍVEIS:

financeiro.transacoes — cada entrada/saída de dinheiro
  id UUID, conta_id UUID, data_transacao DATE, descricao VARCHAR,
  valor NUMERIC (sempre positivo), tipo VARCHAR ('credito'=entrada / 'debito'=saída),
  categoria VARCHAR, conciliado BOOLEAN, created_at TIMESTAMPTZ

financeiro.contas_financeiras — contas bancárias cadastradas
  id UUID, apelido VARCHAR, banco VARCHAR, agencia VARCHAR, conta VARCHAR,
  tipo VARCHAR, saldo NUMERIC (saldo atual), ativa BOOLEAN, ultima_sync TIMESTAMPTZ

EXEMPLOS:

Usuário: "Quais despesas de Uber em janeiro?"
` + "```sql" + `
SELECT data_transacao, descricao, valor
FROM financeiro.transacoes
WHERE tipo = 'debito'
  AND descricao ILIKE '%uber%'
  AND data_transacao BETWEEN '2026-01-01' AND '2026-01-31'
ORDER BY data_transacao DESC
LIMIT 100
` + "```" + `

Usuário: "Total de entradas e saídas por mês em 2026"
` + "```sql" + `
SELECT
  TO_CHAR(data_transacao, 'YYYY-MM') AS mes,
  SUM(CASE WHEN tipo='credito' THEN valor ELSE 0 END) AS entradas,
  SUM(CASE WHEN tipo='debito'  THEN valor ELSE 0 END) AS saidas,
  SUM(CASE WHEN tipo='credito' THEN valor ELSE -valor END) AS resultado
FROM financeiro.transacoes
WHERE EXTRACT(YEAR FROM data_transacao) = 2026
GROUP BY mes
ORDER BY mes
LIMIT 12
` + "```" + `

Usuário: "Qual o saldo atual de todas as contas e quanto gastei este ano?"
` + "```sql" + `
SELECT
  (SELECT COALESCE(SUM(saldo),0) FROM financeiro.contas_financeiras WHERE ativa = true) AS saldo_total,
  COALESCE(SUM(CASE WHEN tipo='credito' THEN valor ELSE 0 END),0) AS entradas_ano,
  COALESCE(SUM(CASE WHEN tipo='debito'  THEN valor ELSE 0 END),0) AS saidas_ano,
  COALESCE(SUM(CASE WHEN tipo='credito' THEN valor ELSE -valor END),0) AS resultado_liquido
FROM financeiro.transacoes
WHERE EXTRACT(YEAR FROM data_transacao) = EXTRACT(YEAR FROM CURRENT_DATE)
LIMIT 1
` + "```" + `

Usuário: "Gastos por categoria no último mês"
` + "```sql" + `
SELECT
  COALESCE(categoria, 'Sem categoria') AS categoria,
  COUNT(*) AS qtd,
  SUM(valor) AS total
FROM financeiro.transacoes
WHERE tipo = 'debito'
  AND data_transacao >= DATE_TRUNC('month', CURRENT_DATE - INTERVAL '1 month')
  AND data_transacao <  DATE_TRUNC('month', CURRENT_DATE)
GROUP BY categoria
ORDER BY total DESC
LIMIT 20
` + "```" + `

Usuário: "Qual o maior valor no extrato?"
` + "```sql" + `
SELECT data_transacao, descricao, valor, tipo
FROM financeiro.transacoes
ORDER BY valor DESC
LIMIT 1
` + "```" + `

Usuário: "Qual foi a maior saída deste mês?"
` + "```sql" + `
SELECT data_transacao, descricao, valor
FROM financeiro.transacoes
WHERE tipo = 'debito'
  AND DATE_TRUNC('month', data_transacao) = DATE_TRUNC('month', CURRENT_DATE)
ORDER BY valor DESC
LIMIT 1
` + "```" + `

Usuário: "Top 5 maiores despesas do ano"
` + "```sql" + `
SELECT data_transacao, descricao, valor, COALESCE(categoria,'Sem categoria') AS categoria
FROM financeiro.transacoes
WHERE tipo = 'debito'
  AND EXTRACT(YEAR FROM data_transacao) = EXTRACT(YEAR FROM CURRENT_DATE)
ORDER BY valor DESC
LIMIT 5
` + "```" + `

Usuário: "Quantos Pix enviei nos últimos 30 dias?"
` + "```sql" + `
SELECT COUNT(*) AS quantidade, SUM(valor) AS total
FROM financeiro.transacoes
WHERE tipo = 'debito'
  AND descricao ILIKE '%pix%'
  AND data_transacao >= CURRENT_DATE - INTERVAL '30 days'
LIMIT 1
` + "```" + `

Usuário: "Quanto recebi de salário este ano?"
` + "```sql" + `
SELECT COALESCE(SUM(valor),0) AS total, COUNT(*) AS pagamentos
FROM financeiro.transacoes
WHERE tipo = 'credito'
  AND descricao ILIKE '%salario%'
  AND EXTRACT(YEAR FROM data_transacao) = EXTRACT(YEAR FROM CURRENT_DATE)
LIMIT 1
` + "```"

const financeiroNarrativaPrompt = `Você é um analista financeiro pessoal. Receberá:
1. A pergunta original do usuário
2. O resultado de uma consulta ao banco de dados em JSON

Sua tarefa: responder diretamente à pergunta do usuário em português, usando os números exatos do resultado.

REGRAS:
- Seja direto e prático — máximo 4 frases.
- Responda APENAS o que foi perguntado. Se a pergunta foi "qual o maior", mostre só o maior — NÃO inclua o menor nem qualquer comparação que o usuário não pediu.
- Se a pergunta envolver cálculo (parcelas, capacidade de gasto), FAÇA o cálculo com os dados e apresente o resultado.
- Para "quantas parcelas": use saldo_total e resultado_liquido_mensal para calcular.
- Use formatação monetária brasileira: R$ 1.234,56.
- Para transações: cite data (DD/MM), descrição resumida e valor. Ex: "Em 27/05 você gastou R$ 735,00 com Stravaganza Padaria."
- Lembre que o campo 'valor' é sempre positivo; o sinal de entrada/saída está em 'tipo' ('credito' = entrada, 'debito' = saída).
- Se o resultado for vazio: informe gentilmente que não há dados para o período.
- Não repita o JSON, não invente dados, não dê saudações.`

// ── Tipos ─────────────────────────────────────────────────────────────────────

type FinanceiroQueryResult struct {
	Reply    string                   `json:"reply"`
	SQL      string                   `json:"sql"`
	Columns  []string                 `json:"columns"`
	Rows     []map[string]interface{} `json:"rows"`
	Truncado bool                     `json:"truncado"`
	Erro     string                   `json:"erro,omitempty"`
}

type FinanceiroChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ── Cliente Z.AI ──────────────────────────────────────────────────────────────

type zaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func chamarZAIFinanceiro(systemPrompt, userMsg string, maxTokens int) (string, error) {
	apiKey := os.Getenv("ZAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ZAI_API_KEY não configurada")
	}
	body, _ := json.Marshal(map[string]interface{}{
		"model":       "glm-4.5-air",
		"max_tokens":  maxTokens,
		"temperature": 0.1,
		"messages": []zaiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
	})
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", "https://api.z.ai/api/coding/paas/v4/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Z.AI status %d: %s", resp.StatusCode, string(raw))
	}
	var r struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", err
	}
	if len(r.Choices) == 0 {
		return "", fmt.Errorf("sem resposta da IA")
	}
	out := r.Choices[0].Message.Content
	if out == "" {
		out = r.Choices[0].Message.ReasoningContent
	}
	return strings.TrimSpace(out), nil
}

// ── Validação de SQL financeiro ───────────────────────────────────────────────

var financeiroAllowedTables = map[string]bool{
	"financeiro.transacoes":           true,
	"financeiro.contas_financeiras":   true,
	"transacoes":                      true,
	"contas_financeiras":              true,
}

var financeiroForbiddenKW = []string{
	"insert", "update", "delete", "drop", "truncate", "alter", "create",
	"grant", "revoke", "copy", "vacuum", "do", "execute", "perform", "call",
	"pg_", "current_user", "session_user",
}

var rxFromJoinFin = regexp.MustCompile(`(?i)\b(?:from|join)\s+([a-z_][a-z0-9_.]*)`)
var rxSQLBlockFin = regexp.MustCompile("(?s)```(?:sql)?\\s*(.*?)```")
var rxBalancedParens = regexp.MustCompile(`\([^()]*\)`)

func extrairSQLFin(texto string) string {
	if m := rxSQLBlockFin.FindStringSubmatch(texto); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return strings.TrimSpace(texto)
}

// stripParenContent remove iterativamente todo conteúdo em parênteses balanceados.
// Útil para isolar a estrutura top-level do SELECT antes de procurar referências
// de tabela — evita falsos positivos com EXTRACT(YEAR FROM col), SUBSTRING(... FROM ...),
// TRIM(... FROM ...), POSITION(... IN ...) etc., que usam FROM como keyword interno.
// Subqueries também são removidas; aceitável porque o modelo é restrito por prompt
// e o timeout/read-only blindam o blast radius.
func stripParenContent(s string) string {
	for i := 0; i < 20; i++ { // limite de profundidade pra evitar loop em caso patológico
		next := rxBalancedParens.ReplaceAllString(s, "")
		if next == s {
			return s
		}
		s = next
	}
	return s
}

func validarSQLFinanceiro(rawSQL string) (string, error) {
	clean := strings.TrimRight(strings.TrimSpace(rawSQL), "; \n\t")
	if clean == "" {
		return "", fmt.Errorf("SQL vazio")
	}
	lower := strings.ToLower(clean)
	if !strings.HasPrefix(lower, "select") && !strings.HasPrefix(lower, "with") {
		return "", fmt.Errorf("apenas SELECT/WITH permitidos")
	}
	if strings.Contains(clean, ";") {
		return "", fmt.Errorf("';' não permitido")
	}
	for _, kw := range financeiroForbiddenKW {
		if regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(kw)+`\b`).MatchString(clean) {
			return "", fmt.Errorf("palavra-chave proibida: %s", kw)
		}
	}
	// Procura referências de tabela na estrutura sem parênteses balanceados
	// (evita matchar FROM dentro de EXTRACT/SUBSTRING/TRIM/POSITION).
	skeleton := stripParenContent(clean)
	for _, m := range rxFromJoinFin.FindAllStringSubmatch(skeleton, -1) {
		if len(m) < 2 {
			continue
		}
		obj := strings.ToLower(m[1])
		if !financeiroAllowedTables[obj] {
			return "", fmt.Errorf("tabela não permitida: %s", m[1])
		}
	}
	if !regexp.MustCompile(`(?i)\blimit\s+\d+`).MatchString(clean) {
		clean += "\nLIMIT 200"
	}
	return clean, nil
}

// ── Pipeline principal ────────────────────────────────────────────────────────

func ResponderPerguntaFinanceiro(db *sql.DB, pergunta string, historico []FinanceiroChatMsg) (*FinanceiroQueryResult, error) {
	// Monta prompt com histórico (até 4 trocas anteriores)
	userPrompt := pergunta
	if len(historico) > 0 {
		var sb strings.Builder
		sb.WriteString("CONVERSA ANTERIOR:\n")
		start := 0
		if len(historico) > 4 {
			start = len(historico) - 4
		}
		for _, m := range historico[start:] {
			text := m.Content
			if len(text) > 400 {
				text = text[:400] + "…"
			}
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, text))
		}
		sb.WriteString("\nPERGUNTA ATUAL: ")
		sb.WriteString(pergunta)
		userPrompt = sb.String()
	}

	// 1. Gera SQL
	respIA, err := chamarZAIFinanceiro(financeiroSQLPrompt, userPrompt, 600)
	if err != nil {
		return nil, fmt.Errorf("IA indisponível: %w", err)
	}
	sqlGerado := extrairSQLFin(respIA)
	log.Printf("[chat-fin] pergunta=%q sql=%q", pergunta, sqlGerado)

	// 2. Valida SQL
	sqlClean, err := validarSQLFinanceiro(sqlGerado)
	if err != nil {
		return nil, fmt.Errorf("SQL inválido (%s)", err)
	}

	// 3. Executa em transação read-only com timeout
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("início de transação: %w", err)
	}
	defer tx.Rollback()
	tx.ExecContext(ctx, "SET LOCAL statement_timeout = '6s'")

	rows, err := tx.QueryContext(ctx, sqlClean)
	if err != nil {
		return nil, fmt.Errorf("execução: %w", err)
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	const maxRows = 200
	var resultado []map[string]interface{}
	truncado := false
	for rows.Next() {
		if len(resultado) >= maxRows {
			truncado = true
			break
		}
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		linha := map[string]interface{}{}
		for i, c := range cols {
			v := vals[i]
			switch x := v.(type) {
			case []byte:
				linha[c] = string(x)
			case time.Time:
				linha[c] = x.Format("2006-01-02")
			default:
				linha[c] = x
			}
		}
		resultado = append(resultado, linha)
	}

	// 4. Narrativa
	jsonResult, _ := json.Marshal(resultado)
	jsonStr := string(jsonResult)
	if len(jsonStr) > 3000 {
		jsonStr = jsonStr[:3000] + "...(truncado)"
	}
	narrativaPrompt := fmt.Sprintf(
		"Pergunta: %s\n\nResultado (%d linha(s)%s):\n%s",
		pergunta,
		len(resultado),
		map[bool]string{true: ", truncado em 200", false: ""}[truncado],
		jsonStr,
	)
	narrativa, err := chamarZAIFinanceiro(financeiroNarrativaPrompt, narrativaPrompt, 350)
	if err != nil {
		log.Printf("[chat-fin] narrativa falhou: %v", err)
		narrativa = fmt.Sprintf("Encontrei %d resultado(s). Veja a tabela abaixo.", len(resultado))
	}

	if resultado == nil {
		resultado = []map[string]interface{}{}
	}
	return &FinanceiroQueryResult{
		Reply:    narrativa,
		SQL:      sqlClean,
		Columns:  cols,
		Rows:     resultado,
		Truncado: truncado,
	}, nil
}
