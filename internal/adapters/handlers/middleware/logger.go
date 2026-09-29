package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// SlogLogger es un middleware para Gin que emite logs en formato JSON mediante slog.
func SlogLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		attrs := []any{
			slog.Int("status", statusCode),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", latency),
			slog.String("user_agent", c.Request.UserAgent()),
		}

		if len(c.Errors) > 0 {
			slog.Error("Petición HTTP completada con errores",
				slog.Group("http", attrs...),
				slog.String("error", c.Errors.String()),
			)
		} else if statusCode >= 400 {
			slog.Warn("Petición HTTP con estado de advertencia/error", attrs...)
		} else {
			slog.Info("Petición HTTP procesada", attrs...)
		}
	}
}
