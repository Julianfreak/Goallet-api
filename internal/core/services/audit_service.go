package services

import (
	"context"
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
func (s *AuditService) ReprocesarEventos(ctx context.Context) error {

	slog.Info("Iniciando ciclo de reproceso de eventos de auditoría...")

	return nil
}

// IniciarWorkerReproceso levanta una goroutine que ejecuta el reproceso periódicamente
func (s *AuditService) IniciarWorkerReproceso(intervalo time.Duration) {
	ticker := time.NewTicker(intervalo)
	go func() {
		for range ticker.C {
			ctx := context.Background()
			if err := s.ReprocesarEventos(ctx); err != nil {
				slog.Error("Error en worker de reproceso de auditoría", "error", err)
			}
		}
	}()
	slog.Info("Worker de reproceso de auditoría iniciado", "intervalo", intervalo.String())
}
