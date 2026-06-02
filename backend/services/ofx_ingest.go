package services

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log"
	"math"
	"strings"

	"github.com/aclindsa/ofxgo"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// OFXDetectedAccount contém os dados identificados no OFX quando a conta não foi encontrada automaticamente.
type OFXDetectedAccount struct {
	BankID   string `json:"bankid"`
	AcctID   string `json:"acctid"`
	BranchID string `json:"branchid"`
}

// OFXErroDetalhe descreve um erro de parse ou insert em uma linha do OFX.
type OFXErroDetalhe struct {
	FITID  string `json:"fitid"`
	Motivo string `json:"motivo"`
}

// OFXIngestResult resume o resultado da ingestão de um extrato OFX.
type OFXIngestResult struct {
	Importadas    int              `json:"importadas"`
	Duplicadas    int              `json:"duplicadas"`
	Erros         int              `json:"erros"`
	IdsInseridos  []string         `json:"ids_inseridos"`
	DetalhesErros []OFXErroDetalhe `json:"detalhes_erros,omitempty"`
	ContaApelido  string           `json:"conta_apelido,omitempty"`
}

// ErrContaNaoDetectada é retornado quando o BANKID+ACCTID do OFX não mapeia nenhuma conta cadastrada.
type ErrContaNaoDetectada struct {
	Detected OFXDetectedAccount
}

func (e *ErrContaNaoDetectada) Error() string {
	return fmt.Sprintf("conta não detectada automaticamente: bankid=%s acctid=%s", e.Detected.BankID, e.Detected.AcctID)
}

// IngestOFX faz o parse de um extrato OFX, detecta ou usa a conta indicada, e insere as
// transações em financeiro.transacoes com dedup por referencia_ext, conciliado=false e origem='ofx_upload'.
//
// Se contaID for vazio, tenta detectar a conta pelo BANKID+ACCTID do arquivo.
// Se não encontrar, retorna *ErrContaNaoDetectada com o payload detectado.
func IngestOFX(db *sql.DB, contaID string, r io.Reader) (*OFXIngestResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("erro lendo arquivo: %w", err)
	}

	resp, parseErr := ofxgo.ParseResponse(bytes.NewReader(data))
	if parseErr != nil {
		// Tentar recodificar cp1252 → utf-8 (Inter e outros bancos brasileiros)
		decoder := charmap.Windows1252.NewDecoder()
		converted, convErr := io.ReadAll(transform.NewReader(bytes.NewReader(data), decoder))
		if convErr == nil {
			resp, parseErr = ofxgo.ParseResponse(bytes.NewReader(converted))
		}
		if parseErr != nil {
			return nil, fmt.Errorf("parse OFX falhou: %w", parseErr)
		}
		data = converted
	}
	_ = data // usado apenas para re-parse; manter para evitar lint error

	// Extrair os StatementResponse dos bancos
	var stmts []*ofxgo.StatementResponse
	for _, msg := range resp.Bank {
		if stmt, ok := msg.(*ofxgo.StatementResponse); ok {
			stmts = append(stmts, stmt)
		}
	}
	if len(stmts) == 0 {
		return nil, fmt.Errorf("nenhuma conta bancária encontrada no arquivo OFX")
	}

	stmt := stmts[0]

	// Auto-detect de conta se contaID não foi fornecido
	if contaID == "" {
		bankID := string(stmt.BankAcctFrom.BankID)
		acctID := string(stmt.BankAcctFrom.AcctID)
		branchID := string(stmt.BankAcctFrom.BranchID)

		var foundID string
		err := db.QueryRow(`
			SELECT id FROM financeiro.contas_financeiras
			WHERE (banco ILIKE $1 OR provedor_id = $1) AND conta = $2
			  AND ativa = true
			LIMIT 1`, bankID, acctID).Scan(&foundID)
		if err == sql.ErrNoRows {
			return nil, &ErrContaNaoDetectada{
				Detected: OFXDetectedAccount{
					BankID:   bankID,
					AcctID:   acctID,
					BranchID: branchID,
				},
			}
		}
		if err != nil {
			return nil, fmt.Errorf("erro consultando conta: %w", err)
		}
		contaID = foundID
	}

	// Verificar se a conta existe e obter o apelido
	var apelido string
	err = db.QueryRow(`SELECT apelido FROM financeiro.contas_financeiras WHERE id = $1 AND ativa = true`, contaID).Scan(&apelido)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conta não encontrada: %s", contaID)
	}
	if err != nil {
		return nil, fmt.Errorf("erro verificando conta: %w", err)
	}

	result := &OFXIngestResult{
		IdsInseridos:  []string{},
		DetalhesErros: []OFXErroDetalhe{},
		ContaApelido:  apelido,
	}

	if stmt.BankTranList == nil {
		// Nenhuma transação no arquivo — retorna resultado vazio sem erro
		return result, nil
	}

	for _, tx := range stmt.BankTranList.Transactions {
		fitid := strings.TrimSpace(string(tx.FiTID))

		// Normalizar descrição: preferir MEMO, fallback para NAME
		descricao := strings.TrimSpace(string(tx.Memo))
		if descricao == "" {
			descricao = strings.TrimSpace(string(tx.Name))
		}
		if descricao == "" {
			result.Erros++
			result.DetalhesErros = append(result.DetalhesErros, OFXErroDetalhe{
				FITID:  fitid,
				Motivo: "descrição obrigatória (MEMO/NAME) ausente",
			})
			continue
		}

		// Determinar referencia_ext
		var referenciaExt string
		if fitid != "" {
			referenciaExt = fitid
		} else {
			// Hash determinístico como fallback
			valorStr := tx.TrnAmt.FloatString(2)
			dataStr := tx.DtPosted.Time.Format("2006-01-02")
			descNorm := strings.ToLower(strings.TrimSpace(descricao))
			raw := fmt.Sprintf("%s|%s|%s|%s", contaID, dataStr, valorStr, descNorm)
			h := sha256.Sum256([]byte(raw))
			referenciaExt = fmt.Sprintf("%x", h)
		}

		// Dedup: verificar se já existe
		var exists bool
		db.QueryRow(`SELECT EXISTS(SELECT 1 FROM financeiro.transacoes WHERE referencia_ext = $1)`, referenciaExt).Scan(&exists)
		if exists {
			result.Duplicadas++
			continue
		}

		// Determinar tipo
		tipo := "debito"
		if tx.TrnType == ofxgo.TrnTypeCredit {
			tipo = "credito"
		}

		// Valor absoluto
		f, _ := tx.TrnAmt.Float64()
		valor := math.Abs(f)

		// Data da transação
		dataTransacao := tx.DtPosted.Time.Format("2006-01-02")

		// Inserir
		var newID string
		err := db.QueryRow(`
			INSERT INTO financeiro.transacoes
			    (conta_id, data_transacao, descricao, valor, tipo, categoria, referencia_ext, conciliado, origem)
			VALUES ($1, $2, $3, $4, $5, NULL, $6, false, 'ofx_upload')
			RETURNING id`,
			contaID, dataTransacao, descricao, valor, tipo, referenciaExt,
		).Scan(&newID)
		if err != nil {
			log.Printf("[ofx-ingest] erro inserindo tx %s conta %s: %v", referenciaExt, contaID, err)
			result.Erros++
			result.DetalhesErros = append(result.DetalhesErros, OFXErroDetalhe{
				FITID:  fitid,
				Motivo: "erro ao inserir: " + err.Error(),
			})
			continue
		}

		result.Importadas++
		result.IdsInseridos = append(result.IdsInseridos, newID)
	}

	return result, nil
}
