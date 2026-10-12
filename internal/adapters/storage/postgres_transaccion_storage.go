package storage

import (
	"context"
	"database/sql"
	"fmt"
	"goallet-api/internal/core/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresTransaccionStorage struct {
	db *pgxpool.Pool
}

func NewPostgresTransaccionStorage(db *pgxpool.Pool) *PostgresTransaccionStorage {
	return &PostgresTransaccionStorage{db: db}
}

// Guardar registra una nueva transacción usando context.Background() internamente
func (s *PostgresTransaccionStorage) Guardar(tx *domain.Transaccion) error {
	query := `
		INSERT INTO transacciones (id, cuenta_id, tipo, monto, saldo_resultante, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := s.db.Exec(
		context.Background(),
		query,
		tx.ID,
		tx.CuentaOrigenID,
		tx.Tipo,
		tx.Monto,
		0.00, // Ajusta si manejas saldo resultante en tu dominio
		tx.Fecha,
	)
	if err != nil {
		return fmt.Errorf("error al guardar transacción en postgres: %w", err)
	}
	return nil
}

// ListarPorCuentaID obtiene todas las transacciones asociadas a una cuenta específica
func (s *PostgresTransaccionStorage) ListarPorCuentaID(cuentaID string) ([]domain.Transaccion, error) {
	query := `
		SELECT id, cuenta_id, tipo, monto, cuenta_destino_id, created_at 
		FROM transacciones 
		WHERE cuenta_id = $1 
		ORDER BY created_at DESC
	`
	rows, err := s.db.Query(context.Background(), query, cuentaID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar transacciones en postgres: %w", err)
	}
	defer rows.Close()

	var transacciones []domain.Transaccion
	for rows.Next() {
		var t domain.Transaccion
		var cuentaDestino sql.NullString // Maneja el campo opcional/nulo de destino

		if err := rows.Scan(&t.ID, &t.CuentaOrigenID, &t.Tipo, &t.Monto, &cuentaDestino, &t.Fecha); err != nil {
			return nil, fmt.Errorf("error al escanear transacción: %w", err)
		}

		if cuentaDestino.Valid {
			t.CuentaDestinoID = cuentaDestino.String
		}

		transacciones = append(transacciones, t)
	}

	return transacciones, nil
}
