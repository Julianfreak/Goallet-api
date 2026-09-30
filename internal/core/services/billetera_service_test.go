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
	cuentas               map[string]*domain.Cuenta
	forzarErrorGuardar    error
	forzarErrorBuscar     error
	forzarErrorActualizar error
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
	if m.forzarErrorActualizar != nil {
		return m.forzarErrorActualizar
	}
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
	transacciones      []domain.Transaccion
	forzarErrorGuardar error
}

func NewMockTxRepo() *MockTxRepo {
	return &MockTxRepo{
		transacciones: make([]domain.Transaccion, 0),
	}
}

func (m *MockTxRepo) Guardar(tx *domain.Transaccion) error {
	if m.forzarErrorGuardar != nil {
		return m.forzarErrorGuardar
	}
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

// ==========================================================
// 5. TEST: Consultar Cuenta y Listar
// ==========================================================

func TestConsultarCuenta(t *testing.T) {
	t.Run("Fallo por ID vacío", func(t *testing.T) {
		service := services.NewBilleteraService(NewMockCuentaRepo(), NewMockTxRepo())
		_, err := service.ConsultarCuenta("")
		if err == nil || err.Error() != "el ID de la cuenta es obligatorio" {
			t.Errorf("Se esperaba error de ID obligatorio")
		}
	})

	t.Run("Fallo por cuenta no encontrada (return nil, err)", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockCuenta.forzarErrorBuscar = errors.New("error de base de datos")
		service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())

		_, err := service.ConsultarCuenta("id-falso")
		if err == nil {
			t.Errorf("Se esperaba que pasara el error del repositorio hacia arriba")
		}
	})
}

func TestListarCuentas(t *testing.T) {
	mockCuenta := NewMockCuentaRepo()
	mockCuenta.cuentas["1"] = &domain.Cuenta{ID: "1"}
	service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())

	lista, err := service.ListarCuentas()
	if err != nil || len(lista) != 1 {
		t.Errorf("Se esperaba obtener la lista correctamente")
	}
}

// ==========================================================
// 6. TEST: Depositar y Retirar (Cubriendo los return nil, err)
// ==========================================================

func TestDepositar_ErroresInfraestructura(t *testing.T) {
	t.Run("Fallo al actualizar el repositorio de cuentas", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		mockCuenta.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 100.0}
		// Simulamos que la base de datos falla justo al hacer el UPDATE
		mockCuenta.forzarErrorActualizar = errors.New("timeout en DB cuentas")

		_, err := service.Depositar("cta-1", 50.0)
		if err == nil {
			t.Errorf("Se esperaba error al actualizar repositorio")
		}
	})

	t.Run("Fallo al guardar en repositorio de transacciones", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		mockCuenta.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 100.0}
		// Simulamos que la base de datos falla al guardar el historial (auditoría)
		mockTx.forzarErrorGuardar = errors.New("timeout en DB transacciones")

		_, err := service.Depositar("cta-1", 50.0)
		if err == nil {
			t.Errorf("Se esperaba error al guardar transacción")
		}
	})
}

func TestRetirar_ErroresInfraestructura(t *testing.T) {
	t.Run("Fallo por error al buscar cuenta", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockCuenta.forzarErrorBuscar = errors.New("db error")
		service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())

		_, err := service.Retirar("cta-1", 50.0)
		if err == nil {
			t.Errorf("Se esperaba error al buscar cuenta")
		}
	})

	t.Run("Fallo al actualizar el saldo", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockCuenta.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 100.0}
		mockCuenta.forzarErrorActualizar = errors.New("db error al actualizar")

		service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())
		_, err := service.Retirar("cta-1", 50.0)
		if err == nil {
			t.Errorf("Se esperaba error al intentar guardar nuevo saldo")
		}
	})
}

// ==========================================================
// 7. TEST: Actualizar Titular
// ==========================================================

func TestActualizarTitular(t *testing.T) {
	t.Run("Fallo por nombre vacío", func(t *testing.T) {
		service := services.NewBilleteraService(NewMockCuentaRepo(), NewMockTxRepo())
		_, err := service.ActualizarTitular("cta-1", "")
		if err == nil || err.Error() != "el nuevo titular no puede estar vacío" {
			t.Errorf("Se esperaba error por titular vacío")
		}
	})

	t.Run("Fallo porque cuenta no existe", func(t *testing.T) {
		service := services.NewBilleteraService(NewMockCuentaRepo(), NewMockTxRepo())
		_, err := service.ActualizarTitular("no-existe", "Nuevo Titular")
		if err == nil {
			t.Errorf("Se esperaba error de cuenta no encontrada")
		}
	})

	t.Run("Éxito al actualizar", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockCuenta.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Titular: "Viejo"}
		service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())

		cuenta, err := service.ActualizarTitular("cta-1", "Nuevo Titular")
		if err != nil || cuenta.Titular != "Nuevo Titular" {
			t.Errorf("Se esperaba actualización exitosa")
		}
	})
}

// ==========================================================
// 8. TEST: Completar cobertura de Retirar
// ==========================================================

func TestRetirar_CoberturaFinal(t *testing.T) {
	t.Run("Retiro exitoso completo", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		// Preparamos una cuenta con saldo suficiente
		mockCuenta.cuentas["cta-retiro"] = &domain.Cuenta{ID: "cta-retiro", Titular: "Julian", Saldo: 500.0}

		// Ejecutamos el retiro
		tx, err := service.Retirar("cta-retiro", 150.0)

		// Verificaciones
		if err != nil {
			t.Fatalf("No se esperaba error en un retiro válido, se obtuvo: %v", err)
		}
		if tx == nil {
			t.Fatalf("Se esperaba recibir la transacción creada, pero fue nil")
		}
		if tx.Monto != 150.0 {
			t.Errorf("El monto de la transacción debía ser 150.0, fue %.2f", tx.Monto)
		}
		if mockCuenta.cuentas["cta-retiro"].Saldo != 350.0 {
			t.Errorf("El saldo final debía ser 350.0, fue %.2f", mockCuenta.cuentas["cta-retiro"].Saldo)
		}
	})

	t.Run("Fallo al guardar transacción de retiro", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockTx := NewMockTxRepo()
		service := services.NewBilleteraService(mockCuenta, mockTx)

		mockCuenta.cuentas["cta-retiro"] = &domain.Cuenta{ID: "cta-retiro", Saldo: 500.0}

		// Forzamos el error justo en el paso E (Guardar en auditoría)
		mockTx.forzarErrorGuardar = errors.New("falla al conectar con base de datos de transacciones")

		_, err := service.Retirar("cta-retiro", 100.0)
		if err == nil {
			t.Errorf("Se esperaba un error al fallar el guardado de la transacción")
		}
	})
}

// ==========================================================
// 9. TEST: Cubrir los return nil, err restantes
// ==========================================================

func TestDepositar_ErroresRestantes(t *testing.T) {
	// 1. Cuando falla BuscarPorID al depositar
	t.Run("Fallo porque la cuenta no existe", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockCuenta.forzarErrorBuscar = errors.New("cuenta no encontrada")
		service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())

		_, err := service.Depositar("id-inexistente", 100.0)
		if err == nil {
			t.Errorf("Se esperaba error al no encontrar la cuenta en el depósito")
		}
	})

	// 2. Cuando falla la regla de dominio (ej. depositar monto negativo o cero)
	t.Run("Fallo por regla de negocio del dominio", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockCuenta.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1"}
		service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())

		// Le pasamos un monto inválido para que cuenta.Depositar() lance error
		_, err := service.Depositar("cta-1", -50.0)
		if err == nil {
			t.Errorf("Se esperaba error desde el dominio por monto inválido")
		}
	})
}

func TestRetirar_ErroresDominio(t *testing.T) {
	// 3. Cuando falla la regla de dominio (saldo insuficiente)
	t.Run("Fallo por saldo insuficiente", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		// Creamos una cuenta con solo 10.0 de saldo
		mockCuenta.cuentas["cta-pobre"] = &domain.Cuenta{ID: "cta-pobre", Saldo: 10.0}
		service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())

		// Intentamos retirar 100.0
		_, err := service.Retirar("cta-pobre", 100.0)
		if err == nil {
			t.Errorf("Se esperaba error por saldo insuficiente desde el dominio")
		}
	})
}

func TestActualizarTitular_ErrorActualizar(t *testing.T) {
	// 4. Cuando falla la actualización en la BD al cambiar el titular
	t.Run("Fallo en base de datos al guardar titular", func(t *testing.T) {
		mockCuenta := NewMockCuentaRepo()
		mockCuenta.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Titular: "Julian"}
		// Forzamos el error justo antes de hacer el update
		mockCuenta.forzarErrorActualizar = errors.New("timeout BD")
		service := services.NewBilleteraService(mockCuenta, NewMockTxRepo())

		_, err := service.ActualizarTitular("cta-1", "Nuevo Nombre")
		if err == nil {
			t.Errorf("Se esperaba error al fallar el update en la base de datos")
		}
	})
}
