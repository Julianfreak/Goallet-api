package domain

import "time"

type EventoFallido struct {
	ID            int       `json:"id"`
	TransaccionID string    `json:"transaccion_id"`
	Payload       any       `json:"payload"`
	ErrorDetalle  string    `json:"error_detalle"`
	Estado        string    `json:"estado"`
	Intentos      int       `json:"intentos"`
	CreatedAt     time.Time `json:"created_at"`
}
