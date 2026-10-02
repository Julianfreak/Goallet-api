package handlers

import (
	"errors"
	"net/http"

	"goallet-api/internal/adapters/handlers/dto"
	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

type BilleteraHandler struct {
	service ports.BilleteraService
}

func NewBilleteraHandler(service ports.BilleteraService) *BilleteraHandler {
	return &BilleteraHandler{
		service: service,
	}
}

// CrearCuenta maneja la apertura de una nueva cuenta
// @Summary      Crear nueva cuenta bancaria
// @Description  Registra un titular y asigna un saldo inicial opcional
// @Tags         Cuentas
// @Accept       json
// @Produce      json
// @Param        request body dto.CrearCuentaRequest true "Datos de apertura de cuenta"
// @Success      201  {object}  domain.Cuenta
// @Failure      400  {object}  dto.ErrorResponse "Formato de datos inválido o titular ausente"
// @Failure      500  {object}  dto.ErrorResponse "Error interno del servidor"
// @Router       /cuentas [post]
func (h *BilleteraHandler) CrearCuenta(c *gin.Context) {
	var req dto.CrearCuentaRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "formato de datos inválido o titular ausente"})
		return
	}

	cuenta, err := h.service.CrearCuenta(req.Titular, req.SaldoInicial)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cuenta)
}

// ConsultarCuenta obtiene el detalle de una cuenta por su ID
// @Summary      Consultar cuenta por ID
// @Description  Obtiene la información detallada de una cuenta existente
// @Tags         Cuentas
// @Produce      json
// @Param        id   path      string  true  "ID de la cuenta"
// @Success      200  {object}  domain.Cuenta
// @Failure      404  {object}  dto.ErrorResponse "Cuenta no encontrada"
// @Failure      500  {object}  dto.ErrorResponse "Error interno del servidor"
// @Router       /cuentas/{id} [get]
func (h *BilleteraHandler) ConsultarCuenta(c *gin.Context) {
	id := c.Param("id")

	cuenta, err := h.service.ConsultarCuenta(id)
	if err != nil {
		if errors.Is(err, domain.ErrCuentaNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cuenta)
}

// Depositar suma un monto al saldo de la cuenta
// @Summary      Realizar un depósito
// @Description  Incrementa el saldo de una cuenta con un monto positivo
// @Tags         Transacciones
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "ID de la cuenta"
// @Param        request  body      dto.OperacionMontoRequest true  "Monto a depositar"
// @Success      200      {object}  domain.Transaccion
// @Failure      400      {object}  dto.ErrorResponse "Monto inválido o cuerpo de petición incorrecto"
// @Failure      404      {object}  dto.ErrorResponse "Cuenta no encontrada"
// @Failure      500      {object}  dto.ErrorResponse "Error interno del servidor"
// @Router       /cuentas/{id}/depositar [post]
func (h *BilleteraHandler) Depositar(c *gin.Context) {
	id := c.Param("id")
	var req dto.OperacionMontoRequest
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

// Retirar descuenta un monto del saldo de la cuenta
// @Summary      Realizar un retiro
// @Description  Deduce un monto de una cuenta si dispone de saldo suficiente
// @Tags         Transacciones
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "ID de la cuenta"
// @Param        request  body      dto.OperacionMontoRequest true  "Monto a retirar"
// @Success      200      {object}  domain.Transaccion
// @Failure      400      {object}  dto.ErrorResponse "Monto inválido o saldo insuficiente"
// @Failure      404      {object}  dto.ErrorResponse "Cuenta no encontrada"
// @Failure      500      {object}  dto.ErrorResponse "Error interno del servidor"
// @Router       /cuentas/{id}/retirar [post]
func (h *BilleteraHandler) Retirar(c *gin.Context) {
	id := c.Param("id")
	var req dto.OperacionMontoRequest
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

// Transferir mueve saldo entre dos cuentas
// @Summary      Transferir fondos entre cuentas
// @Description  Realiza una transferencia desde la cuenta origen hacia la cuenta de destino
// @Tags         Transacciones
// @Accept       json
// @Produce      json
// @Param        id       path      string                true  "ID de la cuenta origen"
// @Param        request  body      dto.TransferirRequest true  "Datos de la transferencia"
// @Success      200      {object}  domain.Transaccion
// @Failure      400      {object}  dto.ErrorResponse "Datos inválidos, saldo insuficiente o misma cuenta"
// @Failure      404      {object}  dto.ErrorResponse "Cuenta no encontrada"
// @Failure      500      {object}  dto.ErrorResponse "Error interno del servidor"
// @Router       /cuentas/{id}/transferir [post]
func (h *BilleteraHandler) Transferir(c *gin.Context) {
	origenID := c.Param("id")
	var req dto.TransferirRequest
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

// ListarCuentas obtiene la lista de todas las cuentas
// @Summary      Listar todas las cuentas
// @Description  Obtiene el listado completo de cuentas registradas en el sistema
// @Tags         Cuentas
// @Produce      json
// @Success      200  {array}   domain.Cuenta
// @Failure      500  {object}  dto.ErrorResponse "Error interno del servidor"
// @Router       /cuentas [get]
func (h *BilleteraHandler) ListarCuentas(c *gin.Context) {
	cuentas, err := h.service.ListarCuentas()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cuentas)
}

// ActualizarTitular modifica el nombre del titular de una cuenta
// @Summary      Actualizar el titular de una cuenta
// @Description  Modifica el nombre del titular asociado a una cuenta
// @Tags         Cuentas
// @Accept       json
// @Produce      json
// @Param        id       path      string                       true  "ID de la cuenta"
// @Param        request  body      dto.ActualizarTitularRequest true  "Nuevo nombre del titular"
// @Success      200      {object}  domain.Cuenta
// @Failure      400      {object}  dto.ErrorResponse "Petición inválida"
// @Failure      404      {object}  dto.ErrorResponse "Cuenta no encontrada"
// @Router       /cuentas/{id} [put]
func (h *BilleteraHandler) ActualizarTitular(c *gin.Context) {
	id := c.Param("id")

	var req dto.ActualizarTitularRequest
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

// EliminarCuenta remueve una cuenta si su saldo es cero
// @Summary      Eliminar una cuenta
// @Description  Elimina una cuenta existente únicamente si su saldo es igual a 0
// @Tags         Cuentas
// @Produce      json
// @Param        id   path      string  true  "ID de la cuenta"
// @Success      200  {object}  dto.MensajeResponse "Cuenta eliminada exitosamente"
// @Failure      400  {object}  dto.ErrorResponse "La cuenta tiene saldo pendiente"
// @Failure      404  {object}  dto.ErrorResponse "Cuenta no encontrada"
// @Failure      500  {object}  dto.ErrorResponse "Error interno del servidor"
// @Router       /cuentas/{id} [delete]
func (h *BilleteraHandler) EliminarCuenta(c *gin.Context) {
	id := c.Param("id")

	err := h.service.EliminarCuenta(id)
	if err != nil {
		if errors.Is(err, domain.ErrCuentaNoEncontrada) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrCuentaConSaldo) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "cuenta eliminada exitosamente"})
}
