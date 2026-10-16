//go:build integration
// +build integration

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgresContainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestContainer inicializa un contenedor efímero de PostgreSQL usando Testcontainers
func setupTestContainer(t *testing.T) (*pgxpool.Pool, context.Context) {
	ctx := context.Background()

	dbName := "goallet_test_db"
	dbUser := "postgres"
	dbPassword := "postgres"

	pgContainer, err := postgresContainer.Run(ctx,
		"postgres:16-alpine",
		postgresContainer.WithDatabase(dbName),
		postgresContainer.WithUsername(dbUser),
		postgresContainer.WithPassword(dbPassword),
		postgresContainer.WithInitScripts(), // Aquí puedes apuntar a tus archivos .sql de migración si los tienes
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	// Registrar limpieza al finalizar la prueba
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Conectar usando pgxpool (el mismo driver que usa tu API en producción)
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Aplicar esquema inicial de prueba (Simulando tu tabla de cuentas)
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS cuentas (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			titular VARCHAR(255) NOT NULL,
			saldo NUMERIC(15,2) NOT NULL DEFAULT 0.00,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)

	return pool, ctx
}

func TestCuentaRepository_Integration(t *testing.T) {
	pool, ctx := setupTestContainer(t)
	defer pool.Close()

	t.Run("Crear y Consultar Cuenta Exitosamente", func(t *testing.T) {
		var cuentaID string
		titular := "Julian Serna"

		// Insertar cuenta de prueba
		err := pool.QueryRow(ctx,
			"INSERT INTO cuentas (titular, saldo) VALUES ($1, $2) RETURNING id",
			titular, 500.00,
		).Scan(&cuentaID)

		require.NoError(t, err)
		assert.NotEmpty(t, cuentaID)

		// Consultar la cuenta insertada
		var retrievedTitular string
		var retrievedSaldo float64
		err = pool.QueryRow(ctx,
			"SELECT titular, saldo FROM cuentas WHERE id = $1",
			cuentaID,
		).Scan(&retrievedTitular, &retrievedSaldo)

		require.NoError(t, err)
		assert.Equal(t, titular, retrievedTitular)
		assert.Equal(t, 500.00, retrievedSaldo)
	})

	t.Run("Validar Bloqueo de Fila SELECT FOR UPDATE bajo Transacción", func(t *testing.T) {
		var cuentaID string
		err := pool.QueryRow(ctx,
			"INSERT INTO cuentas (titular, saldo) VALUES ($1, $2) RETURNING id",
			"Concurrencia Test", 1000.00,
		).Scan(&cuentaID)
		require.NoError(t, err)

		// Iniciar transacción con SELECT FOR UPDATE
		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		var saldoActual float64
		err = tx.QueryRow(ctx, "SELECT saldo FROM cuentas WHERE id = $1 FOR UPDATE", cuentaID).Scan(&saldoActual)
		require.NoError(t, err)
		assert.Equal(t, 1000.00, saldoActual)

		// Actualizar saldo dentro de la transacción
		_, err = tx.Exec(ctx, "UPDATE cuentas SET saldo = saldo - 200 WHERE id = $1", cuentaID)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)
	})
}
