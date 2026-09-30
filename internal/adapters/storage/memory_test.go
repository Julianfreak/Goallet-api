package storage_test

import (
	"testing"

	"goallet-api/internal/adapters/storage"
	"goallet-api/internal/core/domain"
)

func TestMemoriaStorage(t *testing.T) {
	repo := storage.NewMemoryCuentaStorage() // Ajusta el constructor si varía en tu código

	// 1. Guardar y Buscar
	cuenta := &domain.Cuenta{ID: "c1", Titular: "Julian", Saldo: 100.0}
	if err := repo.Guardar(cuenta); err != nil {
		t.Fatalf("Error al guardar cuenta: %v", err)
	}

	c, err := repo.BuscarPorID("c1")
	if err != nil || c.Titular != "Julian" {
		t.Fatalf("Error al buscar por ID: %v", err)
	}

	// 2. Buscar cuenta inexistente
	if _, err := repo.BuscarPorID("fantasma"); err == nil {
		t.Errorf("Se esperaba error para cuenta inexistente")
	}

	// 3. Actualizar
	cuenta.Saldo = 200.0
	if err := repo.Actualizar(cuenta); err != nil {
		t.Fatalf("Error al actualizar: %v", err)
	}

	// 4. Listar
	lista, err := repo.Listar()
	if err != nil || len(lista) != 1 {
		t.Fatalf("Error al listar cuentas")
	}

	// 5. Eliminar
	if err := repo.Eliminar("c1"); err != nil {
		t.Fatalf("Error al eliminar: %v", err)
	}

	if _, err := repo.BuscarPorID("c1"); err == nil {
		t.Errorf("La cuenta debió ser eliminada")
	}
}
