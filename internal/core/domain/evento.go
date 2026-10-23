package domain

import "time"

type TransferenciaEvent struct {
	ID              string    `json:"id"`
	CuentaOrigenID  string    `json:"cuenta_origen_id"`
	CuentaDestinoID string    `json:"cuenta_destino_id"`
	Monto           float64   `json:"monto"`
	Timestamp       time.Time `json:"timestamp"`
}
