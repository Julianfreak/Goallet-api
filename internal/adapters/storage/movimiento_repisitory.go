package storage

import (
	"context"
	"fmt"
	"goallet-api/internal/core/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MovimientoRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewMovimientoRepositoryPostgres(db *pgxpool.Pool) *MovimientoRepositoryPostgres {
	return &MovimientoRepositoryPostgres{db: db}
}

func (r *MovimientoRepositoryPostgres) Registrar(ctx context.Context, tx interface{}, mov *domain.Movimiento) error {
	query := `
		INSERT INTO movimientos (cuenta_id, tipo, monto, saldo_anterior, saldo_nuevo)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	var err error
	var row pgx.Row

	// Permite ejecutar la inserción dentro de una transacción existente o usando el pool directamente
	if sqlTx, ok := tx.(pgx.Tx); ok {
		row = sqlTx.QueryRow(ctx, query, mov.CuentaID, mov.Tipo, mov.Monto, mov.SaldoAnterior, mov.SaldoNuevo)
	} else {
		row = r.db.QueryRow(ctx, query, mov.CuentaID, mov.Tipo, mov.Monto, mov.SaldoAnterior, mov.SaldoNuevo)
	}

	err = row.Scan(&mov.ID, &mov.CreatedAt)
	if err != nil {
		return fmt.Errorf("error al registrar movimiento en ledger: %w", err)
	}

	return nil
}

func (r *MovimientoRepositoryPostgres) ObtenerPorCuentaID(ctx context.Context, cuentaID string) ([]domain.Movimiento, error) {
	query := `
		SELECT id, cuenta_id, tipo, monto, saldo_anterior, saldo_nuevo, created_at
		FROM movimientos
		WHERE cuenta_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, cuentaID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar movimientos: %w", err)
	}
	defer rows.Close()

	var movimientos []domain.Movimiento
	for rows.Next() {
		var m domain.Movimiento
		if err := rows.Scan(&m.ID, &m.CuentaID, &m.Tipo, &m.Monto, &m.SaldoAnterior, &m.SaldoNuevo, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("error al escanear movimiento: %w", err)
		}
		movimientos = append(movimientos, m)
	}

	return movimientos, nil
}
