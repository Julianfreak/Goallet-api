package dto

type CrearCuentaRequest struct {
	Titular      string  `json:"titular" binding:"required"`
	SaldoInicial float64 `json:"saldo_inicial"`
}

type OperacionMontoRequest struct {
	Monto float64 `json:"monto" binding:"required"`
}

type TransferirRequest struct {
	CuentaDestinoID string  `json:"cuenta_destino_id" binding:"required"`
	Monto           float64 `json:"monto" binding:"required"`
}

type ActualizarTitularRequest struct {
	NuevoTitular string `json:"nuevo_titular" binding:"required"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"mensaje de error explicativo"`
}

type MensajeResponse struct {
	Mensaje string `json:"mensaje" example:"operación realizada exitosamente"`
}
