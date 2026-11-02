package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// nfseSerieEmissorWeb é a série que o Emissor Web já usou (DPS 1 a 8). O sistema não a usa
// (a faixa 70000-79999 é do Emissor Web, E0010), mas se alguém a configurar, começa do 101.
// Qualquer outra série, inclusive a padrão 00001, começa do 1.
const (
	nfseSerieEmissorWeb         = "70000"
	nfseUltimoInicialSerie70000 = 100
)

// NormalizarSerieDPS devolve a série com 5 dígitos (mesma regra do GerarDPS).
func NormalizarSerieDPS(serie string) (string, error) {
	s := soDigitos(serie)
	if s == "" {
		s = NFSeSeriePadrao
	}
	if len(s) > 5 {
		return "", errors.New("série excede 5 dígitos")
	}
	return fmt.Sprintf("%05s", s), nil
}

// ReservarNDPS devolve o próximo número de DPS da empresa no ambiente e série. É um único
// UPSERT atômico, fora de qualquer transação do chamador: se a emissão falhar depois, o
// número permanece consumido (a Sefin não aceita DPS repetido). O contador é separado por
// ambiente para que a homologação não gaste números da produção.
func ReservarNDPS(ctx context.Context, db *sql.DB, empresaID string, ambiente int, serie string) (int64, error) {
	serie, err := NormalizarSerieDPS(serie)
	if err != nil {
		return 0, err
	}
	if ambiente != 1 && ambiente != 2 {
		return 0, errors.New("ambiente inválido")
	}
	var inicial int64
	if serie == nfseSerieEmissorWeb {
		inicial = nfseUltimoInicialSerie70000
	}
	var n int64
	err = db.QueryRowContext(ctx, `
		INSERT INTO financeiro.nfse_dps_contador (empresa_id, ambiente, serie, ultimo)
		VALUES ($1, $2, $3, $4 + 1)
		ON CONFLICT (empresa_id, ambiente, serie)
		DO UPDATE SET ultimo = financeiro.nfse_dps_contador.ultimo + 1
		RETURNING ultimo`, empresaID, ambiente, serie, inicial).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("reservar nDPS: %w", err)
	}
	return n, nil
}
