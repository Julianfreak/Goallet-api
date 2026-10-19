package middleware

import (
	"bytes"
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func IdempotencyMiddleware(dbPool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		idempotencyKey := c.GetHeader("X-Idempotency-Key")

		// Si la petición no envía la clave, continúa la ejecución normal
		if idempotencyKey == "" {
			c.Next()
			return
		}

		ctx := context.Background()

		// 1. Verificar si la clave ya fue procesada anteriormente
		var statusCode int
		var responseBody string
		err := dbPool.QueryRow(ctx, "SELECT status_code, response_body FROM idempotency_keys WHERE key = $1", idempotencyKey).Scan(&statusCode, &responseBody)

		if err == nil {
			// Clave encontrada: retornar la respuesta previa de forma idéntica
			c.Header("X-Cache-Lookup", "HIT")
			c.Header("Content-Type", "application/json")
			c.String(statusCode, responseBody)
			c.Abort()
			return
		}

		// 2. Interceptar la respuesta de la solicitud actual
		w := &responseWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		// 3. Guardar en la base de datos si la operación fue exitosa (2xx)
		if c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			_, _ = dbPool.Exec(ctx,
				"INSERT INTO idempotency_keys (key, status_code, response_body) VALUES ($1, $2, $3) ON CONFLICT (key) DO NOTHING",
				idempotencyKey, c.Writer.Status(), w.body.String(),
			)
		}
	}
}
