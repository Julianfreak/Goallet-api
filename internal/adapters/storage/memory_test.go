package storage_test

import (
	"testing"

	"goallet-api/internal/adapters/storage"
	"goallet-api/internal/core/domain"
)

func TestMemoryCuentaStorage(t *testing.T) {
	repo := storage.NewMemoryCuentaStorage()

	// 1. Guardar exitoso
	c1 := &domain.Cuenta{ID: "cta-1", Titular: "Julian", Saldo: 100.0}
	if err := repo.Guardar(c1); err != nil {
		t.Fatalf("Error al guardar cuenta: %v", err)
	}

	// 2. Guardar duplicado (error)
	if err := repo.Guardar(c1); err == nil {
		t.Error("Se esperaba error al guardar cuenta duplicada")
	}

	// 3. BuscarPorID exitoso
	encontrada, err := repo.BuscarPorID("cta-1")
	if err != nil || encontrada.Titular != "Julian" {
		t.Fatalf("Error al buscar cuenta existente: %v", err)
	}

	// 4. BuscarPorID inexistente (error)
	if _, err := repo.BuscarPorID("cta-inexistente"); err == nil {
		t.Error("Se esperaba error para cuenta inexistente")
	}

	// 5. Actualizar exitoso
	c1.Saldo = 200.0
	if err := repo.Actualizar(c1); err != nil {
		t.Fatalf("Error al actualizar cuenta: %v", err)
	}

	// 6. Actualizar inexistente (error)
	cInexistente := &domain.Cuenta{ID: "cta-inexistente", Titular: "Fantasma"}
	if err := repo.Actualizar(cInexistente); err == nil {
		t.Error("Se esperaba error al actualizar cuenta inexistente")
	}

	// 7. Listar
	lista, err := repo.Listar()
	if err != nil || len(lista) != 1 {
		t.Fatalf("Error al listar cuentas: %v", err)
	}

	// 8. Eliminar inexistente (error)
	if err := repo.Eliminar("cta-inexistente"); err == nil {
		t.Error("Se esperaba error al eliminar cuenta inexistente")
	}

	// 9. Eliminar exitoso
	if err := repo.Eliminar("cta-1"); err != nil {
		t.Fatalf("Error al eliminar cuenta existente: %v", err)
	}

	// 10. Verificar eliminación
	if _, err := repo.BuscarPorID("cta-1"); err == nil {
		t.Error("La cuenta debía haber sido eliminada")
	}
}

func TestMemoryTransaccionStorage(t *testing.T) {
	repo := storage.NewMemoryTransaccionStorage()

	tx1 := domain.Transaccion{
		ID:              "tx-1",
		CuentaOrigenID:  "cta-1",
		CuentaDestinoID: "cta-2",
		Monto:           50.0,
	}
	tx2 := domain.Transaccion{
		ID:              "tx-2",
		CuentaOrigenID:  "cta-3",
		CuentaDestinoID: "cta-1",
		Monto:           20.0,
	}

	// 1. Guardar
	if err := repo.Guardar(&tx1); err != nil {
		t.Fatalf("Error al guardar transacción 1: %v", err)
	}
	if err := repo.Guardar(&tx2); err != nil {
		t.Fatalf("Error al guardar transacción 2: %v", err)
	}

	// 2. ListarPorCuentaID (origen y destino)
	txs, err := repo.ListarPorCuentaID("cta-1")
	if err != nil || len(txs) != 2 {
		t.Fatalf("Se esperaban 2 transacciones asociadas a cta-1, obtenidas %d", len(txs))
	}

	// 3. Listar sin coincidencias
	txsVacias, err := repo.ListarPorCuentaID("cta-sin-tx")
	if err != nil || len(txsVacias) != 0 {
		t.Fatalf("Se esperaban 0 transacciones, obtenidas %d", len(txsVacias))
	}
}
