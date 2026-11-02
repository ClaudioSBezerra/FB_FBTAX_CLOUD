package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// nfseUltimoInicialSerie70000 é o último número já consumido da série 70000 no Emissor Web
// (DPS 1 a 8 usados ou rejeitados); o primeiro número do sistema nessa série é o 101.
// Qualquer outra série começa do 1.
const nfseUltimoInicialSerie70000 = 100

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
	if serie == NFSeSeriePadrao {
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
