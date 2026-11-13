package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"goallet-api/internal/adapters/handlers"

	"github.com/gin-gonic/gin"
)

// TestHealthCheck_LivenessSinBD verifica el liveness probe básico
// cuando no se inyecta el pool de base de datos (dbPool = nil).
// Debe responder 200 OK con status "UP" y sin el campo database.
func TestHealthCheck_LivenessSinBD(t *testing.T) {
	gin.SetMode(gin.TestMode)

	healthHandler := handlers.NewHealthHandler(nil) // Sin pool de BD

	r := gin.New()
	r.GET("/health", healthHandler.HealthCheck)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// Validar código HTTP 200
	if w.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtenido %d", w.Code)
	}

	// Validar estructura JSON de respuesta
	var response handlers.HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("error al deserializar respuesta: %v", err)
	}

	if response.Status != "UP" {
		t.Errorf("esperado status \"UP\", obtenido %q", response.Status)
	}

	if response.Service != "goallet-api" {
		t.Errorf("esperado service \"goallet-api\", obtenido %q", response.Service)
	}

	if response.Timestamp == "" {
		t.Error("timestamp no debería estar vacío")
	}

	// Sin pool de BD, el campo database no debe estar presente
	if response.Database != "" {
		t.Errorf("esperado database vacío (sin pool), obtenido %q", response.Database)
	}

	// Validar Content-Type
	if ct := w.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("esperado Content-Type \"application/json; charset=utf-8\", obtenido %q", ct)
	}
}

// TestHealthCheck_EstructuraJSON verifica que el payload JSON
// contenga exactamente los campos esperados con las claves correctas.
func TestHealthCheck_EstructuraJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	healthHandler := handlers.NewHealthHandler(nil)

	r := gin.New()
	r.GET("/health", healthHandler.HealthCheck)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var raw map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("error al deserializar respuesta: %v", err)
	}

	clavesEsperadas := []string{"status", "service", "timestamp"}
	for _, clave := range clavesEsperadas {
		if _, ok := raw[clave]; !ok {
			t.Errorf("clave esperada %q no encontrada en la respuesta JSON", clave)
		}
	}
}
