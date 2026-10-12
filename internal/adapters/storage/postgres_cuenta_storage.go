package storage

import (
	"context"
	"errors"
	"fmt"
	"goallet-api/internal/core/domain"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresCuentaStorage struct {
	db *pgxpool.Pool
}

func NewPostgresCuentaStorage(db *pgxpool.Pool) *PostgresCuentaStorage {
	return &PostgresCuentaStorage{db: db}
}

// CrearCuenta guarda una nueva cuenta en PostgreSQL
func (s *PostgresCuentaStorage) CrearCuenta(ctx context.Context, cuenta *domain.Cuenta) error {
	query := `
		INSERT INTO cuentas (id, titular, saldo, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	// Usamos created_at tanto para created como para updated inicial
	_, err := s.db.Exec(ctx, query, cuenta.ID, cuenta.Titular, cuenta.Saldo, cuenta.CreatedAt, cuenta.CreatedAt)
	if err != nil {
		return fmt.Errorf("error al insertar cuenta en postgres: %w", err)
	}
	return nil
}

// ConsultarCuenta busca una cuenta por su ID
func (s *PostgresCuentaStorage) ConsultarCuenta(ctx context.Context, id string) (*domain.Cuenta, error) {
	query := `SELECT id, titular, saldo, created_at FROM cuentas WHERE id = $1`

	var c domain.Cuenta
	err := s.db.QueryRow(ctx, query, id).Scan(&c.ID, &c.Titular, &c.Saldo, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCuentaNoEncontrada
		}
		return nil, fmt.Errorf("error al consultar cuenta: %w", err)
	}
	return &c, nil
}

// RetirarConBloqueo realiza un retiro asegurando bloqueo exclusivo de la fila (SELECT ... FOR UPDATE)
// Esto evita condiciones de carrera si dos hilos intentan retirar simultáneamente de la misma cuenta.
func (s *PostgresCuentaStorage) RetirarConBloqueo(ctx context.Context, cuentaID string, monto float64) (*domain.Cuenta, *domain.Transaccion, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("error al iniciar transacción: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Bloquear la fila de la cuenta exclusivamente para esta transacción
	selectQuery := `SELECT id, titular, saldo, created_at FROM cuentas WHERE id = $1 FOR UPDATE`
	var c domain.Cuenta
	err = tx.QueryRow(ctx, selectQuery, cuentaID).Scan(&c.ID, &c.Titular, &c.Saldo, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, domain.ErrCuentaNoEncontrada
		}
		return nil, nil, fmt.Errorf("error al bloquear cuenta: %w", err)
	}

	// 2. Validar saldo suficiente utilizando el método de dominio
	if err := c.PuedeRetirar(monto); err != nil {
		return nil, nil, err
	}

	// 3. Modificar saldo usando el método de dominio
	if err := c.Retirar(monto); err != nil {
		return nil, nil, err
	}

	now := time.Now()

	// 4. Actualizar saldo en la base de datos
	updateQuery := `UPDATE cuentas SET saldo = $1, updated_at = $2 WHERE id = $3`
	_, err = tx.Exec(ctx, updateQuery, c.Saldo, now, cuentaID)
	if err != nil {
		return nil, nil, fmt.Errorf("error al actualizar saldo: %w", err)
	}

	// 5. Registrar la transacción financiera usando la estructura real de dominio
	transaccionID := fmt.Sprintf("tx-%d", now.UnixNano())
	insertTxQuery := `
		INSERT INTO transacciones (id, cuenta_id, tipo, monto, saldo_resultante, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, insertTxQuery, transaccionID, cuentaID, domain.Retiro, monto, c.Saldo, now)
	if err != nil {
		return nil, nil, fmt.Errorf("error al registrar transacción: %w", err)
	}

	// 6. Confirmar transacción (COMMIT)
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("error al confirmar transacción: %w", err)
	}

	transaccion := &domain.Transaccion{
		ID:             transaccionID,
		CuentaOrigenID: cuentaID,
		Tipo:           domain.Retiro,
		Monto:          monto,
		Fecha:          now,
	}

	return &c, transaccion, nil
}

// Actualizar modifica los datos de una cuenta existente en la base de datos
func (s *PostgresCuentaStorage) Actualizar(cuenta *domain.Cuenta) error {
	query := `
		UPDATE cuentas 
		SET titular = $1, saldo = $2, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $3
	`
	_, err := s.db.Exec(context.Background(), query, cuenta.Titular, cuenta.Saldo, cuenta.ID)
	if err != nil {
		return fmt.Errorf("error al actualizar cuenta en postgres: %w", err)
	}
	return nil
}

// BuscarPorID busca una cuenta por su ID cumpliendo con la interfaz ports.CuentaRepository
func (s *PostgresCuentaStorage) BuscarPorID(id string) (*domain.Cuenta, error) {
	query := `SELECT id, titular, saldo, created_at FROM cuentas WHERE id = $1`

	var c domain.Cuenta
	err := s.db.QueryRow(context.Background(), query, id).Scan(&c.ID, &c.Titular, &c.Saldo, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCuentaNoEncontrada
		}
		return nil, fmt.Errorf("error al buscar cuenta por id: %w", err)
	}
	return &c, nil
}

// Eliminar borra una cuenta de la base de datos por su ID
func (s *PostgresCuentaStorage) Eliminar(id string) error {
	query := `DELETE FROM cuentas WHERE id = $1`

	result, err := s.db.Exec(context.Background(), query, id)
	if err != nil {
		return fmt.Errorf("error al eliminar cuenta en postgres: %w", err)
	}

	// Opcional: verificar si la cuenta realmente existía
	if result.RowsAffected() == 0 {
		return domain.ErrCuentaNoEncontrada
	}

	return nil
}

// Guardar inserta una nueva cuenta en la base de datos para cumplir con ports.CuentaRepository
func (s *PostgresCuentaStorage) Guardar(cuenta *domain.Cuenta) error {
	query := `
		INSERT INTO cuentas (id, titular, saldo, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := s.db.Exec(context.Background(), query, cuenta.ID, cuenta.Titular, cuenta.Saldo, cuenta.CreatedAt, cuenta.CreatedAt)
	if err != nil {
		return fmt.Errorf("error al guardar cuenta en postgres: %w", err)
	}
	return nil
}

// Listar obtiene todas las cuentas almacenadas en la base de datos
func (s *PostgresCuentaStorage) Listar() ([]*domain.Cuenta, error) {
	query := `SELECT id, titular, saldo, created_at FROM cuentas ORDER BY created_at DESC`

	rows, err := s.db.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("error al listar cuentas en postgres: %w", err)
	}
	defer rows.Close()

	var cuentas []*domain.Cuenta
	for rows.Next() {
		var c domain.Cuenta
		if err := rows.Scan(&c.ID, &c.Titular, &c.Saldo, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("error al escanear cuenta: %w", err)
		}
		cuentas = append(cuentas, &c)
	}

	return cuentas, nil
}
