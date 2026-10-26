package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"goallet-api/internal/core/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditRepositoryPostgres struct {
	db *pgxpool.Pool
}

func NewAuditRepositoryPostgres(db *pgxpool.Pool) *AuditRepositoryPostgres {
	return &AuditRepositoryPostgres{db: db}
}

func (r *AuditRepositoryPostgres) GuardarEventoFallido(evento domain.EventoFallido) error {
	query := `
		INSERT INTO eventos_auditoria (transaccion_id, payload, error_detalle, estado, intentos)
		VALUES ($1, $2, $3, $4, $5)
	`

	payloadJSON, err := json.Marshal(evento.Payload)
	if err != nil {
		return fmt.Errorf("error al serializar payload para auditoría: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = r.db.Exec(ctx, query, evento.TransaccionID, payloadJSON, evento.ErrorDetalle, evento.Estado, evento.Intentos)
	if err != nil {
		return fmt.Errorf("error al insertar evento fallido en la base de datos: %w", err)
	}

	return nil
}
