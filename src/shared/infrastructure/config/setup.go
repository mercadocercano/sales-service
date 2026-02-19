package config

import (
	"sales/src/shared/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

// GzipSharedConfig contiene la configuración para el módulo compartido de compresión
type GzipSharedConfig struct {
	EnableGzip            bool
	AlwaysTryDecompress   bool
	ForceGzipCompression  bool
	ForceGzipCheckSupport bool     // Verifica si el cliente soporta gzip antes de forzar compresión
	ForceGzipPaths        []string // Rutas donde forzar compresión
	GzipExcludedPaths     []string
}

// DefaultSharedConfig devuelve una configuración por defecto
func DefaultSharedConfig() GzipSharedConfig {
	return GzipSharedConfig{
		EnableGzip:            true,
		AlwaysTryDecompress:   true,
		ForceGzipCompression:  false,
		ForceGzipCheckSupport: true,
		ForceGzipPaths:        []string{"/order/api/v1/warehouses"},
		GzipExcludedPaths:     []string{"/health", "/metrics", "/api-docs", "/order/api/v1/locations"},
	}
}

// SetupSharedMiddleware configura los middlewares compartidos
func SetupSharedMiddleware(router *gin.Engine, config GzipSharedConfig) {
	// Aplicar middleware para intentar descomprimir todas las solicitudes entrantes si está habilitado
	if config.AlwaysTryDecompress {
		router.Use(middleware.GzipReader())
	}

	// Aplicar middleware de compresión gzip si está habilitado
	if config.EnableGzip {
		gzipOpts := middleware.GzipOptions{
			ExcludedPaths: config.GzipExcludedPaths,
		}
		router.Use(middleware.GzipMiddleware(gzipOpts))

		// Configurar rutas que siempre deben usar compresión gzip
		if config.ForceGzipCompression && len(config.ForceGzipPaths) > 0 {
			forceGzipOpts := middleware.ForceGzipOptions{
				CheckClientSupport: config.ForceGzipCheckSupport,
			}

			// Ejemplo de cómo aplicar compresión forzada a rutas específicas
			for _, path := range config.ForceGzipPaths {
				router.Group(path).Use(middleware.ForceGzipMiddleware(forceGzipOpts))
			}
		}
	}

	// Aquí se pueden agregar más middlewares compartidos en el futuro
	// Por ejemplo:
	// - Logging
	// - CORS
	// - Medición de rendimiento
	// - Autenticación/Autorización
}
