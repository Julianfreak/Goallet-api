package domain

import (
	"errors"
	"time"
)

// Errores de Dominio (Reglas del Negocio)
var (
	ErrSaldoInsuficiente  = errors.New("saldo insuficiente para realizar la operación")
	ErrMontoInvalido      = errors.New("el monto debe ser mayor a cero")
	ErrCuentaNoEncontrada = errors.New("la cuenta especificada no existe")
	ErrMismaCuentaDestino = errors.New("no es posible transferir a la misma cuenta de origen")
)

// TipoTransaccion define si el movimiento es crédito o débito
type TipoTransaccion string

const (
	Deposito      TipoTransaccion = "DEPOSITO"
	Retiro        TipoTransaccion = "RETIRO"
	Transferencia TipoTransaccion = "TRANSFERENCIA"
)

// Cuenta representa la entidad central de una billetera
type Cuenta struct {
	ID        string    `json:"id"`
	Titular   string    `json:"titular"`
	Saldo     float64   `json:"saldo"`
	CreatedAt time.Time `json:"created_at"`
}

// Transaccion representa el registro contable de cada movimiento
type Transaccion struct {
	ID              string          `json:"id"`
	CuentaOrigenID  string          `json:"cuenta_origen_id"`
	CuentaDestinoID string          `json:"cuenta_destino_id,omitempty"`
	Tipo            TipoTransaccion `json:"tipo"`
	Monto           float64         `json:"monto"`
	Fecha           time.Time       `json:"fecha"`
}

// Métodos de Dominio (Operaciones protegidas)

func (c *Cuenta) PuedeRetirar(monto float64) error {
	if monto <= 0 {
		return ErrMontoInvalido
	}
	if c.Saldo < monto {
		return ErrSaldoInsuficiente
	}
	return nil
}
func (c *Cuenta) Depositar(monto float64) error {
	if monto <= 0 {
		return ErrMontoInvalido
	}
	c.Saldo += monto
	return nil
}

func (c *Cuenta) Retirar(monto float64) error {
	if err := c.PuedeRetirar(monto); err != nil {
		return err
	}
	c.Saldo -= monto
	return nil
}
