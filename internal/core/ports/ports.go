package ports

import (
	"goallet-api/internal/core/domain"
)

// ==========================================
// PUERTOS DE SALIDA (Repositorios / Persistencia)
// Contratos que la base de datos DEBE cumplir
// ==========================================

// CuentaRepository define las operaciones de persistencia para Cuentas
type CuentaRepository interface {
	Guardar(cuenta *domain.Cuenta) error
	BuscarPorID(id string) (*domain.Cuenta, error)
	Actualizar(cuenta *domain.Cuenta) error
	Listar() ([]*domain.Cuenta, error)
	Eliminar(id string) error
}

// TransaccionRepository define el guardado y auditoría de movimientos
type TransaccionRepository interface {
	Guardar(tx *domain.Transaccion) error
	ListarPorCuentaID(cuentaID string) ([]domain.Transaccion, error)
}

// ==========================================
// PUERTOS DE ENTRADA (Casos de Uso / Servicios)
// Contratos de lo que la Billetera sabe hacer
// ==========================================

// BilleteraService define los casos de uso que expone nuestra aplicación
type BilleteraService interface {
	CrearCuenta(titular string, saldoInicial float64) (*domain.Cuenta, error)
	ConsultarCuenta(id string) (*domain.Cuenta, error)
	ListarCuentas() ([]*domain.Cuenta, error)
	ActualizarTitular(id string, nuevoTitular string) (*domain.Cuenta, error)
	EliminarCuenta(id string) error
	Transferir(origenID, destinoID string, monto float64) (*domain.Transaccion, error)
	Depositar(cuentaID string, monto float64) (*domain.Transaccion, error)
	Retirar(cuentaID string, monto float64) (*domain.Transaccion, error)
}
