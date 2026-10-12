package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "goallet-api/docs"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"goallet-api/internal/adapters/handlers"
	"goallet-api/internal/adapters/storage"
	"goallet-api/internal/core/services"
)

// @title           Goallet API
// @version         1.0
// @description     Motor backend transaccional para billetera digital de alta concurrencia.
// @termsOfService  http://swagger.io/terms/

// @contact.name   Julian Serna Saavedra
// @contact.url    https://github.com/Julianfreak/Goallet-api

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

func main() {
	fmt.Println("==========================================================")
	fmt.Println("   Goallet API - Motor Transaccional de Billetera Digital  ")
	fmt.Println("==========================================================")

	ctx := context.Background()

	// 1. Configuración de Logs JSON (slog) hacia Consola y app.log
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("No se pudo abrir app.log: %v", err)
	}
	defer file.Close()

	multiWriter := io.MultiWriter(os.Stdout, file)
	logger := slog.New(slog.NewJSONHandler(multiWriter, nil))
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

	// Inyectamos ambos repositorios en el Servicio (Núcleo)
	billeteraService := services.NewBilleteraService(cuentaRepo, transaccionRepo)

	// 5. Inyectamos el servicio en el Adaptador de entrada (Handler HTTP)
	billeteraHandler := handlers.NewBilleteraHandler(billeteraService)

	// 6. Inicializamos Gin sin el logger por defecto (usaremos slog en formato JSON)
	router := gin.New()
	router.Use(gin.Recovery())         // Captura panics de forma segura
	router.Use(jsonLoggerMiddleware()) // Middleware de registro estructurado JSON

	// --- RUTA INTERACTIVA SWAGGER UI ---
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 7. Mapeamos las rutas de la API REST
	api := router.Group("/api/v1")
	{
		// --- CRUD DE CUENTAS ---
		api.POST("/cuentas", billeteraHandler.CrearCuenta)          // Create
		api.GET("/cuentas", billeteraHandler.ListarCuentas)         // Read (todas)
		api.GET("/cuentas/:id", billeteraHandler.ConsultarCuenta)   // Read (por ID)
		api.PUT("/cuentas/:id", billeteraHandler.ActualizarTitular) // Update (solo titular)
		api.DELETE("/cuentas/:id", billeteraHandler.EliminarCuenta) // Delete (solo si saldo == 0)

		// --- OPERACIONES TRANSACCIONALES ---
		api.POST("/cuentas/:id/depositar", billeteraHandler.Depositar)
		api.POST("/cuentas/:id/retirar", billeteraHandler.Retirar)
		api.POST("/cuentas/:id/transferir", billeteraHandler.Transferir)
	}

	// 8. Encendemos el servidor HTTP principal en el puerto 8080
	slog.Info("Servidor HTTP iniciado exitosamente", "url", "http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		slog.Error("Error al iniciar el servidor Gin", "error", err)
		os.Exit(1)
	}
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
