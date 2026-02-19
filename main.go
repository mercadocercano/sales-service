package main

import (
	"database/sql"
	"log"
	"os"

	apiConfig "sales/src/api/config"
	salesUseCase "sales/src/sales/application/usecase"
	"sales/src/sales/domain/port"
	salesCache "sales/src/sales/infrastructure/cache"
	salesClient "sales/src/sales/infrastructure/client"
	salesController "sales/src/sales/infrastructure/controller"
	salesPersistence "sales/src/sales/infrastructure/persistence"
	sharedConfig "sales/src/shared/infrastructure/config"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq" // Driver de PostgreSQL
	"github.com/prometheus/client_golang/prometheus/promhttp"
	
	"github.com/mercadocercano/eventbus"
)

// getEnv obtiene una variable de entorno o devuelve un valor por defecto
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	log.Println("🚀 Sales Service - HITO v0.2 - Iniciando...")

	// Configurar el router con Gin
	router := gin.New()

	// Agregar middlewares básicos necesarios
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Configurar Prometheus metrics si está habilitado
	prometheusEnabled := os.Getenv("PROMETHEUS_ENABLED")
	log.Printf("PROMETHEUS_ENABLED value: '%s'", prometheusEnabled)

	if prometheusEnabled == "true" {
		log.Println("Registering /metrics endpoint for Sales service")
		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
		log.Println("/metrics endpoint registered successfully for Sales service")
	} else {
		log.Println("Prometheus metrics disabled for Sales service")
	}

	// Configurar GZIP y otros middlewares compartidos
	gzipSharedCfg := sharedConfig.DefaultSharedConfig()
	sharedConfig.SetupSharedMiddleware(router, gzipSharedCfg)

	// Obtener configuración de la base de datos de variables de entorno
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "order_db")

	// Crear string de conexión para order_db
	connStr := "postgres://" + dbUser + ":" + dbPassword + "@" + dbHost + ":" + dbPort + "/" + dbName + "?sslmode=disable"
	log.Printf("Intentando conectar a order_db: %s", connStr)

	// Conectar a la base de datos (opcional para bootstrap)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("⚠️  Advertencia: Error al conectar a la base de datos: %v", err)
		log.Println("⚠️  Continuando sin DB (solo health check)")
		db = nil
	} else {
		defer db.Close()
		// Comprobar la conexión
		err = db.Ping()
		if err != nil {
			log.Printf("⚠️  Advertencia: Error al verificar la conexión a la base de datos: %v", err)
			log.Println("⚠️  Continuando sin DB (solo health check)")
			db = nil
		} else {
			log.Println("✅ Conexión a order_db establecida con éxito")
		}
	}

	// HITO: Conectar a payment_method_db para cache de métodos de pago
	pmDBName := getEnv("PAYMENT_METHOD_DB_NAME", "payment_method_db")
	pmConnStr := "postgres://" + dbUser + ":" + dbPassword + "@" + dbHost + ":" + dbPort + "/" + pmDBName + "?sslmode=disable"
	log.Printf("Intentando conectar a payment_method_db: %s", pmConnStr)

	var paymentMethodDB *sql.DB
	paymentMethodDB, err = sql.Open("postgres", pmConnStr)
	if err != nil {
		log.Printf("⚠️  Advertencia: Error al conectar a payment_method_db: %v", err)
		log.Println("⚠️  Continuando sin payment method cache")
		paymentMethodDB = nil
	} else {
		defer paymentMethodDB.Close()
		err = paymentMethodDB.Ping()
		if err != nil {
			log.Printf("⚠️  Advertencia: Error al verificar conexión a payment_method_db: %v", err)
			log.Println("⚠️  Continuando sin payment method cache")
			paymentMethodDB = nil
		} else {
			log.Println("✅ Conexión a payment_method_db establecida con éxito")
		}
	}

	// HITO v0.1: Conectar a EventBus DB
	eventBusHost := getEnv("EVENTBUS_DB_HOST", dbHost)
	eventBusPort := getEnv("EVENTBUS_DB_PORT", "5432")
	eventBusUser := getEnv("EVENTBUS_DB_USER", dbUser)
	eventBusPassword := getEnv("EVENTBUS_DB_PASSWORD", dbPassword)
	eventBusName := getEnv("EVENTBUS_DB_NAME", "eventbus")
	
	eventBusConnStr := "postgres://" + eventBusUser + ":" + eventBusPassword + "@" + eventBusHost + ":" + eventBusPort + "/" + eventBusName + "?sslmode=disable"
	log.Printf("Intentando conectar a eventbus: %s", eventBusConnStr)
	
	var eventBusDB *sql.DB
	var publishUseCase *eventbus.PublishEventUseCase
	
	eventBusDB, err = sql.Open("postgres", eventBusConnStr)
	if err != nil {
		log.Printf("⚠️  Advertencia: Error al conectar a eventbus: %v", err)
		log.Println("⚠️  Continuando sin publicación de eventos")
		publishUseCase = nil
	} else {
		err = eventBusDB.Ping()
		if err != nil {
			log.Printf("⚠️  Advertencia: Error al verificar conexión a eventbus: %v", err)
			log.Println("⚠️  Continuando sin publicación de eventos")
			publishUseCase = nil
		} else {
			log.Println("✅ Conexión a eventbus establecida con éxito")
			
			// Inicializar eventbus publisher
			logger := eventbus.NewLogger(eventbus.LevelInfo)
			eventStore := eventbus.NewSQLEventStore(eventBusDB, logger)
			publishUseCase = eventbus.NewPublishEventUseCase(eventStore, logger)
			
			if eventBusDB != nil {
				defer eventBusDB.Close()
			}
		}
	}

	// API v1 grupo de rutas
	v1 := router.Group("/api/v1")

	// Configurar el módulo API (health check y documentación)
	apiCfg := apiConfig.DefaultAPIConfig()
	apiCfg.DB = db
	apiCfg.Version = "1.0.0-bootstrap"
	apiConfig.SetupAPIModule(router, v1, apiCfg)

	// Configurar módulo Sales (con eventbus)
	setupSalesModule(v1, db, paymentMethodDB, publishUseCase)

	// Iniciar el servidor
	port := getEnv("PORT", "8080")
	log.Printf("✅ Servidor Sales Service iniciado en http://localhost:%s", port)
	log.Printf("✅ Health endpoint: GET http://localhost:%s/health", port)
	log.Printf("✅ Health endpoint: GET http://localhost:%s/api/v1/health", port)
	router.Run(":" + port)
}

// setupSalesModule configura el módulo Sales
func setupSalesModule(router *gin.RouterGroup, db *sql.DB, paymentMethodDB *sql.DB, publishUseCase *eventbus.PublishEventUseCase) {
	log.Println("Configurando módulo Sales...")

	// Crear cliente de stock-service
	stockClient := salesClient.NewStockClient()

	// Crear cliente de pim-service (para snapshots)
	pimClient := salesClient.NewPIMClient()

	// HITO: Inicializar cache de payment methods
	var pmCache *salesCache.PaymentMethodCache
	if paymentMethodDB != nil {
		pmCache = salesCache.NewPaymentMethodCache()
		err := pmCache.LoadFromDB(paymentMethodDB)
		if err != nil {
			log.Printf("⚠️  Warning: Could not load payment methods cache: %v", err)
			pmCache = nil
		}
	} else {
		log.Println("⚠️  Payment method cache disabled (no DB connection)")
	}

	// Crear repositorios
	var salesRepo *salesPersistence.OrderPostgresRepository
	var posSaleRepo port.PosSaleRepository
	if db != nil {
		salesRepo = salesPersistence.NewOrderPostgresRepository(db)
		posSaleRepo = salesPersistence.NewPosSalePostgresRepository(db)
	}

	// Crear casos de uso
	validateStockUC := salesUseCase.NewValidateStockUseCase(stockClient)
	reserveStockUC := salesUseCase.NewReserveStockUseCase(stockClient)
	releaseStockUC := salesUseCase.NewReleaseStockUseCase(stockClient)
	
	// POS Sale UseCase - ahora con repo, cache y eventbus
	var posSaleUC *salesUseCase.POSSaleUseCase
	var listPosSalesUC *salesUseCase.ListPosSalesUseCase
	if posSaleRepo != nil {
		posSaleUC = salesUseCase.NewPOSSaleUseCase(stockClient, posSaleRepo, pmCache, publishUseCase)
		listPosSalesUC = salesUseCase.NewListPosSalesUseCase(posSaleRepo)
	} else {
		// Fallback sin repo (solo para desarrollo sin DB)
		posSaleUC = salesUseCase.NewPOSSaleUseCase(stockClient, nil, pmCache, publishUseCase)
	}

	var createOrderUC *salesUseCase.CreateOrderUseCase
	var confirmOrderUC *salesUseCase.ConfirmOrderUseCase
	var cancelOrderUC *salesUseCase.CancelOrderUseCase
	var listOrdersUC *salesUseCase.ListOrdersUseCase
	var getOrderUC *salesUseCase.GetOrderUseCase
	if salesRepo != nil {
		createOrderUC = salesUseCase.NewCreateOrderUseCase(salesRepo, pimClient, stockClient)
		confirmOrderUC = salesUseCase.NewConfirmOrderUseCase(salesRepo, stockClient, publishUseCase)
		cancelOrderUC = salesUseCase.NewCancelOrderUseCase(salesRepo, stockClient)
		listOrdersUC = salesUseCase.NewListOrdersUseCase(salesRepo)
		getOrderUC = salesUseCase.NewGetOrderUseCase(salesRepo)
	}

	// Crear controladores
	salesCtrl := salesController.NewOrderController(validateStockUC, reserveStockUC, releaseStockUC, createOrderUC, confirmOrderUC, cancelOrderUC, listOrdersUC, getOrderUC, posSaleUC, listPosSalesUC)

	// HITO C - Report Controller
	dailyReportUC := salesUseCase.NewDailyReportUseCase(db)
	reportCtrl := salesController.NewReportController(dailyReportUC)

	// Registrar rutas
	salesCtrl.RegisterRoutes(router)
	reportCtrl.RegisterRoutes(router)

	log.Println("Módulo Sales configurado exitosamente")
}
