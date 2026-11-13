package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthResponse es la estructura de respuesta del endpoint de comprobación de estado.
type HealthResponse struct {
	Status    string `json:"status" example:"UP"`
	Service   string `json:"service" example:"goallet-api"`
	Timestamp string `json:"timestamp" example:"2026-11-13T12:00:00Z"`
	Database  string `json:"database,omitempty" example:"connected"`
}

// HealthHandler maneja las peticiones de comprobación de estado del servicio.
// Recibe el pool de conexiones a PostgreSQL por inyección de dependencias
// para verificar la disponibilidad de la base de datos (readiness probe).
type HealthHandler struct {
	dbPool *pgxpool.Pool
}

// NewHealthHandler crea una nueva instancia del handler de health check.
// Si dbPool es nil, el endpoint solo reporta liveness (sin verificar la BD).
func NewHealthHandler(dbPool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{dbPool: dbPool}
}

// HealthCheck godoc
// @Summary      Comprobación de estado del servicio
// @Description  Verifica que la API esté operativa y, si hay pool de BD, que la conexión a PostgreSQL responda. Usado por el ALB de AWS ECS Fargate.
// @Tags         Health
// @Produce      json
// @Success      200  {object}  HealthResponse "Servicio operativo"
// @Failure      503  {object}  HealthResponse "Base de datos no disponible"
// @Router       /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	response := HealthResponse{
		Status:    "UP",
		Service:   "goallet-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Si hay pool de BD, verificar conectividad (readiness probe)
	if h.dbPool != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := h.dbPool.Ping(ctx); err != nil {
			response.Status = "DOWN"
			response.Database = "disconnected"
			c.JSON(http.StatusServiceUnavailable, response)
			return
		}
		response.Database = "connected"
	}

	c.JSON(http.StatusOK, response)
}
