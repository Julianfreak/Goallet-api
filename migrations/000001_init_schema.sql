-- Habilitar extensión para UUIDs si se requiere (opcional, usaremos VARCHAR/UUID)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tabla de Cuentas Billetera
CREATE TABLE IF NOT EXISTS cuentas (
    id VARCHAR(36) PRIMARY KEY,
    titular VARCHAR(150) NOT NULL,
    saldo NUMERIC(15, 2) NOT NULL DEFAULT 0.00 CHECK (saldo >= 0.00),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Tabla de Transacciones Financieras
CREATE TABLE IF NOT EXISTS transacciones (
    id VARCHAR(36) PRIMARY KEY,
    cuenta_id VARCHAR(36) NOT NULL,
    tipo VARCHAR(30) NOT NULL, -- 'DEPOSITO', 'RETIRO', 'TRANSFERENCIA_ENVIADA', 'TRANSFERENCIA_RECIBIDA'
    monto NUMERIC(15, 2) NOT NULL CHECK (monto > 0.00),
    saldo_resultante NUMERIC(15, 2) NOT NULL,
    cuenta_destino_id VARCHAR(36) NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_cuenta FOREIGN KEY (cuenta_id) REFERENCES cuentas(id) ON DELETE CASCADE
);

-- Índices para optimizar consultas por cuenta
CREATE INDEX IF NOT EXISTS idx_transacciones_cuenta_id ON transacciones(cuenta_id);