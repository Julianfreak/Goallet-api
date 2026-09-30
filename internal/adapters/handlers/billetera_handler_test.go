package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"goallet-api/internal/adapters/handlers"
	"goallet-api/internal/core/domain"
	"goallet-api/internal/core/services"

	"github.com/gin-gonic/gin"
)

// ==========================================================
// 1. MOCKS COMPLETOS DE PERSISTENCIA
// ==========================================================

type MockCuentaRepo struct {
	cuentas               map[string]*domain.Cuenta
	forzarErrorGuardar    error
	forzarErrorBuscar     error
	forzarErrorActualizar error
	forzarErrorListar     error
	forzarErrorEliminar   error
}

func NewMockCuentaRepo() *MockCuentaRepo {
	return &MockCuentaRepo{cuentas: make(map[string]*domain.Cuenta)}
}

func (m *MockCuentaRepo) Guardar(c *domain.Cuenta) error {
	if m.forzarErrorGuardar != nil {
		return m.forzarErrorGuardar
	}
	m.cuentas[c.ID] = c
	return nil
}

func (m *MockCuentaRepo) BuscarPorID(id string) (*domain.Cuenta, error) {
	if m.forzarErrorBuscar != nil {
		return nil, m.forzarErrorBuscar
	}
	c, existe := m.cuentas[id]
	if !existe {
		return nil, domain.ErrCuentaNoEncontrada
	}
	return c, nil
}

func (m *MockCuentaRepo) Actualizar(c *domain.Cuenta) error {
	if m.forzarErrorActualizar != nil {
		return m.forzarErrorActualizar
	}
	m.cuentas[c.ID] = c
	return nil
}

func (m *MockCuentaRepo) Listar() ([]*domain.Cuenta, error) {
	if m.forzarErrorListar != nil {
		return nil, m.forzarErrorListar
	}
	var lista []*domain.Cuenta
	for _, c := range m.cuentas {
		lista = append(lista, c)
	}
	return lista, nil
}

func (m *MockCuentaRepo) Eliminar(id string) error {
	if m.forzarErrorEliminar != nil {
		return m.forzarErrorEliminar
	}
	delete(m.cuentas, id)
	return nil
}

type MockTxRepo struct {
	forzarErrorGuardar error
}

func (m *MockTxRepo) Guardar(tx *domain.Transaccion) error {
	if m.forzarErrorGuardar != nil {
		return m.forzarErrorGuardar
	}
	return nil
}

func (m *MockTxRepo) ListarPorCuentaID(id string) ([]domain.Transaccion, error) {
	return nil, nil
}

// ==========================================================
// 2. HELPER CON TODAS LAS RUTAS REGISTRADAS
// ==========================================================

func setupTestRouter(repoCuenta *MockCuentaRepo, mockTx *MockTxRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)

	if mockTx == nil {
		mockTx = &MockTxRepo{}
	}

	service := services.NewBilleteraService(repoCuenta, mockTx)
	handler := handlers.NewBilleteraHandler(service)

	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.POST("/cuentas", handler.CrearCuenta)
		v1.GET("/cuentas", handler.ListarCuentas)
		v1.GET("/cuentas/:id", handler.ConsultarCuenta)
		v1.PUT("/cuentas/:id", handler.ActualizarTitular)
		v1.DELETE("/cuentas/:id", handler.EliminarCuenta)
		v1.POST("/cuentas/:id/depositar", handler.Depositar)
		v1.POST("/cuentas/:id/retirar", handler.Retirar)
		v1.POST("/cuentas/:id/transferir", handler.Transferir)
	}

	return router
}

// Helper para realizar peticiones HTTP sintéticas
func ejecutarRequest(router *gin.Engine, metodo, url string, body interface{}) *httptest.ResponseRecorder {
	var bodyBuffer *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyBuffer = bytes.NewBuffer(jsonBytes)
	} else {
		bodyBuffer = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(metodo, url, bodyBuffer)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ==========================================================
// 3. PRUEBAS COMPLETAS DE COBERTURA DE HANDLERS
// ==========================================================

func TestHandler_ListarCuentas(t *testing.T) {
	t.Run("200 OK - Exitoso", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Titular: "Ana"}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodGet, "/api/v1/cuentas", nil)
		if w.Code != http.StatusOK {
			t.Errorf("Esperado 200 OK, obtenido %d", w.Code)
		}
	})

	t.Run("500 Internal Server Error", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.forzarErrorListar = errors.New("falla interna en repositorio")
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodGet, "/api/v1/cuentas", nil)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Esperado 500, obtenido %d", w.Code)
		}
	})
}

func TestHandler_Depositar(t *testing.T) {
	t.Run("200 OK - Depósito exitoso", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 100.0}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/depositar", map[string]interface{}{"monto": 50.0})
		if w.Code != http.StatusOK {
			t.Errorf("Esperado 200 OK, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - Body JSON malformado", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/cuentas/cta-1/depositar", bytes.NewBufferString("{monto: malformado"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400 Bad Request, obtenido %d", w.Code)
		}
	})

	t.Run("404 Not Found - Cuenta inexistente", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/fantasma/depositar", map[string]interface{}{"monto": 50.0})
		if w.Code != http.StatusNotFound {
			t.Errorf("Esperado 404 Not Found, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - Monto inválido", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 100.0}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/depositar", map[string]interface{}{"monto": -10.0})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400 Bad Request, obtenido %d", w.Code)
		}
	})

	t.Run("500 Internal Server Error - Falla base de datos", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 100.0}
		repo.forzarErrorActualizar = errors.New("db caida")
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/depositar", map[string]interface{}{"monto": 50.0})
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Esperado 500, obtenido %d", w.Code)
		}
	})
}

func TestHandler_Retirar(t *testing.T) {
	t.Run("200 OK - Retiro exitoso", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 100.0}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/retirar", map[string]interface{}{"monto": 40.0})
		if w.Code != http.StatusOK {
			t.Errorf("Esperado 200 OK, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - JSON malformado", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/cuentas/cta-1/retirar", bytes.NewBufferString("{bad-json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400, obtenido %d", w.Code)
		}
	})

	t.Run("404 Not Found - Cuenta no encontrada", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/fantasma/retirar", map[string]interface{}{"monto": 20.0})
		if w.Code != http.StatusNotFound {
			t.Errorf("Esperado 404, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - Saldo insuficiente o monto invalido", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 10.0}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/retirar", map[string]interface{}{"monto": 500.0})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400 Bad Request por saldo insuficiente, obtenido %d", w.Code)
		}
	})

	t.Run("500 Internal Server Error", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 100.0}
		repo.forzarErrorActualizar = errors.New("db error")
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/retirar", map[string]interface{}{"monto": 20.0})
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Esperado 500, obtenido %d", w.Code)
		}
	})
}

func TestHandler_Transferir(t *testing.T) {
	t.Run("200 OK - Transferencia exitosa", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 300.0}
		repo.cuentas["cta-2"] = &domain.Cuenta{ID: "cta-2", Saldo: 100.0}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/transferir", map[string]interface{}{
			"cuenta_destino_id": "cta-2",
			"monto":             50.0,
		})
		if w.Code != http.StatusOK {
			t.Errorf("Esperado 200 OK, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - Body inválido", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/cuentas/cta-1/transferir", bytes.NewBufferString("{bad}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400, obtenido %d", w.Code)
		}
	})

	t.Run("404 Not Found - Cuenta origen inexistente", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/fantasma/transferir", map[string]interface{}{
			"cuenta_destino_id": "cta-2",
			"monto":             50.0,
		})
		if w.Code != http.StatusNotFound {
			t.Errorf("Esperado 404, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - Transferir a la misma cuenta o sin saldo", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 300.0}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/transferir", map[string]interface{}{
			"cuenta_destino_id": "cta-1",
			"monto":             50.0,
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400 Bad Request por misma cuenta destino, obtenido %d", w.Code)
		}
	})

	t.Run("500 Internal Server Error", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Saldo: 300.0}
		repo.cuentas["cta-2"] = &domain.Cuenta{ID: "cta-2", Saldo: 100.0}
		repo.forzarErrorActualizar = errors.New("db error")
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPost, "/api/v1/cuentas/cta-1/transferir", map[string]interface{}{
			"cuenta_destino_id": "cta-2",
			"monto":             50.0,
		})
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Esperado 500, obtenido %d", w.Code)
		}
	})
}

func TestHandler_ActualizarTitular(t *testing.T) {
	t.Run("200 OK - Actualización exitosa", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Titular: "Viejo"}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPut, "/api/v1/cuentas/cta-1", map[string]interface{}{
			"nuevo_titular": "Nuevo Titular",
		})
		if w.Code != http.StatusOK {
			t.Errorf("Esperado 200 OK, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - Body inválido o titular vacío", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/cuentas/cta-1", bytes.NewBufferString("{bad}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400, obtenido %d", w.Code)
		}
	})

	t.Run("404 Not Found - Cuenta no encontrada", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPut, "/api/v1/cuentas/fantasma", map[string]interface{}{
			"nuevo_titular": "Nuevo Titular",
		})
		if w.Code != http.StatusNotFound {
			t.Errorf("Esperado 404, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - Regla de dominio (nombre vacío)", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-1"] = &domain.Cuenta{ID: "cta-1", Titular: "Viejo"}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodPut, "/api/v1/cuentas/cta-1", map[string]interface{}{
			"nuevo_titular": "",
		})
		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400 Bad Request, obtenido %d", w.Code)
		}
	})
}

func TestHandler_EliminarCuenta(t *testing.T) {
	t.Run("200 OK - Eliminación exitosa", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-vacia"] = &domain.Cuenta{ID: "cta-vacia", Saldo: 0.0}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodDelete, "/api/v1/cuentas/cta-vacia", nil)
		if w.Code != http.StatusOK {
			t.Errorf("Esperado 200 OK, obtenido %d", w.Code)
		}
	})

	t.Run("404 Not Found - Cuenta inexistente", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodDelete, "/api/v1/cuentas/fantasma", nil)
		if w.Code != http.StatusNotFound {
			t.Errorf("Esperado 404 Not Found, obtenido %d", w.Code)
		}
	})

	t.Run("400 Bad Request - Intento de eliminar cuenta con saldo", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-plata"] = &domain.Cuenta{ID: "cta-plata", Saldo: 100.0}
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodDelete, "/api/v1/cuentas/cta-plata", nil)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Esperado 400 Bad Request, obtenido %d", w.Code)
		}
	})

	t.Run("500 Internal Server Error", func(t *testing.T) {
		repo := NewMockCuentaRepo()
		repo.cuentas["cta-vacia"] = &domain.Cuenta{ID: "cta-vacia", Saldo: 0.0}
		repo.forzarErrorEliminar = errors.New("falla al borrar en db")
		router := setupTestRouter(repo, nil)

		w := ejecutarRequest(router, http.MethodDelete, "/api/v1/cuentas/cta-vacia", nil)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Esperado 500, obtenido %d", w.Code)
		}
	})
}
