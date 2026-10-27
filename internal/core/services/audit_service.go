package services

import (
	"encoding/json"
	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/ports"
	"log/slog"
	"time"
)

type AuditService struct {
	auditRepo ports.AuditRepository
	eventPub  ports.EventPublisher
}

func NewAuditService(auditRepo ports.AuditRepository, eventPub ports.EventPublisher) *AuditService {
	return &AuditService{
		auditRepo: auditRepo,
		eventPub:  eventPub,
	}
}

// ReprocesarEventos lee eventos fallidos y reintenta su publicación
func (s *AuditService) ReprocesarEventos() error {
	pendientes, err := s.auditRepo.ListarPendientes()
	if err != nil {
		return err // Propaga el error de listado
	}

	for _, evento := range pendientes {
		var transferenciaEvent domain.TransferenciaEvent
		payloadBytes, ok := evento.Payload.([]byte)
		if !ok {
			continue
		}

		if err := json.Unmarshal(payloadBytes, &transferenciaEvent); err != nil {
			continue
		}

		errPub := s.eventPub.PublishTransferencia(transferenciaEvent)
		if errPub == nil {
			if errUpd := s.auditRepo.ActualizarEstado(evento.ID, "PROCESADO", evento.Intentos); errUpd != nil {
				return errUpd // Propaga el error de actualización si falla
			}
		}
	}
	return nil
}

// IniciarWorkerReproceso levanta una goroutine que ejecuta el reproceso periódicamente
func (s *AuditService) IniciarWorkerReproceso(intervalo time.Duration) {
	ticker := time.NewTicker(intervalo)
	go func() {
		for range ticker.C {
			// Llamada corregida sin argumentos ya que ReprocesarEventos no los requiere
			if err := s.ReprocesarEventos(); err != nil {
				slog.Error("Error en worker de reproceso de auditoría", "error", err)
			}
		}
	}()
	slog.Info("Worker de reproceso de auditoría iniciado", "intervalo", intervalo.String())
}
