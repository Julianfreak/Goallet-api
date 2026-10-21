package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "goallet-api/docs"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/time/rate"

	"goallet-api/internal/adapters/handlers"
	"goallet-api/internal/adapters/handlers/middleware"
	"goallet-api/internal/adapters/storage"
	"goallet-api/internal/core/services"
)

// @title           Goallet API - Motor Transaccional de Billetera Digital
// @version         1.0
// @description     API RESTful para gestión de cuentas, transacciones y auditoría financiera con arquitectura hexagonal.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Julian Dario Serna Saavedra
// @contact.url    https://github.com/Julianfreak/Goallet-api
// @contact.email  julianrideon@gmail.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8090
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Escribe "Bearer " seguido de tu token JWT. Ejemplo: "Bearer eyJhbGciOi..."

func main() {
	fmt.Println("==========================================================")
	fmt.Println("   Goallet API - Motor Transaccional de Billetera Digital  ")
	fmt.Println("==========================================================")

	ctx := context.Background()

	// 1. Configuración de Logs JSON (slog) hacia la salida estándar (Stdout)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Servidor de Métricas Prometheus en Goroutine paralela (:2112)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		slog.Info("Servidor de métricas Prometheus escuchando en :2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			slog.Error("Error iniciando servidor de métricas Prometheus", "error", err)
		}
	}()

	// 3. Configuración y Conexión al Pool de PostgreSQL (pgxpool)
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "goallet_user")
	dbPassword := getEnv("DB_PASSWORD", "goallet_password")
	dbName := getEnv("DB_NAME", "goallet_db")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, dbSSLMode,
	)

	dbPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		slog.Error("Error crítico al crear el pool de conexiones a PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		slog.Error("No se pudo establecer conexión con PostgreSQL (Ping fallido)", "error", err)
		os.Exit(1)
	}
	slog.Info("Conexión exitosa y verificada con PostgreSQL (pgxpool)")

	// 4. Instanciamos ambos adaptadores de salida (Persistencia PostgreSQL)
	cuentaRepo := storage.NewPostgresCuentaStorage(dbPool)
	transaccionRepo := storage.NewPostgresTransaccionStorage(dbPool)
	movimientoRepo := storage.NewMovimientoRepositoryPostgres(dbPool)

	// Inyectamos ambos repositorios en el Servicio (Núcleo)
	billeteraService := services.NewBilleteraService(cuentaRepo, transaccionRepo, movimientoRepo)

	// 5. Inyectamos el servicio en el Adaptador de entrada (Handler HTTP)
	billeteraHandler := handlers.NewBilleteraHandler(billeteraService)
	authHandler := handlers.NewAuthHandler()

	// 6. Inicializamos Gin sin el logger por defecto (usaremos slog en formato JSON)
	router := gin.New()
	router.Use(gin.Recovery())         // Captura panics de forma segura
	router.Use(jsonLoggerMiddleware()) // Middleware de registro estructurado JSON

	// --- RUTA INTERACTIVA SWAGGER UI ---
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 7. Mapeamos las rutas de la API REST
	v1 := router.Group("/api/v1")
	// Aplicar Rate Limiter global a todas las rutas de la API (ej. 10 req/s, ráfaga de 20)
	limiter := middleware.NewIPRateLimiter(rate.Limit(10), 20)
	v1.Use(middleware.RateLimitMiddleware(limiter))
	{
		// --- RUTA PÚBLICA DE AUTENTICACIÓN ---
		v1.POST("/auth/token", authHandler.GenerarToken)
		// Middleware de Idempotencia para operaciones financieras
		idempotency := middleware.IdempotencyMiddleware(dbPool)

		// --- RUTAS PROTEGIDAS CON JWT ---
		protegidas := v1.Group("/cuentas")
		protegidas.Use(middleware.AuthMiddleware()) // Middleware que valida el token Bearer
		{
			// CRUD de Cuentas
			protegidas.POST("", billeteraHandler.CrearCuenta)          // Create
			protegidas.GET("", billeteraHandler.ListarCuentas)         // Read (todas)
			protegidas.GET("/:id", billeteraHandler.ConsultarCuenta)   // Read (por ID)
			protegidas.PUT("/:id", billeteraHandler.ActualizarTitular) // Update (solo titular)
			protegidas.DELETE("/:id", billeteraHandler.EliminarCuenta) // Delete (solo si saldo == 0)

			// Operaciones Transaccionales
			protegidas.POST("/:id/depositar", idempotency, billeteraHandler.Depositar)
			protegidas.POST("/:id/retirar", idempotency, billeteraHandler.Retirar)
			protegidas.POST("/:id/transferir", idempotency, billeteraHandler.Transferir)

			// Historial / Auditoría (Ledger)
			protegidas.GET("/:id/movimientos", billeteraHandler.ConsultarMovimientos)
		}
	}

	// 8. Configuración del Servidor HTTP con soporte para Graceful Shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Canal en goroutine paralela para iniciar el servidor HTTP sin bloquear la ejecución principal
	go func() {
		slog.Info("Servidor HTTP iniciado exitosamente", "url", "http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Error no esperado en el servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	// 9. Captura de señales de interrupción del sistema operativo (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Bloquea la ejecución hasta que se reciba una señal de apagado
	sig := <-quit
	slog.Info("Señal de apagado recibida, iniciando cierre elegante...", "signal", sig.String())

	// Contexto con timeout de 10 segundos para dar tiempo a las peticiones activas a finalizar
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Detenemos el servidor HTTP esperando peticiones activas
	if err := srv.Shutdown(ctxShutdown); err != nil {
		slog.Error("Forzando el apagado del servidor HTTP por timeout/error", "error", err)
	} else {
		slog.Info("Servidor HTTP detenido correctamente sin interrumpir peticiones activas")
	}

	// Cierre explícito del pool de conexiones a PostgreSQL
	slog.Info("Cerrando pool de conexiones a PostgreSQL (pgxpool)...")
	dbPool.Close()
	slog.Info("Aplicación Goallet API finalizada limpiamente")
}

// Helper para leer variables de entorno con valor por defecto
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// jsonLoggerMiddleware intercepta cada HTTP request de Gin y emite un log JSON con slog.
func jsonLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		latency := time.Since(start)
		status := c.Writer.Status()

		// Declaramos el slice como []any para que sea compatible con slog
		attrs := []any{
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.Duration("latency", latency),
		}

		if len(c.Errors) > 0 {
			// Agregamos el error al slice []any antes de enviarlo a slog
			attrs = append(attrs, slog.String("errors", c.Errors.String()))
			slog.Error("Petición HTTP fallida", attrs...)
		} else if status >= 400 {
			slog.Warn("Petición HTTP procesada con error cliente/servidor", attrs...)
		} else {
			slog.Info("Petición HTTP procesada", attrs...)
		}
	}
}
