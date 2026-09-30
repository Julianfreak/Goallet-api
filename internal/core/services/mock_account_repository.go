package services

import (
	"goallet-api/internal/core/domain" // Ajusta el módulo según tu go.mod

	"github.com/stretchr/testify/mock"
)

// MockAccountRepository simula el puerto de persistencia
type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) Create(account *domain.Cuenta) error {
	args := m.Called(account)
	return args.Error(0)
}

func (m *MockAccountRepository) GetByID(id string) (*domain.Cuenta, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.Cuenta), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAccountRepository) Update(account *domain.Cuenta) error {
	args := m.Called(account)
	return args.Error(0)
}

// Agrega los demás métodos de tu interfaz (Delete, GetAll, etc.) siguiendo este patrón.
