package ports

import "goallet-api/internal/core/domain"

type EventPublisher interface {
	PublishTransferencia(event domain.TransferenciaEvent) error
}

type EventConsumer interface {
	StartConsuming(queueName, routingKey string) error
	Stop() error
}
