-- Tabla de Cuentas Billetera
CREATE TABLE IF NOT EXISTS cuentas (
    id VARCHAR(255) PRIMARY KEY,
    titular VARCHAR(255) NOT NULL,
    saldo NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Transacciones Financieras
CREATE TABLE IF NOT EXISTS transacciones (
    id VARCHAR(255) PRIMARY KEY,
    cuenta_id VARCHAR(255) NOT NULL,
    tipo VARCHAR(30) NOT NULL, -- 'DEPOSITO', 'RETIRO', 'TRANSFERENCIA_ENVIADA', 'TRANSFERENCIA_RECIBIDA'
    monto NUMERIC(15, 2) NOT NULL CHECK (monto > 0.00),
    saldo_resultante NUMERIC(15, 2) NOT NULL,
    cuenta_destino_id VARCHAR(255) NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_cuenta FOREIGN KEY (cuenta_id) REFERENCES cuentas(id) ON DELETE CASCADE
);

-- Tabla de Movimientos (Ledger)
CREATE TABLE IF NOT EXISTS movimientos (
    id VARCHAR(255) PRIMARY KEY,
    cuenta_id VARCHAR(255) NOT NULL,
    tipo VARCHAR(50) NOT NULL, -- DEPOSITO, RETIRO, TRANSFERENCIA
    monto NUMERIC(15, 2) NOT NULL,
    saldo_anterior NUMERIC(15, 2) NOT NULL,
    saldo_nuevo NUMERIC(15, 2) NOT NULL,
    descripcion TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT movimientos_cuenta_id_fkey FOREIGN KEY (cuenta_id) REFERENCES cuentas(id) ON DELETE CASCADE
);

-- Tabla de Idempotencia
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(255) PRIMARY KEY,
    response_code INT NOT NULL,
    response_body TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Índices para optimizar consultas por cuenta
CREATE INDEX IF NOT EXISTS idx_transacciones_cuenta_id ON transacciones(cuenta_id);
CREATE INDEX IF NOT EXISTS idx_movimientos_cuenta_id ON movimientos(cuenta_id);