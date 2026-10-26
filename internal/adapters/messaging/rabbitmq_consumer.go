package messaging

import (
	"encoding/json"
	"fmt"
	"goallet-api/internal/core/domain"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQConsumer(amqpURL string) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("error al conectar consumer a RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error al abrir canal en consumer: %w", err)
	}

	return &RabbitMQConsumer{
		conn:    conn,
		channel: ch,
	}, nil
}

func (c *RabbitMQConsumer) StartConsuming(queueName, routingKey string) error {
	// 1. Declarar Exchange
	err := c.channel.ExchangeDeclare(
		"billetera.events",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error al declarar exchange en consumer: %w", err)
	}

	// 2. Declarar Cola durable
	q, err := c.channel.QueueDeclare(
		queueName, // Ej: "billetera.notificaciones"
		true,      // durable
		false,     // auto-delete
		false,     // exclusive
		false,     // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("error al declarar cola: %w", err)
	}

	// 3. Crear Binding entre Exchange y Cola
	err = c.channel.QueueBind(
		q.Name,
		routingKey, // Ej: "transferencia.completada"
		"billetera.events",
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error al enlazar cola con exchange: %w", err)
	}

	// 4. Suscribirse a los mensajes
	msgs, err := c.channel.Consume(
		q.Name,
		"",    // consumer tag
		false, // auto-ack (false para confirmar manualmente solo tras procesar éxito)
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error al iniciar consumo: %w", err)
	}

	// 5. Goroutine en segundo plano escuchando la cola
	go func() {
		for msg := range msgs {
			var evento domain.TransferenciaEvent
			if err := json.Unmarshal(msg.Body, &evento); err != nil {
				slog.Error("Error al deserializar evento de RabbitMQ", "error", err)
				msg.Nack(false, false) // Descartar si el JSON es inválido
				continue
			}

			// Lógica de procesamiento (ejemplo: simular envío de notificación)
			slog.Info("📬 Evento recibido y procesado por Consumer",
				"transaccion_id", evento.ID,
				"origen", evento.CuentaOrigenID,
				"destino", evento.CuentaDestinoID,
				"monto", evento.Monto,
			)

			// Confirmación manual a RabbitMQ de que el mensaje fue procesado
			msg.Ack(false)
		}
	}()

	slog.Info("Worker de RabbitMQ escuchando mensajes", "queue", queueName, "routing_key", routingKey)
	return nil
}

func (c *RabbitMQConsumer) Stop() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
