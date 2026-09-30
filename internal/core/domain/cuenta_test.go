package domain_test

import (
	"testing"

	"goallet-api/internal/core/domain"
)

func TestReglasDominioCuenta(t *testing.T) {
	c := &domain.Cuenta{ID: "1", Titular: "Julian", Saldo: 100.0}

	// Depositar válido e inválido
	if err := c.Depositar(50.0); err != nil || c.Saldo != 150.0 {
		t.Errorf("Fallo en depósito de dominio")
	}
	if err := c.Depositar(-10.0); err == nil {
		t.Errorf("Debía fallar con monto negativo")
	}

	// Retirar válido e inválido
	if err := c.Retirar(50.0); err != nil || c.Saldo != 100.0 {
		t.Errorf("Fallo en retiro de dominio")
	}
	if err := c.Retirar(500.0); err == nil {
		t.Errorf("Debía fallar por saldo insuficiente")
	}
}
