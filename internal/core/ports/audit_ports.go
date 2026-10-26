package ports

import "goallet-api/internal/core/domain"

type AuditRepository interface {
	GuardarEventoFallido(evento domain.EventoFallido) error
}
