package handlers

import (
	"net/http"
	"time"

	"goallet-api/internal/adapters/handlers/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// GenerarToken godoc
// @Summary      Generar token JWT de acceso
// @Description  Emite un token JWT con firmado HMAC-SHA256 válido por 24 horas para consumir los endpoints protegidos.
// @Tags         Autenticación
// @Accept       json
// @Produce      json
// @Success      200  {object}  map[string]string "Token generado exitosamente"
// @Failure      500  {object}  map[string]string "Error interno al firmar el token"
// @Security     BearerAuth
func (h *AuthHandler) GenerarToken(c *gin.Context) {
	claims := jwt.MapClaims{
		"sub": "admin-goallet-user",
		"exp": time.Now().Add(time.Hour * 24).Unix(), // Expira en 24 horas
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(middleware.JwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar el token de acceso"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"tipo":  "Bearer",
		"exp":   "24 horas",
	})
}
