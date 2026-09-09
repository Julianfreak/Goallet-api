package storage

import (
	"errors"
	"sync"

	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/ports"
)

// ==========================================================
// 1. REPOSITORIO DE CUENTAS (Implementa ports.CuentaRepository)
// ==========================================================

type MemoryCuentaStorage struct {
	mu      sync.RWMutex
	cuentas map[string]*domain.Cuenta
}

func NewMemoryCuentaStorage() ports.CuentaRepository {
	return &MemoryCuentaStorage{
		cuentas: make(map[string]*domain.Cuenta),
	}
}

func (m *MemoryCuentaStorage) Guardar(cuenta *domain.Cuenta) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, existe := m.cuentas[cuenta.ID]; existe {
		return errors.New("la cuenta ya se encuentra registrada")
	}

	m.cuentas[cuenta.ID] = cuenta
	return nil
}

func (m *MemoryCuentaStorage) BuscarPorID(id string) (*domain.Cuenta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cuenta, existe := m.cuentas[id]
	if !existe {
		return nil, domain.ErrCuentaNoEncontrada
	}

	return cuenta, nil
}

func (m *MemoryCuentaStorage) Actualizar(cuenta *domain.Cuenta) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, existe := m.cuentas[cuenta.ID]; !existe {
		return domain.ErrCuentaNoEncontrada
	}

	m.cuentas[cuenta.ID] = cuenta
	return nil
}

// ==========================================================
// 2. REPOSITORIO DE TRANSACCIONES (Implementa ports.TransaccionRepository)
// ==========================================================

type MemoryTransaccionStorage struct {
	mu            sync.RWMutex
	transacciones []domain.Transaccion
}

func NewMemoryTransaccionStorage() ports.TransaccionRepository {
	return &MemoryTransaccionStorage{
		transacciones: make([]domain.Transaccion, 0),
	}
}

func (m *MemoryTransaccionStorage) Guardar(tx *domain.Transaccion) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.transacciones = append(m.transacciones, *tx)
	return nil
}

func (m *MemoryTransaccionStorage) ListarPorCuentaID(cuentaID string) ([]domain.Transaccion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var resultado []domain.Transaccion
	for _, tx := range m.transacciones {
		if tx.CuentaOrigenID == cuentaID || tx.CuentaDestinoID == cuentaID {
			resultado = append(resultado, tx)
		}
	}

	return resultado, nil
}

// Listar todas las cuentas registradas (Lectura segura)
func (m *MemoryCuentaStorage) Listar() ([]*domain.Cuenta, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cuentas := make([]*domain.Cuenta, 0, len(m.cuentas))
	for _, cuenta := range m.cuentas {
		cuentas = append(cuentas, cuenta)
	}

	return cuentas, nil
}

// Eliminar una cuenta por su ID (Escritura exclusiva segura)
func (m *MemoryCuentaStorage) Eliminar(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, existe := m.cuentas[id]; !existe {
		return domain.ErrCuentaNoEncontrada
	}

	// delete es la función nativa de Go para borrar una clave de un map
	delete(m.cuentas, id)
	return nil
}
