package ports

import "goallet-api/internal/core/domain"

type EventPublisher interface {
	PublishTransferencia(event domain.TransferenciaEvent) error
}
