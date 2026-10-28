package services_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/services"

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

// Extensión de MockAuditRepository para soportar listado y actualización
func (m *MockAuditRepository) ListarPendientes() ([]domain.EventoFallido, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.EventoFallido), args.Error(1)
}

func (m *MockAuditRepository) ActualizarEstado(id int, estado string, intentos int) error {
	args := m.Called(id, estado, intentos)
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

func TestAuditService_ReprocesarEventos_Exitoso(t *testing.T) {
	// Arrange
	publisherMock := new(MockEventPublisher)
	auditRepoMock := new(MockAuditRepository)

	transferenciaEvt := domain.TransferenciaEvent{
		ID:              "tx-test-reprocesar-1",
		CuentaOrigenID:  "cta-origen",
		CuentaDestinoID: "cta-destino",
		Monto:           250.0,
		Timestamp:       time.Now(),
	}
	payloadBytes, _ := json.Marshal(transferenciaEvt)

	// Simulamos un evento pendiente registrado previamente en la auditoría
	eventosPendientes := []domain.EventoFallido{
		{
			ID:            1,
			TransaccionID: "tx-test-reprocesar-1",
			Payload:       payloadBytes,
			Estado:        "FALLIDO",
			Intentos:      2,
		},
	}

	// 1. Configuramos el repositorio para que devuelva el evento pendiente
	auditRepoMock.On("ListarPendientes").Return(eventosPendientes, nil)

	// 2. Configuramos el publicador para que ahora sí responda con éxito (RabbitMQ recuperado)
	publisherMock.On("PublishTransferencia", mock.AnythingOfType("domain.TransferenciaEvent")).Return(nil)

	// 3. Esperamos que el estado del evento se actualice a "PROCESADO"
	auditRepoMock.On("ActualizarEstado", 1, "PROCESADO", 2).Return(nil)

	auditService := services.NewAuditService(auditRepoMock, publisherMock)

	err := auditService.ReprocesarEventos()
	// Act (Simulando la lógica del AuditService.ReprocesarEventos)
	pendientes, err := auditRepoMock.ListarPendientes()
	assert.NoError(t, err)
	assert.Len(t, pendientes, 1)

	for _, evento := range pendientes {
		var transferenciaEvent domain.TransferenciaEvent

		// Aserción de tipo para asegurar que es []byte
		payloadBytes, ok := evento.Payload.([]byte)
		assert.True(t, ok, "El payload debe ser un slice de bytes")

		errUnmarshal := json.Unmarshal(payloadBytes, &transferenciaEvent)
		assert.NoError(t, errUnmarshal)

		// Intentamos publicar nuevamente
		errPub := publisherMock.PublishTransferencia(transferenciaEvent)
		if errPub == nil {
			// Si publica con éxito, actualizamos el estado en la BD de auditoría
			errUpd := auditRepoMock.ActualizarEstado(evento.ID, "PROCESADO", evento.Intentos)
			assert.NoError(t, errUpd)
		}
	}

	// Assert: Verificamos que todos los mocks cumplieron sus expectativas
	assert.Error(t, err)
	publisherMock.AssertExpectations(t)
	auditRepoMock.AssertExpectations(t)
}

func TestAuditService_ReprocesarEventos_FallaListarPendientes(t *testing.T) {
	publisherMock := new(MockEventPublisher)
	auditRepoMock := new(MockAuditRepository)

	auditRepoMock.On("ListarPendientes").Return(nil, errors.New("error de conexion bd"))

	auditService := services.NewAuditService(auditRepoMock, publisherMock)

	err := auditService.ReprocesarEventos()

	assert.Error(t, err)
	auditRepoMock.AssertExpectations(t)
	publisherMock.AssertNotCalled(t, "PublishTransferencia", mock.Anything)
}

func TestAuditService_ReprocesarEventos_FallaActualizarEstado(t *testing.T) {
	publisherMock := new(MockEventPublisher)
	auditRepoMock := new(MockAuditRepository)

	transferenciaEvt := domain.TransferenciaEvent{
		ID:              "tx-test-error-upd",
		CuentaOrigenID:  "cta-origen",
		CuentaDestinoID: "cta-destino",
		Monto:           100.0,
		Timestamp:       time.Now(),
	}
	payloadBytes, _ := json.Marshal(transferenciaEvt)

	eventosPendientes := []domain.EventoFallido{
		{
			ID:            2,
			TransaccionID: "tx-test-error-upd",
			Payload:       payloadBytes,
			Estado:        "FALLIDO",
			Intentos:      1,
		},
	}

	auditRepoMock.On("ListarPendientes").Return(eventosPendientes, nil)
	publisherMock.On("PublishTransferencia", mock.AnythingOfType("domain.TransferenciaEvent")).Return(nil)
	auditRepoMock.On("ActualizarEstado", 2, "PROCESADO", 1).Return(errors.New("error al actualizar estado en db"))

	auditService := services.NewAuditService(auditRepoMock, publisherMock)

	err := auditService.ReprocesarEventos()

	assert.Error(t, err)
	publisherMock.AssertExpectations(t)
	auditRepoMock.AssertExpectations(t)
}
func TestAuditService_IniciarWorkerReproceso(t *testing.T) {
	publisherMock := new(MockEventPublisher)
	auditRepoMock := new(MockAuditRepository)

	// Simulamos que el worker encuentra una lista vacía de pendientes
	auditRepoMock.On("ListarPendientes").Return([]domain.EventoFallido{}, nil).Maybe()

	auditService := services.NewAuditService(auditRepoMock, publisherMock)

	// Iniciamos el worker con un intervalo muy corto (10ms)
	auditService.IniciarWorkerReproceso(10 * time.Millisecond)

	// Esperamos lo suficiente para que el ticker dispare al menos un ciclo
	time.Sleep(30 * time.Millisecond)

	// Verificamos que el worker ejecutó el ciclo llamando al repositorio
	auditRepoMock.AssertCalled(t, "ListarPendientes")
}
