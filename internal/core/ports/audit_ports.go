package ports

import "goallet-api/internal/core/domain"

type AuditRepository interface {
	GuardarEventoFallido(evento domain.EventoFallido) error
	ListarPendientes() ([]domain.EventoFallido, error)
	ActualizarEstado(id int, estado string, intentos int) error
}
