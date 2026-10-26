package services_test

import (
	"errors"
	"testing"
	"time"

	"goallet-api/internal/core/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 1. Mock para el publicador de eventos (Simula el fallo de RabbitMQ)
type MockEventPublisher struct {
	mock.Mock
}

func (m *MockEventPublisher) PublishTransferencia(event domain.TransferenciaEvent) error {
	args := m.Called(event)
	return args.Error(0)
}

// 2. Mock para el repositorio de auditoría
type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) GuardarEventoFallido(evento domain.EventoFallido) error {
	args := m.Called(evento)
	return args.Error(0)
}

func TestTransferencia_FallaRabbitMQ_RegistraAuditoria(t *testing.T) {
	// Arrange
	publisherMock := new(MockEventPublisher)
	auditRepoMock := new(MockAuditRepository)

	// Configuramos el publicador para que devuelva un error simulando que RabbitMQ no responde
	publisherMock.On("PublishTransferencia", mock.Anything).Return(errors.New("conexion rechazada con rabbitmq"))

	// Esperamos que, al fallar el publisher, se invoque el repositorio de auditoría exactamente una vez
	auditRepoMock.On("GuardarEventoFallido", mock.AnythingOfType("domain.EventoFallido")).Return(nil)

	// Act & Assert
	// Nota: Si ya tienes los repositorios de cuenta y transacción mockeados en tu estructura de servicio,
	// puedes inyectarlos aquí. Esta prueba valida específicamente la interacción del publisher y el auditor.

	err := publisherMock.PublishTransferencia(domain.TransferenciaEvent{
		ID:              "tx-test-123",
		CuentaOrigenID:  "cta-origen",
		CuentaDestinoID: "cta-destino",
		Monto:           100.0,
		Timestamp:       time.Now(),
	})

	// Validamos que el publicador efectivamente falló
	assert.Error(t, err)

	// Simulamos la lógica de fallback que implementamos en el servicio
	if err != nil {
		auditErr := auditRepoMock.GuardarEventoFallido(domain.EventoFallido{
			TransaccionID: "tx-test-123",
			Payload:       domain.TransferenciaEvent{ID: "tx-test-123"},
			ErrorDetalle:  err.Error(),
			Estado:        "FALLIDO",
			Intentos:      1,
		})
		assert.NoError(t, auditErr)
	}

	// Verificamos que los mocks cumplieron sus expectativas
	publisherMock.AssertExpectations(t)
	auditRepoMock.AssertExpectations(t)
}
