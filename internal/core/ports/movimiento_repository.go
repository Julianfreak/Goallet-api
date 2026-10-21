package ports

import (
	"context"
	"goallet-api/internal/core/domain"
	"time"
)

type MovimientoRepository interface {
	// Registrar inserta un movimiento financiero dentro de una transacción activa
	Registrar(ctx context.Context, tx interface{}, mov *domain.Movimiento) error

	// ObtenerPorCuentaID recupera el historial de movimientos de una cuenta específica
	ObtenerPorCuentaID(ctx context.Context, cuentaID string) ([]domain.Movimiento, error)
	ObtenerConFiltros(ctx context.Context, cuentaID string, fechaInicio, fechaFin *time.Time, limit, offset int) ([]domain.Movimiento, error)
}
