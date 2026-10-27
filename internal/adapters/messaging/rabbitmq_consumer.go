package messaging

import (
	"encoding/json"
	"errors"
	"fmt"
	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/ports"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

const MaxRetries = 3
const RetryHeaderKey = "x-retry-count"

type RabbitMQConsumer struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	auditRepo ports.AuditRepository
}

func NewRabbitMQConsumer(amqpURL string, auditRepo ports.AuditRepository) (*RabbitMQConsumer, error) {
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
		conn:      conn,
		channel:   ch,
		auditRepo: auditRepo,
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
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("error al declarar cola: %w", err)
	}

	// 3. Crear Binding entre Exchange y Cola
	err = c.channel.QueueBind(
		q.Name,
		routingKey,
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
		"",
		false, // auto-ack false para control manual
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

			// Simulamos procesamiento (puedes reemplazar esto por tu lógica de negocio en el consumer)
			errProcesamiento := c.procesarEvento(evento)

			if errProcesamiento != nil {
				// Obtener o inicializar el contador de reintentos desde las cabeceras
				retryCount := getRetryCount(msg.Headers)

				if retryCount < MaxRetries {
					retryCount++
					slog.Warn("Error procesando evento, reintentando...",
						"transaccion_id", evento.ID,
						"intento", retryCount,
						"error", errProcesamiento,
					)

					// Publicar de nuevo a la cola incrementando el contador en las cabeceras
					errPublish := c.publishWithRetry(q.Name, msg.Body, retryCount)
					if errPublish != nil {
						slog.Error("Error al republicar mensaje para reintento", "error", errPublish)
					}

					// Confirmar el mensaje actual para sacarlo de la cabecera de entrega inmediata
					msg.Ack(false)
					continue
				} else {
					// Se agotaron los reintentos: Persistir en la tabla de auditoría
					slog.Error("Límite de reintentos agotado. Registrando en auditoría.",
						"transaccion_id", evento.ID,
						"intentos", retryCount,
					)

					if c.auditRepo != nil {
						eventoFallido := domain.EventoFallido{
							TransaccionID: evento.ID,
							Payload:       msg.Body,
							ErrorDetalle:  fmt.Sprintf("Agotó max reintentos (%d): %v", MaxRetries, errProcesamiento),
							Estado:        "FALLIDO_MAX_REINTENTOS",
							Intentos:      retryCount,
						}
						if errAudit := c.auditRepo.GuardarEventoFallido(eventoFallido); errAudit != nil {
							slog.Error("Error al guardar evento fallido en auditoría", "error", errAudit)
						}
					}

					// Rechazar definitivamente sin reencolar (va a DLX si está configurado, o se descarta)
					msg.Nack(false, false)
					continue
				}
			}

			// Éxito en el procesamiento
			slog.Info("Evento recibido y procesado exitosamente por Consumer",
				"transaccion_id", evento.ID,
			)
			msg.Ack(false)
		}
	}()

	slog.Info("Worker de RabbitMQ escuchando mensajes", "queue", queueName, "routing_key", routingKey)
	return nil
}

func (c *RabbitMQConsumer) procesarEvento(evento domain.TransferenciaEvent) error {

	return errors.New("error simulado de procesamiento")
}

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	val, ok := headers[RetryHeaderKey]
	if !ok {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}

func (c *RabbitMQConsumer) publishWithRetry(queueName string, body []byte, retryCount int) error {
	headers := amqp.Table{
		RetryHeaderKey: retryCount,
	}

	return c.channel.Publish(
		"billetera.events",
		"transferencia.completada",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Headers:     headers,
		},
	)
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
