package domain

import "time"

type TipoMovimiento string

const (
	TipoDeposito              TipoMovimiento = "DEPOSITO"
	TipoRetiro                TipoMovimiento = "RETIRO"
	TipoTransferenciaEnviada  TipoMovimiento = "TRANSFERENCIA_ENVIADA"
	TipoTransferenciaRecibida TipoMovimiento = "TRANSFERENCIA_RECIBIDA"
)

type Movimiento struct {
	ID            string         `json:"id"`
	CuentaID      string         `json:"cuenta_id"`
	Tipo          TipoMovimiento `json:"tipo"`
	Monto         float64        `json:"monto"`
	SaldoAnterior float64        `json:"saldo_anterior"`
	SaldoNuevo    float64        `json:"saldo_nuevo"`
	CreatedAt     time.Time      `json:"created_at"`
}
