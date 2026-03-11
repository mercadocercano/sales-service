package config

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIConfig contiene la configuración para el módulo API
type APIConfig struct {
	DB      *sql.DB
	Version string
}

// DefaultAPIConfig devuelve una configuración por defecto para el módulo API
func DefaultAPIConfig() *APIConfig {
	return &APIConfig{
		Version: "1.0.0",
	}
}

// SetupAPIModule configura el módulo API
func SetupAPIModule(router *gin.Engine, apiGroup *gin.RouterGroup, config *APIConfig) {
	// Ruta de salud (health check)
	router.GET("/health", func(c *gin.Context) {
		status := "ok"
		dbStatus := "ok"

		// Verificar conexión a la base de datos si está configurada
		if config.DB != nil {
			err := config.DB.Ping()
			if err != nil {
				dbStatus = "error: " + err.Error()
				status = "error"
			}
		} else {
			dbStatus = "not configured"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   status,
			"service":  "sales",
			"version":  config.Version,
			"database": dbStatus,
		})
	})

	// Información sobre la API
	apiGroup.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "Sales Management Service",
			"version":     config.Version,
			"description": "Servicio para la gestión de órdenes y ventas POS",
		})
	})

	// Documentación de la API (si está disponible)
	router.GET("/api-docs", func(c *gin.Context) {
		c.HTML(http.StatusOK, "swagger.html", gin.H{
			"title": "Sales API Documentation",
		})
	})
}
