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

func (r *AuditRepositoryPostgres) ListarPendientes() ([]domain.EventoFallido, error) {
	query := `SELECT id, transaccion_id, payload, error_detalle, estado, intentos, created_at 
              FROM eventos_auditoria 
              WHERE estado IN ('FALLIDO', 'FALLIDO_MAX_REINTENTOS') 
              ORDER BY created_at ASC LIMIT 50`
	rows, err := r.db.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("error al listar eventos pendientes: %w", err)
	}
	defer rows.Close()

	var eventos []domain.EventoFallido
	for rows.Next() {
		var evento domain.EventoFallido
		if err := rows.Scan(&evento.ID, &evento.TransaccionID, &evento.Payload, &evento.ErrorDetalle, &evento.Estado, &evento.Intentos, &evento.CreatedAt); err != nil {
			return nil, fmt.Errorf("error al escanear evento pendiente: %w", err)
		}
		eventos = append(eventos, evento)
	}
	return eventos, nil
}

func (r *AuditRepositoryPostgres) ActualizarEstado(id int, estado string, intentos int) error {
	query := `UPDATE eventos_auditoria SET estado = $1, intentos = $2 WHERE id = $3`
	_, err := r.db.Exec(context.Background(), query, estado, intentos, id)
	return err
}
