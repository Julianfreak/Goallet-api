package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"goallet-api/internal/adapters/handlers"
	"goallet-api/internal/adapters/storage"
	"goallet-api/internal/core/services"
)

func main() {
	// 1. Instanciamos los adaptadores de salida (Persistencia)
	cuentaRepo := storage.NewMemoryCuentaStorage()
	transaccionRepo := storage.NewMemoryTransaccionStorage()

	// 2. Inyectamos los repositorios en el Servicio (Núcleo)
	billeteraService := services.NewBilleteraService(cuentaRepo, transaccionRepo)

	// 3. Inyectamos el servicio en el Adaptador de entrada (Handler HTTP)
	billeteraHandler := handlers.NewBilleteraHandler(billeteraService)

	// 4. Inicializamos el motor de Gin con middlewares por defecto (Logger y Recovery)
	router := gin.Default()

	// 5. Mapeamos las rutas de la API REST
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

	// 6. Encendemos el servidor en el puerto 8080
	log.Println("Servidor iniciado en http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}
