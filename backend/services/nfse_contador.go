package services

import (
	"context"
	"database/sql"
	"fmt"
)

// nfseContadorInicial é o último número já consumido ao criar o contador da série. A série
// 70000 foi usada à mão no Emissor Web (DPS 1 a 8), então o primeiro número do sistema é 101.
const nfseContadorInicial = 100

// ReservarNDPS devolve o próximo número de DPS da empresa na série. É um único UPSERT
// atômico, fora de qualquer transação do chamador: se a emissão falhar depois, o número
// permanece consumido (a Sefin não aceita DPS repetido).
func ReservarNDPS(ctx context.Context, db *sql.DB, empresaID, serie string) (int64, error) {
	var n int64
	err := db.QueryRowContext(ctx, `
		INSERT INTO financeiro.nfse_dps_contador (empresa_id, serie, ultimo)
		VALUES ($1, $2, $3)
		ON CONFLICT (empresa_id, serie)
		DO UPDATE SET ultimo = financeiro.nfse_dps_contador.ultimo + 1
		RETURNING ultimo`, empresaID, serie, nfseContadorInicial+1).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("reservar nDPS: %w", err)
	}
	return n, nil
}
