package handlers

import (
	"errors"
	"net/http"

	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

// ==========================================
// DTOs (Data Transfer Objects para HTTP)
// ==========================================

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

// ==========================================
// ESTRUCTURA DEL HANDLER
// ==========================================

type BilleteraHandler struct {
	service ports.BilleteraService // Inyección del puerto de entrada
}

func NewBilleteraHandler(service ports.BilleteraService) *BilleteraHandler {
	return &BilleteraHandler{
		service: service,
	}
}

// ==========================================
// ENDPOINTS
// ==========================================

// POST /cuentas
func (h *BilleteraHandler) CrearCuenta(c *gin.Context) {
	var req CrearCuentaRequest

	// Validamos el JSON recibido según las reglas del tag 'binding'
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de datos inválido o titular ausente"})
		return
	}

	// Delegamos la operación al servicio
	cuenta, err := h.service.CrearCuenta(req.Titular, req.SaldoInicial)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cuenta)
}

// GET /cuentas/:id
func (h *BilleteraHandler) ConsultarCuenta(c *gin.Context) {
	// Extraemos el parámetro de la ruta dinámica /cuentas/:id
	id := c.Param("id")

	cuenta, err := h.service.ConsultarCuenta(id)
	if err != nil {
		// Mapeo de errores de dominio a códigos HTTP
		if errors.Is(err, domain.ErrCuentaNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cuenta)
}

func (h *BilleteraHandler) Depositar(c *gin.Context) {
	id := c.Param("id")
	var req OperacionMontoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de datos inválido o monto ausente"})
		return
	}

	transaccion, err := h.service.Depositar(id, req.Monto)
	if err != nil {
		if errors.Is(err, domain.ErrCuentaNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrMontoInvalido) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaccion)

}

func (h *BilleteraHandler) Retirar(c *gin.Context) {
	id := c.Param("id")
	var req OperacionMontoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de datos inválido o monto ausente"})
		return
	}

	transaccion, err := h.service.Retirar(id, req.Monto)
	if err != nil {
		if errors.Is(err, domain.ErrCuentaNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrMontoInvalido) || errors.Is(err, domain.ErrSaldoInsuficiente) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaccion)
}

func (h *BilleteraHandler) Transferir(c *gin.Context) {
	origenID := c.Param("id")
	var req TransferirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de datos inválido o datos ausentes"})
		return
	}

	transaccion, err := h.service.Transferir(origenID, req.CuentaDestinoID, req.Monto)
	if err != nil {
		if errors.Is(err, domain.ErrCuentaNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrMontoInvalido) || errors.Is(err, domain.ErrSaldoInsuficiente) || errors.Is(err, domain.ErrMismaCuentaDestino) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaccion)
}

// GET /api/v1/cuentas
func (h *BilleteraHandler) ListarCuentas(c *gin.Context) {
	cuentas, err := h.service.ListarCuentas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cuentas)
}

// PUT /api/v1/cuentas/:id
func (h *BilleteraHandler) ActualizarTitular(c *gin.Context) {
	id := c.Param("id")

	var req ActualizarTitularRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el campo nuevo_titular es obligatorio"})
		return
	}

	cuentaActualizada, err := h.service.ActualizarTitular(id, req.NuevoTitular)
	if err != nil {
		if errors.Is(err, domain.ErrCuentaNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cuentaActualizada)
}

// DELETE /api/v1/cuentas/:id
func (h *BilleteraHandler) EliminarCuenta(c *gin.Context) {
	id := c.Param("id")

	err := h.service.EliminarCuenta(id)
	if err != nil {
		if errors.Is(err, domain.ErrCuentaNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		// Si la cuenta tiene saldo > 0, retornamos 400 Bad Request
		if errors.Is(err, domain.ErrCuentaConSaldo) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "cuenta eliminada exitosamente"})
}
