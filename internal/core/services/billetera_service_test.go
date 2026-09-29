package services_test

import (
	"errors"
	"testing"

	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/services"
)

// ==========================================================
// 1. MOCKS DE PERSISTENCIA
// ==========================================================

type MockCuentaRepo struct {
	cuentas            map[string]*domain.Cuenta
	forzarErrorGuardar error
	forzarErrorBuscar  error
}

func NewMockCuentaRepo() *MockCuentaRepo {
	return &MockCuentaRepo{
		cuentas: make(map[string]*domain.Cuenta),
	}
}

func (m *MockCuentaRepo) Guardar(cuenta *domain.Cuenta) error {
	if m.forzarErrorGuardar != nil {
		return m.forzarErrorGuardar
	}
	m.cuentas[cuenta.ID] = cuenta
	return nil
}

func (m *MockCuentaRepo) BuscarPorID(id string) (*domain.Cuenta, error) {
	if m.forzarErrorBuscar != nil {
		return nil, m.forzarErrorBuscar
	}
	cuenta, existe := m.cuentas[id]
	if !existe {
		return nil, domain.ErrCuentaNoEncontrada
	}
	return cuenta, nil
}

func (m *MockCuentaRepo) Actualizar(cuenta *domain.Cuenta) error {
	m.cuentas[cuenta.ID] = cuenta
	return nil
}

func (m *MockCuentaRepo) Listar() ([]*domain.Cuenta, error) {
	var lista []*domain.Cuenta
	for _, c := range m.cuentas {
		lista = append(lista, c)
	}
	return lista, nil
}

func (m *MockCuentaRepo) Eliminar(id string) error {
	delete(m.cuentas, id)
	return nil
}

type MockTxRepo struct {
	transacciones []domain.Transaccion
}

func NewMockTxRepo() *MockTxRepo {
	return &MockTxRepo{
		transacciones: make([]domain.Transaccion, 0),
	}
}

func (m *MockTxRepo) Guardar(tx *domain.Transaccion) error {
	m.transacciones = append(m.transacciones, *tx)
	return nil
}

func (m *MockTxRepo) ListarPorCuentaID(cuentaID string) ([]domain.Transaccion, error) {
	return m.transacciones, nil
}

// ==========================================================
// 2. TEST: CrearCuenta
// ==========================================================

func TestCrearCuenta(t *testing.T) {
	tests := []struct {
		nombre        string
		titular       string
		saldoInicial  float64
		errRepo       error
		errorEsperado bool
		errDominio    error
	}{
		{
			nombre:        "Creación exitosa con saldo positivo",
			titular:       "Ana Gomez",
			saldoInicial:  500.0,
			errRepo:       nil,
			errorEsperado: false,
		},
		{
			nombre:        "Creación exitosa con saldo cero",
			titular:       "Carlos Ruiz",
			saldoInicial:  0.0,
			errRepo:       nil,
			errorEsperado: false,
		},
		{
			nombre:        "Fallo por titular vacío",
			titular:       "",
			saldoInicial:  100.0,
			errRepo:       nil,
			errorEsperado: true,
		},
		{
			nombre:        "Fallo por saldo inicial negativo",
			titular:       "Pedro Lopez",
			saldoInicial:  -50.0,
			errRepo:       nil,
			errorEsperado: true,
			errDominio:    domain.ErrMontoInvalido,
		},
		{
			nombre:        "Fallo por error simulado del repositorio",
			titular:       "Sofia Castro",
			saldoInicial:  200.0,
			errRepo:       errors.New("fallo de disco en persistencia"),
			errorEsperado: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.nombre, func(t *testing.T) {
			mockCuenta := NewMockCuentaRepo()
			mockCuenta.forzarErrorGuardar = tt.errRepo // Activamos el error simulado
			mockTx := NewMockTxRepo()
			service := services.NewBilleteraService(mockCuenta, mockTx)

			cuentaCreada, err := service.CrearCuenta(tt.titular, tt.saldoInicial)

			if tt.errorEsperado {
				if err == nil {
					t.Errorf("Se esperaba un error pero se obtuvo nil")
				}
				if tt.errDominio != nil && !errors.Is(err, tt.errDominio) {
					t.Errorf("Esperado error '%v', obtenido '%v'", tt.errDominio, err)
				}
			} else {
				if err != nil {
					t.Fatalf("No se esperaba error, pero se obtuvo: %v", err)
				}
				if cuentaCreada.Titular != tt.titular {
					t.Errorf("Titular esperado '%s', obtenido '%s'", tt.titular, cuentaCreada.Titular)
				}
				if cuentaCreada.Saldo != tt.saldoInicial {
					t.Errorf("Saldo esperado '%.2f', obtenido '%.2f'", tt.saldoInicial, cuentaCreada.Saldo)
				}
			}
		})
	}
}

// ==========================================================
// 3. TEST: Transferir
// ==========================================================

func TestTransferir(t *testing.T) {
	t.Run("Transferencia exitosa", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		// Precargamos las cuentas en el mock
		mockCuenta.cuentas["cta-origen"] = &domain.Cuenta{ID: "cta-origen", Titular: "Origen", Saldo: 500.0}
		mockCuenta.cuentas["cta-destino"] = &domain.Cuenta{ID: "cta-destino", Titular: "Destino", Saldo: 300.0}

		// Transferimos $200
		tx, err := service.Transferir("cta-origen", "cta-destino", 200.0)

		if err != nil {
			t.Fatalf("Error inesperado en transferencia: %v", err)
		}
		if tx.Monto != 200.0 {
			t.Errorf("Monto de transacción esperado 200.0, obtenido %.2f", tx.Monto)
		}

		// Verificamos saldos matemáticos:
		// Origen: 500 - 200 = 300
		// Destino: 300 + 200 = 500
		cuentaOrigen := mockCuenta.cuentas["cta-origen"]
		cuentaDestino := mockCuenta.cuentas["cta-destino"]

		if cuentaOrigen.Saldo != 300.0 {
			t.Errorf("Saldo origen esperado '300.00', obtenido '%.2f'", cuentaOrigen.Saldo)
		}
		if cuentaDestino.Saldo != 500.0 {
			t.Errorf("Saldo destino esperado '500.00', obtenido '%.2f'", cuentaDestino.Saldo)
		}
		if len(mockTx.transacciones) != 1 {
			t.Errorf("Se esperaba 1 registro contable en auditoría, obtenidos %d", len(mockTx.transacciones))
		}
	})

	t.Run("Fallo por transferir a la misma cuenta", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		_, err := service.Transferir("cta-1", "cta-1", 50.0)

		if !errors.Is(err, domain.ErrMismaCuentaDestino) {
			t.Errorf("Esperado error '%v', obtenido '%v'", domain.ErrMismaCuentaDestino, err)
		}
	})

	t.Run("Fallo por saldo insuficiente", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		mockCuenta.cuentas["cta-pobre"] = &domain.Cuenta{ID: "cta-pobre", Titular: "Sin Fondos", Saldo: 50.0}
		mockCuenta.cuentas["cta-destino"] = &domain.Cuenta{ID: "cta-destino", Titular: "Destino", Saldo: 100.0}

		_, err := service.Transferir("cta-pobre", "cta-destino", 500.0)

		if !errors.Is(err, domain.ErrSaldoInsuficiente) {
			t.Errorf("Esperado error '%v', obtenido '%v'", domain.ErrSaldoInsuficiente, err)
		}
	})

	t.Run("Fallo por cuenta origen no encontrada", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		// Solo registramos la de destino; la de origen no existe
		mockCuenta.cuentas["cta-destino"] = &domain.Cuenta{ID: "cta-destino", Titular: "Destino", Saldo: 100.0}

		_, err := service.Transferir("cta-fantasma", "cta-destino", 50.0)

		if !errors.Is(err, domain.ErrCuentaNoEncontrada) {
			t.Errorf("Esperado error '%v', obtenido '%v'", domain.ErrCuentaNoEncontrada, err)
		}
	})
}

// ==========================================================
// 4. TEST: Regla de Eliminación de Cuentas
// ==========================================================

func TestEliminarCuenta(t *testing.T) {
	t.Run("No permite eliminar cuenta con saldo positivo", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		mockCuenta.cuentas["cta-con-plata"] = &domain.Cuenta{ID: "cta-con-plata", Saldo: 150.0}

		err := service.EliminarCuenta("cta-con-plata")
		if !errors.Is(err, domain.ErrCuentaConSaldo) {
			t.Errorf("Esperado error '%v', obtenido '%v'", domain.ErrCuentaConSaldo, err)
		}
	})

	t.Run("Permite eliminar cuenta con saldo cero", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		mockCuenta.cuentas["cta-vacia"] = &domain.Cuenta{ID: "cta-vacia", Saldo: 0.0}

		err := service.EliminarCuenta("cta-vacia")
		if err != nil {
			t.Fatalf("No se esperaba error al eliminar cuenta en cero: %v", err)
		}
		if _, existe := mockCuenta.cuentas["cta-vacia"]; existe {
			t.Errorf("La cuenta debería haber sido eliminada del repositorio")
		}
	})
}
