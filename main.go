package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	apiConfig "sales/src/api/config"
	salesService "sales/src/sales/application/service"
	salesUseCase "sales/src/sales/application/usecase"
	salesReports "sales/src/sales/application/usecase/reports"
	"sales/src/sales/domain/port"
	salesAdapter "sales/src/sales/infrastructure/adapter"
	salesCache "sales/src/sales/infrastructure/cache"
	salesClient "sales/src/sales/infrastructure/client"
	salesController "sales/src/sales/infrastructure/controller"
	salesLogging "sales/src/sales/infrastructure/logging" // ADR-001: logger canónico de eventos
	_ "sales/src/sales/infrastructure/metrics"            // H8: Registra pos_sales_*, pos_sale_latency_ms, ttfs_seconds
	salesPersistence "sales/src/sales/infrastructure/persistence"
	sharedConfig "sales/src/shared/infrastructure/config"

	"github.com/gin-gonic/gin"
	"github.com/hornosg/go-shared/infrastructure/env"
	tenantmw "github.com/hornosg/go-shared/infrastructure/middleware"
	"github.com/hornosg/go-shared/infrastructure/postgres"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mercadocercano/eventbus"
)

// connectWithRetry intenta conectar a PostgreSQL con backoff exponencial,
// usando el helper compartido go-shared postgres.Connect (DSN + open + Ping +
// pool defaults). Retorna nil si no logra conectar después de todos los
// reintentos. En la conexión exitosa arranca el monitor de saturación del pool.
func connectWithRetry(cfg postgres.Config, label string, maxRetries int) *sql.DB {
	backoff := 2 * time.Second
	const maxBackoff = 30 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		db, err := postgres.Connect(cfg)
		if err != nil {
			log.Printf("⚠️  [%s] Intento %d/%d - Error al conectar: %v", label, attempt, maxRetries, err)
		} else {
			log.Printf("✅ Conexión a %s establecida con éxito (intento %d)", label, attempt)
			postgres.StartPoolMonitor(context.Background(), db, postgres.MonitorOptions{
				Service: "sales-service",
				DBName:  cfg.DBName,
			})
			return db
		}

		if attempt < maxRetries {
			log.Printf("🔄 [%s] Reintentando en %v...", label, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	log.Printf("❌ [%s] No se pudo conectar después de %d intentos. Continuando sin DB.", label, maxRetries)
	return nil
}

func main() {
	log.Println("🚀 Sales Service - HITO v0.5 - HTTP Routes Alignment - Iniciando...")

	// Configurar el router con Gin
	router := gin.New()

	// Agregar middlewares básicos necesarios
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	salesNamespace := os.Getenv("SERVICE_NAMESPACE")
	if salesNamespace == "" {
		salesNamespace = "mc"
	}
	router.Use(tenantmw.TenantValidation(tenantmw.TenantValidationConfig{
		JWTSecret: os.Getenv("JWT_SECRET"),
		// sales maneja dinero (caja/POS): se valida namespace y se CIERRA el bypass de
		// tenant (sin tenant_id ⇒ 403). Prerequisito de seguridad de la caja (ADR-003).
		Namespace:           salesNamespace,
		RejectMissingTenant: true,
		ExcludedRoutes: []string{
			"/health",
			"/api/v1/health",
			"/metrics",
		},
	}))

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
	dbHost := env.Get("DB_HOST", "localhost")
	dbPort := env.Get("DB_PORT", "5432")
	dbUser := env.Get("DB_USER", "postgres")
	dbPassword := env.Get("DB_PASSWORD", "postgres")
	dbName := env.Get("DB_NAME", "order_db")

	// Máximo de reintentos para conexiones DB (cubre ~60s de espera con backoff exponencial)
	maxRetries := 10

	// Conectar a order_db con retry
	log.Printf("Intentando conectar a order_db (host=%s port=%s db=%s)", dbHost, dbPort, dbName)
	db := connectWithRetry(postgres.Config{
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPassword,
		DBName:   dbName,
		SSLMode:  "disable",
	}, "order_db", maxRetries)
	if db != nil {
		defer db.Close()
	}

	// Conectar a payment_method_db con retry
	pmDBName := env.Get("PAYMENT_METHOD_DB_NAME", "payment_method_db")
	log.Printf("Intentando conectar a payment_method_db (host=%s port=%s db=%s)", dbHost, dbPort, pmDBName)
	paymentMethodDB := connectWithRetry(postgres.Config{
		Host:     dbHost,
		Port:     dbPort,
		User:     dbUser,
		Password: dbPassword,
		DBName:   pmDBName,
		SSLMode:  "disable",
	}, "payment_method_db", maxRetries)
	if paymentMethodDB != nil {
		defer paymentMethodDB.Close()
	}

	// Conectar a EventBus DB con retry
	eventBusHost := env.Get("EVENTBUS_DB_HOST", dbHost)
	eventBusPort := env.Get("EVENTBUS_DB_PORT", "5432")
	eventBusUser := env.Get("EVENTBUS_DB_USER", dbUser)
	eventBusPassword := env.Get("EVENTBUS_DB_PASSWORD", dbPassword)
	eventBusName := env.Get("EVENTBUS_DB_NAME", "eventbus")

	log.Printf("Intentando conectar a eventbus (host=%s port=%s db=%s)", eventBusHost, eventBusPort, eventBusName)
	eventBusDB := connectWithRetry(postgres.Config{
		Host:     eventBusHost,
		Port:     eventBusPort,
		User:     eventBusUser,
		Password: eventBusPassword,
		DBName:   eventBusName,
		SSLMode:  "disable",
	}, "eventbus", maxRetries)

	var publishUseCase *eventbus.PublishEventUseCase
	if eventBusDB != nil {
		defer eventBusDB.Close()
		logger := eventbus.NewLogger(eventbus.LevelInfo)
		eventStore := eventbus.NewSQLEventStore(eventBusDB, logger)
		publishUseCase = eventbus.NewPublishEventUseCase(eventStore, logger)
	}

	// API v1 grupo de rutas
	v1 := router.Group("/api/v1")

	// Configurar el módulo API (health check y documentación)
	apiCfg := apiConfig.DefaultAPIConfig()
	apiCfg.DB = db
	apiCfg.Version = "0.5.0-routes-alignment"
	apiConfig.SetupAPIModule(router, v1, apiCfg)

	// Configurar módulo Sales (con eventbus)
	setupSalesModule(v1, db, paymentMethodDB, publishUseCase)

	// Iniciar el servidor
	port := env.Get("PORT", "8080")
	log.Printf("✅ Servidor Sales Service iniciado en http://localhost:%s", port)
	log.Printf("✅ Health endpoint: GET http://localhost:%s/health", port)
	log.Printf("✅ Health endpoint: GET http://localhost:%s/api/v1/health", port)
	router.Run(":" + port)
}

// setupSalesModule configura el módulo Sales
func setupSalesModule(router *gin.RouterGroup, db *sql.DB, paymentMethodDB *sql.DB, publishUseCase *eventbus.PublishEventUseCase) {
	log.Println("Configurando módulo Sales...")

	// HITO v0.4: Crear servicio de secuencias
	var sequenceService *salesService.SequenceService
	if db != nil {
		sequenceService = salesService.NewSequenceService(db)
		log.Println("✅ Sequence service inicializado")
	}

	// Crear cliente de stock-service
	stockClient := salesClient.NewStockClient()

	// Crear cliente de pim-service (para snapshots)
	pimClient := salesClient.NewPIMClient()

	// HITO v0.6: Crear cliente de customer-service
	// H8: Crear cliente de tenant (IAM) para TTFS
	kongBaseURL := env.Get("KONG_URL", env.Get("KONG_INTERNAL_URL", "http://localhost:8001"))
	customerClient := salesClient.NewCustomerClient(kongBaseURL)
	tenantClient := salesClient.NewTenantClient(kongBaseURL)

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
	var tenantMetricsRepo port.TenantMetricsRepository
	var reportRepo port.ReportRepository
	if db != nil {
		salesRepo = salesPersistence.NewOrderPostgresRepository(db)
		posSaleRepo = salesPersistence.NewPosSalePostgresRepository(db)
		tenantMetricsRepo = salesPersistence.NewTenantMetricsPostgresRepository(db)
		reportRepo = salesPersistence.NewReportPostgresRepository(db)
	}

	// Crear adaptadores de puertos
	stockAdapter := salesAdapter.NewStockAdapter(stockClient)
	pimAdapter := salesAdapter.NewPIMAdapter(pimClient)
	customerAdapter := salesAdapter.NewCustomerAdapter(customerClient)
	tenantAdapter := salesAdapter.NewTenantAdapter(tenantClient)
	var paymentMethodAdapter port.PaymentMethodPort
	if pmCache != nil {
		paymentMethodAdapter = salesAdapter.NewPaymentMethodAdapter(pmCache)
	}

	// Crear casos de uso
	validateStockUC := salesUseCase.NewValidateStockUseCase(stockAdapter)
	reserveStockUC := salesUseCase.NewReserveStockUseCase(stockAdapter)
	releaseStockUC := salesUseCase.NewReleaseStockUseCase(stockAdapter)

	// ADR-001: logger canónico de eventos de ventas (una línea JSON por evento a stdout)
	salesEventLogger := salesLogging.NewSalesLogger("sales")

	// POS Sale UseCase
	var posSaleUC *salesUseCase.POSSaleUseCase
	var listPosSalesUC *salesUseCase.ListPosSalesUseCase
	var getPosSaleUC *salesUseCase.GetPosSaleUseCase
	var generatePosSalePdfUC *salesUseCase.GeneratePosSalePdfUseCase
	if posSaleRepo != nil {
		posSaleUC = salesUseCase.NewPOSSaleUseCase(stockAdapter, posSaleRepo, tenantMetricsRepo, paymentMethodAdapter, publishUseCase, customerAdapter, tenantAdapter, salesEventLogger)
		listPosSalesUC = salesUseCase.NewListPosSalesUseCase(posSaleRepo)
		getPosSaleUC = salesUseCase.NewGetPosSaleUseCase(posSaleRepo, paymentMethodAdapter, customerAdapter) // E18 Tramo B
		// E18 Tramo C: comprobante PDF A4 (reusa el detalle + nombre del comercio)
		posReceiptRenderer := salesAdapter.NewGofpdfPosReceiptRenderer()
		generatePosSalePdfUC = salesUseCase.NewGeneratePosSalePdfUseCase(getPosSaleUC, tenantAdapter, posReceiptRenderer)
	} else {
		posSaleUC = salesUseCase.NewPOSSaleUseCase(stockAdapter, nil, nil, paymentMethodAdapter, publishUseCase, customerAdapter, tenantAdapter, salesEventLogger)
	}

	var createOrderUC *salesUseCase.CreateOrderUseCase
	var confirmOrderUC *salesUseCase.ConfirmOrderUseCase
	var cancelOrderUC *salesUseCase.CancelOrderUseCase
	var listOrdersUC *salesUseCase.ListOrdersUseCase
	var getOrderUC *salesUseCase.GetOrderUseCase
	var getOrderFinancialUC *salesUseCase.GetOrderFinancialUseCase
	var registerPaymentUC *salesUseCase.RegisterPaymentUseCase
	if salesRepo != nil {
		createOrderUC = salesUseCase.NewCreateOrderUseCase(salesRepo, pimAdapter, stockAdapter, customerAdapter)
		confirmOrderUC = salesUseCase.NewConfirmOrderUseCase(salesRepo, stockAdapter, publishUseCase, sequenceService)
		cancelOrderUC = salesUseCase.NewCancelOrderUseCase(salesRepo, stockAdapter)
		listOrdersUC = salesUseCase.NewListOrdersUseCase(salesRepo)
		getOrderUC = salesUseCase.NewGetOrderUseCase(salesRepo)
		getOrderFinancialUC = salesUseCase.NewGetOrderFinancialUseCase(salesRepo)
		paymentRepo := salesPersistence.NewPaymentPostgresRepository(db)
		registerPaymentUC = salesUseCase.NewRegisterPaymentUseCase(paymentRepo, publishUseCase)
	}

	var applyCustomerCreditUC *salesUseCase.ApplyCustomerCreditUseCase
	var createCustomerCreditUC *salesUseCase.CreateCustomerCreditUseCase
	if db != nil && publishUseCase != nil {
		creditRepo := salesPersistence.NewCreditPostgresRepository(db)
		applyCustomerCreditUC = salesUseCase.NewApplyCustomerCreditUseCase(creditRepo, publishUseCase)
		createCustomerCreditUC = salesUseCase.NewCreateCustomerCreditUseCase(creditRepo, publishUseCase)
	}

	// E18 Tramo A - Caja POS (ADR-003, L4). Tablas de caja viven en el mismo sales DB.
	var cashSessionCtrl *salesController.CashSessionController
	if db != nil {
		cashRepo := salesPersistence.NewCashRegisterPostgresRepository(db)
		cashConfig := salesAdapter.NewEnvCashConfigAdapter()
		openCashUC := salesUseCase.NewOpenCashSessionUseCase(cashRepo, cashConfig, publishUseCase)
		currentCashUC := salesUseCase.NewGetCurrentCashSessionUseCase(cashRepo)
		getCashUC := salesUseCase.NewGetCashSessionUseCase(cashRepo, paymentMethodAdapter)
		movementUC := salesUseCase.NewRegisterCashMovementUseCase(cashRepo, publishUseCase)
		closeCashUC := salesUseCase.NewCloseCashSessionUseCase(cashRepo, paymentMethodAdapter, publishUseCase)
		approveCashUC := salesUseCase.NewApproveCashReviewUseCase(cashRepo, publishUseCase)
		cashSessionCtrl = salesController.NewCashSessionController(openCashUC, currentCashUC, getCashUC, movementUC, closeCashUC, approveCashUC)
	}

	// Crear controladores
	salesCtrl := salesController.NewOrderController(validateStockUC, reserveStockUC, releaseStockUC, createOrderUC, confirmOrderUC, cancelOrderUC, listOrdersUC, getOrderUC, getOrderFinancialUC, posSaleUC, listPosSalesUC, getPosSaleUC, generatePosSalePdfUC, registerPaymentUC, applyCustomerCreditUC)

	// HITO C - Report Controller (daily + open-orders + aging + customer-balance)
	dailyReportRepo := salesPersistence.NewDailyReportPostgresRepository(db)
	dailyReportUC := salesUseCase.NewDailyReportUseCase(dailyReportRepo)
	var openOrdersReportUC *salesReports.GetOpenOrdersReportUseCase
	var agingReportUC *salesReports.GetAgingReportUseCase
	var customerBalanceUC *salesReports.GetCustomerBalanceUseCase
	if reportRepo != nil {
		openOrdersReportUC = salesReports.NewGetOpenOrdersReportUseCase(reportRepo)
		agingReportUC = salesReports.NewGetAgingReportUseCase(reportRepo)
		customerBalanceUC = salesReports.NewGetCustomerBalanceUseCase(reportRepo)
	}
	reportCtrl := salesController.NewReportController(dailyReportUC, openOrdersReportUC, agingReportUC, customerBalanceUC)

	// Registrar rutas
	salesCtrl.RegisterRoutes(router)
	if cashSessionCtrl != nil {
		cashSessionCtrl.RegisterRoutes(router)
	}
	customerCtrl := salesController.NewCustomerController(createCustomerCreditUC)
	customerCtrl.RegisterRoutes(router)
	reportCtrl.RegisterRoutes(router)
	metricsCtrl := salesController.NewMetricsController()
	metricsCtrl.RegisterRoutes(router)

	log.Println("Módulo Sales configurado exitosamente")
}
