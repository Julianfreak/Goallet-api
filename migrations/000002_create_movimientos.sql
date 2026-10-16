-- internal/adapters/storage/postgres/migrations/002_create_movimientos.sql

CREATE TABLE IF NOT EXISTS movimientos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cuenta_id UUID NOT NULL REFERENCES cuentas(id) ON DELETE CASCADE,
    tipo VARCHAR(50) NOT NULL, -- DEPOSITO, RETIRO, TRANSFERENCIA_ENVIADA, TRANSFERENCIA_RECIBIDA
    monto NUMERIC(15,2) NOT NULL,
    saldo_anterior NUMERIC(15,2) NOT NULL,
    saldo_nuevo NUMERIC(15,2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índice para optimizar la consulta de extractos por cuenta
CREATE INDEX idx_movimientos_cuenta_id ON movimientos(cuenta_id);