package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"goallet-api/internal/core/domain"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQPublisher(amqpURL string) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("error al conectar con RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error al abrir canal de RabbitMQ: %w", err)
	}

	// Declarar un exchange tipo direct/topic para transferencias
	err = ch.ExchangeDeclare(
		"billetera.events", // nombre
		"topic",            // tipo
		true,               // durable
		false,              // auto-deleted
		false,              // internal
		false,              // no-wait
		nil,                // argumentos
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("error al declarar exchange: %w", err)
	}

	return &RabbitMQPublisher{conn: conn, channel: ch}, nil
}

func (p *RabbitMQPublisher) PublishTransferencia(event domain.TransferenciaEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error al serializar evento: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		ctx,
		"billetera.events",         // exchange
		"transferencia.completada", // routing key
		false,                      // mandatory
		false,                      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("error al publicar mensaje en RabbitMQ: %w", err)
	}

	return nil
}

func (p *RabbitMQPublisher) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}
