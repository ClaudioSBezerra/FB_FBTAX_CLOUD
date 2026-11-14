package services

// Contas a Receber a partir da NFS-e: ao autorizar a nota, nasce o título correspondente; ao
// cancelar a nota, o título em aberto é cancelado junto.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DiaVencimentoPadrao: as notas são emitidas no dia 01 e vencem no dia 10 do mesmo mês.
const DiaVencimentoPadrao = 10

// VencimentoPadrao devolve o dia 10 do mês da emissão; se a emissão já passou do dia 10, o dia 10
// do mês seguinte (nunca nasce um título vencido por padrão).
func VencimentoPadrao(emissao time.Time) time.Time {
	y, m, d := emissao.Date()
	if d > DiaVencimentoPadrao {
		m++
	}
	return time.Date(y, m, DiaVencimentoPadrao, 0, 0, 0, 0, emissao.Location())
}

var tiposTitulo = map[string]bool{"mensalidade": true, "adesao": true, "projeto": true, "outro": true}

// GerarTituloDaNFSe cria o título da nota autorizada. É idempotente: se a nota já tem título vivo,
// devolve esse; se o título da mensalidade do contrato/competência já havia sido cadastrado à mão,
// apenas o liga à nota (criado=false) em vez de duplicar.
func GerarTituloDaNFSe(ctx context.Context, db *sql.DB, nfseID, tipo string, vencimento time.Time) (tituloID string, criado bool, err error) {
	if !tiposTitulo[tipo] {
		return "", false, fmt.Errorf("tipo de título inválido: %q", tipo)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()

	var cliente, contrato sql.NullString
	var competencia time.Time
	var valor, descricao, status string
	err = tx.QueryRowContext(ctx, `SELECT cliente_id, contrato_id, competencia, valor::text, descricao, status
		FROM financeiro.nfse_emitidas WHERE id = $1 FOR UPDATE`, nfseID).
		Scan(&cliente, &contrato, &competencia, &valor, &descricao, &status)
	if err == sql.ErrNoRows {
		return "", false, errors.New("nota não encontrada")
	}
	if err != nil {
		return "", false, err
	}
	if status != "autorizada" {
		return "", false, fmt.Errorf("a nota está %s: só nota autorizada gera título", status)
	}
	if !cliente.Valid {
		return "", false, errors.New("nota sem cliente")
	}

	err = tx.QueryRowContext(ctx, `SELECT id FROM financeiro.titulos_receber WHERE nfse_id = $1 AND status <> 'cancelado'`, nfseID).Scan(&tituloID)
	if err == nil {
		return tituloID, false, nil
	}
	if err != sql.ErrNoRows {
		return "", false, err
	}

	var comp any
	if tipo == "mensalidade" {
		comp = competencia.Format("2006-01-02")
	}
	if tipo == "mensalidade" && contrato.Valid {
		err = tx.QueryRowContext(ctx, `
			UPDATE financeiro.titulos_receber SET nfse_id = $1, updated_at = NOW()
			WHERE id = (SELECT id FROM financeiro.titulos_receber
			            WHERE contrato_id = $2 AND competencia = $3 AND tipo = 'mensalidade'
			              AND status <> 'cancelado' AND nfse_id IS NULL LIMIT 1)
			RETURNING id`, nfseID, contrato.String, competencia.Format("2006-01-02")).Scan(&tituloID)
		if err == nil {
			return tituloID, false, tx.Commit()
		}
		if err != sql.ErrNoRows {
			return "", false, err
		}
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO financeiro.titulos_receber (cliente_id, contrato_id, tipo, competencia, descricao, valor, vencimento, nfse_id)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7, $8) RETURNING id`,
		cliente.String, nullIfEmpty(contrato.String), tipo, comp, strings.Join(strings.Fields(descricao), " "), valor,
		vencimento.Format("2006-01-02"), nfseID).Scan(&tituloID)
	if err != nil {
		return "", false, err
	}
	return tituloID, true, tx.Commit()
}

// CancelarTituloDaNFSe cancela o título em aberto da nota cancelada. Título já pago não é tocado:
// devolve pago=true para a tela avisar que o estorno é decisão do operador.
func CancelarTituloDaNFSe(ctx context.Context, db *sql.DB, nfseID string) (cancelados int, pago bool, err error) {
	res, err := db.ExecContext(ctx, `UPDATE financeiro.titulos_receber
		SET status = 'cancelado', baixa_obs = 'NFS-e cancelada', updated_at = NOW()
		WHERE nfse_id = $1 AND status = 'aberto'`, nfseID)
	if err != nil {
		return 0, false, err
	}
	n, _ := res.RowsAffected()
	err = db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM financeiro.titulos_receber WHERE nfse_id = $1 AND status = 'pago')`, nfseID).Scan(&pago)
	return int(n), pago, err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
