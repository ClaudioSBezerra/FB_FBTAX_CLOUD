package services

// Regras do Contas a Receber: baixa de título, renovação de token por mensalidade paga e
// conciliação automática entre títulos em aberto e créditos do extrato bancário.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// TokenValidadeDias: cada mensalidade paga vale o vencimento pago + 45 dias de token.
	TokenValidadeDias = 45
	// TokenLiberacaoMinimaDias: ao pagar um título muito atrasado, o token liberado vale pelo menos
	// este prazo a partir do pagamento (senão nasceria vencido).
	TokenLiberacaoMinimaDias = 15
	// janelaPagamentoAntecipadoDias: até quantos dias antes do vencimento um crédito ainda é
	// considerado pagamento do título na conciliação automática.
	janelaPagamentoAntecipadoDias = 45
	// janelaCriacaoDias: lançamentos anteriores à criação do título (menos esta folga) não são
	// candidatos: evita casar o pagamento do mês passado, de mesmo valor, com o título deste mês.
	janelaCriacaoDias = 7
)

func novoTokenStr() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "") + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
}

// ValidadeTokenAposPagamento calcula até quando o token vale após o pagamento de uma mensalidade
// com o vencimento dado: vencimento + 45 dias, mas nunca menos que hoje + 15 dias.
func ValidadeTokenAposPagamento(vencimento, hoje time.Time) time.Time {
	alvo := vencimento.AddDate(0, 0, TokenValidadeDias)
	if min := hoje.AddDate(0, 0, TokenLiberacaoMinimaDias); alvo.Before(min) {
		return min
	}
	return alvo
}

// RenovarTokensContrato renova, para cada plano do contrato, o token mais recente: encerra o atual
// e emite um novo (mesmo modelo da reativação manual, com predecessor_id). Se o token atual já está
// ativo e vale até a data alvo ou depois, nada muda. Devolve quantos tokens foram emitidos.
func RenovarTokensContrato(tx *sql.Tx, contratoID string, vencimento, hoje time.Time) (int, error) {
	alvo := ValidadeTokenAposPagamento(vencimento, hoje)
	rows, err := tx.Query(`
		SELECT DISTINCT ON (plano_id) id, plano_id, status, valid_until
		FROM financeiro.tokens WHERE contrato_id = $1
		ORDER BY plano_id, created_at DESC`, contratoID)
	if err != nil {
		return 0, err
	}
	type atual struct {
		id, plano, status string
		ate               time.Time
	}
	var lista []atual
	for rows.Next() {
		var a atual
		if err := rows.Scan(&a.id, &a.plano, &a.status, &a.ate); err != nil {
			rows.Close()
			return 0, err
		}
		lista = append(lista, a)
	}
	rows.Close()

	renovados := 0
	for _, a := range lista {
		if a.status == "ativo" && !a.ate.Before(alvo) {
			continue
		}
		if a.status != "encerrado" {
			if _, err := tx.Exec(`UPDATE financeiro.tokens SET status = 'encerrado', updated_at = NOW() WHERE id = $1`, a.id); err != nil {
				return renovados, err
			}
		}
		if _, err := tx.Exec(`
			INSERT INTO financeiro.tokens (contrato_id, plano_id, token, status, valid_from, valid_until, predecessor_id)
			VALUES ($1, $2, $3, 'ativo', $4, $5, $6)`,
			contratoID, a.plano, novoTokenStr(), hoje.Format("2006-01-02"), alvo.Format("2006-01-02"), a.id); err != nil {
			return renovados, err
		}
		renovados++
	}
	return renovados, nil
}

// BaixarTitulo marca o título como pago. transacaoID liga ao lançamento bancário (opcional na
// baixa manual). Para mensalidade com contrato, renova os tokens. Roda dentro da transação dada.
func BaixarTitulo(tx *sql.Tx, tituloID string, transacaoID *string, pagoEm time.Time, valorPago float64, origem, obs string) (renovados int, err error) {
	var tipo, status string
	var contrato sql.NullString
	var vencimento time.Time
	err = tx.QueryRow(`SELECT tipo, contrato_id, vencimento, status FROM financeiro.titulos_receber WHERE id = $1 FOR UPDATE`, tituloID).
		Scan(&tipo, &contrato, &vencimento, &status)
	if err == sql.ErrNoRows {
		return 0, errors.New("título não encontrado")
	}
	if err != nil {
		return 0, err
	}
	if status != "aberto" {
		return 0, fmt.Errorf("o título está %s e não pode ser baixado", status)
	}
	if valorPago <= 0 {
		return 0, errors.New("valor pago deve ser maior que zero")
	}
	var tid any
	if transacaoID != nil && *transacaoID != "" {
		tid = *transacaoID
	}
	if _, err = tx.Exec(`
		UPDATE financeiro.titulos_receber
		SET status = 'pago', pago_em = $2, valor_pago = $3, transacao_id = $4, baixa_origem = $5,
		    baixa_obs = NULLIF($6, ''), updated_at = NOW()
		WHERE id = $1`, tituloID, pagoEm.Format("2006-01-02"), valorPago, tid, origem, obs); err != nil {
		return 0, fmt.Errorf("gravar baixa: %w", err)
	}
	if tipo == "mensalidade" && contrato.Valid {
		return RenovarTokensContrato(tx, contrato.String, vencimento, time.Now())
	}
	return 0, nil
}

// raizCNPJ devolve os 8 primeiros dígitos (a raiz identifica a empresa; matriz e filiais diferem
// só no final).
func raizCNPJ(s string) string {
	d := soDigitos(s)
	if len(d) >= 8 {
		return d[:8]
	}
	return ""
}

// ConciliarTitulos baixa automaticamente títulos em aberto com créditos do extrato quando:
//   - o CNPJ do pagador (detalhe do extrato enriquecido) tem a mesma raiz de um CNPJ do cliente;
//   - o valor é igual ao do título;
//   - a data está entre (vencimento − 45 dias) e hoje, e não é anterior à criação do título (− 7 dias);
//   - o lançamento ainda não baixou outro título.
//
// Para vários títulos de mesmo cliente e valor, o mais antigo casa com o pagamento mais antigo; se
// houver mais pagamentos candidatos que títulos, é ambíguo e nada é baixado (fica para o gestor).
func ConciliarTitulos(ctx context.Context, db *sql.DB) (baixados int, err error) {
	type titulo struct {
		id, cliente, valor string
		venc, criado       time.Time
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, cliente_id, valor::text, vencimento, created_at
		FROM financeiro.titulos_receber WHERE status = 'aberto' ORDER BY vencimento, created_at`)
	if err != nil {
		return 0, err
	}
	var abertos []titulo
	for rows.Next() {
		var t titulo
		if err := rows.Scan(&t.id, &t.cliente, &t.valor, &t.venc, &t.criado); err != nil {
			rows.Close()
			return 0, err
		}
		abertos = append(abertos, t)
	}
	rows.Close()

	// agrupa por (cliente, valor), preservando a ordem por vencimento
	grupos := map[string][]titulo{}
	var ordem []string
	for _, t := range abertos {
		k := t.cliente + "|" + t.valor
		if _, ok := grupos[k]; !ok {
			ordem = append(ordem, k)
		}
		grupos[k] = append(grupos[k], t)
	}

	for _, k := range ordem {
		g := grupos[k]
		raizes, err := raizesDoCliente(ctx, db, g[0].cliente)
		if err != nil || len(raizes) == 0 {
			continue
		}
		cands, err := candidatosCredito(ctx, db, g[0].valor, g[0].venc.AddDate(0, 0, -janelaPagamentoAntecipadoDias), g[0].criado.AddDate(0, 0, -janelaCriacaoDias), raizes)
		if err != nil {
			log.Printf("[Titulos] candidatos: %v", err)
			continue
		}
		if len(cands) == 0 || len(cands) > len(g) {
			if len(cands) > len(g) {
				log.Printf("[Titulos] %d pagamentos para %d títulos do cliente %s: ambíguo, nada baixado", len(cands), len(g), g[0].cliente)
			}
			continue
		}
		for i, c := range cands {
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				return baixados, err
			}
			idTx := c.id
			var valor float64
			fmt.Sscanf(c.valor, "%f", &valor)
			if _, err := BaixarTitulo(tx, g[i].id, &idTx, c.data, valor, "automatica", "conciliado automaticamente pelo CNPJ do pagador e valor"); err != nil {
				tx.Rollback()
				log.Printf("[Titulos] baixa automática do título %s: %v", g[i].id, err)
				continue
			}
			if err := tx.Commit(); err != nil {
				return baixados, err
			}
			baixados++
			log.Printf("[Titulos] título %s baixado automaticamente (lançamento %s)", g[i].id, c.id)
		}
	}
	return baixados, nil
}

type creditoCandidato struct {
	id, valor, descricao, pagador, cnpjPagador string
	data                                       time.Time
}

// raizesDoCliente lista as raízes de CNPJ do cliente (cadastro principal e CNPJs vinculados).
func raizesDoCliente(ctx context.Context, db *sql.DB, clienteID string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT cnpj FROM financeiro.clientes WHERE id = $1 UNION SELECT cnpj FROM financeiro.cliente_cnpjs WHERE cliente_id = $1`, clienteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	vistos := map[string]bool{}
	for rows.Next() {
		var c string
		if rows.Scan(&c) == nil {
			if r := raizCNPJ(c); r != "" && !vistos[r] {
				vistos[r] = true
				out = append(out, r)
			}
		}
	}
	return out, rows.Err()
}

// candidatosCredito: créditos livres de mesmo valor, na janela, com CNPJ de pagador de uma das raízes.
func candidatosCredito(ctx context.Context, db *sql.DB, valor string, de, aPartirCriacao time.Time, raizes []string) ([]creditoCandidato, error) {
	inicio := de
	if aPartirCriacao.After(inicio) {
		inicio = aPartirCriacao
	}
	rows, err := db.QueryContext(ctx, `
		SELECT t.id, t.valor::text, t.data_transacao, COALESCE(t.descricao,''),
		       COALESCE(t.detalhes->'detalhes'->>'nomePagador',''), COALESCE(t.detalhes->'detalhes'->>'cpfCnpjPagador','')
		FROM financeiro.transacoes t
		WHERE t.tipo = 'credito' AND t.valor = $1::numeric
		  AND t.data_transacao >= $2 AND t.data_transacao <= CURRENT_DATE
		  AND NOT EXISTS (SELECT 1 FROM financeiro.titulos_receber x WHERE x.transacao_id = t.id)
		  AND LEFT(regexp_replace(COALESCE(t.detalhes->'detalhes'->>'cpfCnpjPagador',''), '\D', '', 'g'), 8) = ANY($3::text[])
		ORDER BY t.data_transacao, t.created_at`, valor, inicio.Format("2006-01-02"), pqArray(raizes))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []creditoCandidato
	for rows.Next() {
		var c creditoCandidato
		if err := rows.Scan(&c.id, &c.valor, &c.data, &c.descricao, &c.pagador, &c.cnpjPagador); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// pqArray formata uma lista de texto como array literal do Postgres ({a,b}); as raízes são só dígitos.
func pqArray(v []string) string { return "{" + strings.Join(v, ",") + "}" }
