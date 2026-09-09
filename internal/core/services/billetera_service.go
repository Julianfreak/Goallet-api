package services

import (
	"errors"
	"fmt"
	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/ports"
	"time"
)

// billeteraService es el "Cajero del Banco".
// Guarda referencias a los puertos de salida (Repositorios).
type billeteraService struct {
	cuentaRepo      ports.CuentaRepository
	transaccionRepo ports.TransaccionRepository
}

// NewBilleteraService es la función constructora (Inyección de Dependencias).
// Devuelve la interfaz ports.BilleteraService para garantizar el cumplimiento del contrato.
func NewBilleteraService(cuentaRepo ports.CuentaRepository, transaccionRepo ports.TransaccionRepository) ports.BilleteraService {
	return &billeteraService{
		cuentaRepo:      cuentaRepo,
		transaccionRepo: transaccionRepo,
	}
}

// 1. Caso de Uso: Crear Cuenta
func (s *billeteraService) CrearCuenta(titular string, saldoInicial float64) (*domain.Cuenta, error) {
	if titular == "" {
		return nil, errors.New("el titular de la cuenta no puede estar vacío")
	}
	if saldoInicial < 0 {
		return nil, domain.ErrMontoInvalido
	}

	// Generamos un identificador único basado en nanosegundos
	cuentaID := fmt.Sprintf("cta-%d", time.Now().UnixNano())

	nuevaCuenta := &domain.Cuenta{
		ID:        cuentaID,
		Titular:   titular,
		Saldo:     saldoInicial,
		CreatedAt: time.Now(),
	}

	// Le pedimos a la persistencia que guarde la cuenta
	if err := s.cuentaRepo.Guardar(nuevaCuenta); err != nil {
		return nil, err
	}

	return nuevaCuenta, nil
}

// 2. Caso de Uso: Consultar Cuenta
func (s *billeteraService) ConsultarCuenta(id string) (*domain.Cuenta, error) {
	if id == "" {
		return nil, errors.New("el ID de la cuenta es obligatorio")
	}
	return s.cuentaRepo.BuscarPorID(id)
}

// 3. Caso de Uso: Depositar Dinero
func (s *billeteraService) Depositar(cuentaID string, monto float64) (*domain.Transaccion, error) {
	// A. El cajero busca la cuenta en el archivo
	cuenta, err := s.cuentaRepo.BuscarPorID(cuentaID)
	if err != nil {
		return nil, err
	}

	// B. Le ordena a la cuenta ejecutar su regla de dominio
	if err := cuenta.Depositar(monto); err != nil {
		return nil, err
	}

	// C. Guarda el nuevo saldo en el repositorio
	if err := s.cuentaRepo.Actualizar(cuenta); err != nil {
		return nil, err
	}

	// D. Crea el comprobante contable de la transacción
	txID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
	transaccion := &domain.Transaccion{
		ID:             txID,
		CuentaOrigenID: cuentaID,
		Tipo:           domain.Deposito,
		Monto:          monto,
		Fecha:          time.Now(),
	}

	// E. Guarda la transacción para auditoría
	if err := s.transaccionRepo.Guardar(transaccion); err != nil {
		return nil, err
	}

	return transaccion, nil
}
func (s *billeteraService) Retirar(cuentaID string, monto float64) (*domain.Transaccion, error) {
	// A. Buscar la cuenta
	cuenta, err := s.cuentaRepo.BuscarPorID(cuentaID)
	if err != nil {
		return nil, err
	}

	// B. Ejecutar la regla de retiro del dominio (valida si hay saldo suficiente)
	if err := cuenta.Retirar(monto); err != nil {
		return nil, err
	}

	// C. Guardar el nuevo saldo en el repositorio
	if err := s.cuentaRepo.Actualizar(cuenta); err != nil {
		return nil, err
	}

	// D. Crear el registro contable del retiro
	transaccion := &domain.Transaccion{
		ID:             fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		CuentaOrigenID: cuentaID,
		Tipo:           domain.Retiro,
		Monto:          monto,
		Fecha:          time.Now(),
	}

	// E. Guardar en auditoría
	if err := s.transaccionRepo.Guardar(transaccion); err != nil {
		return nil, err
	}

	return transaccion, nil
}

func (s *billeteraService) Transferir(origenID, destinoID string, monto float64) (*domain.Transaccion, error) {
	if origenID == destinoID {
		return nil, domain.ErrMismaCuentaDestino
	}
	cuentaOrigen, err := s.cuentaRepo.BuscarPorID(origenID)
	if err != nil {
		return nil, err
	}
	cuentaDestino, err := s.cuentaRepo.BuscarPorID(destinoID)
	if err != nil {
		return nil, err
	}

	if err := cuentaOrigen.Retirar(monto); err != nil {
		return nil, err
	}
	if err := cuentaDestino.Depositar(monto); err != nil {
		return nil, err
	}

	if err := s.cuentaRepo.Actualizar(cuentaOrigen); err != nil {
		return nil, err
	}
	if err := s.cuentaRepo.Actualizar(cuentaDestino); err != nil {
		return nil, err
	}
	transaccion := &domain.Transaccion{
		ID:              fmt.Sprintf("tx-%d", time.Now().UnixNano()),
		CuentaOrigenID:  origenID,
		CuentaDestinoID: destinoID,
		Tipo:            domain.Transferencia,
		Monto:           monto,
		Fecha:           time.Now(),
	}
	if err := s.transaccionRepo.Guardar(transaccion); err != nil {
		return nil, err
	}

	return transaccion, nil
}
func (s *billeteraService) ListarCuentas() ([]*domain.Cuenta, error) {
	return s.cuentaRepo.Listar()
}

func (s *billeteraService) ActualizarTitular(id string, nuevoTitular string) (*domain.Cuenta, error) {
	if nuevoTitular == "" {
		return nil, errors.New("el nuevo titular no puede estar vacío")
	}
	cuenta, err := s.cuentaRepo.BuscarPorID(id)
	if err != nil {
		return nil, err
	}
	cuenta.Titular = nuevoTitular
	if err := s.cuentaRepo.Actualizar(cuenta); err != nil {
		return nil, err
	}
	return cuenta, nil
}

func (s *billeteraService) EliminarCuenta(id string) error {
	cuenta, err := s.cuentaRepo.BuscarPorID(id)
	if cuenta.Saldo > 0 {
		return domain.ErrCuentaConSaldo
	}
	if cuenta.Saldo == 0 {
		return s.cuentaRepo.Eliminar(id)
	}
	return err
}
