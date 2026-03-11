# EventBus - Event Bus Persistente Compartido

Event Bus persistente con arquitectura hexagonal para el ERP Mercado Cercano.

## 🎯 Características

- ✅ **Persistencia append-only** en PostgreSQL
- ✅ **Arquitectura Hexagonal + DDD**
- ✅ **Idempotencia** por consumidor
- ✅ **Retry automático** con backoff
- ✅ **Worker genérico** con procesamiento concurrente
- ✅ **Sin dependencias de dominio** (infraestructura pura)
- ✅ **Migrable** a Redis/Kafka/EventStore

## 🏗️ Arquitectura

```
eventbus/
├── cmd/
│   └── worker/              # Worker genérico ejecutable
├── internal/
│   ├── domain/              # Interfaces y entidades
│   │   ├── event.go
│   │   ├── event_store.go
│   │   ├── consumer.go
│   │   └── errors.go
│   ├── application/         # Casos de uso
│   │   ├── publish_event.go
│   │   └── process_event.go
│   ├── infrastructure/      # Implementaciones
│   │   ├── persistence/
│   │   │   └── sql_event_store.go
│   │   ├── worker/
│   │   │   └── event_worker.go
│   │   └── config/
│   │       └── config.go
│   └── shared/
│       └── logger.go
├── migrations/              # Migraciones SQL
├── examples/                # Ejemplos de uso
└── go.mod
```

## 📦 Instalación

```bash
cd libs/eventbus
go mod download
```

## 🗄️ Base de Datos

### Crear la base de datos

```bash
createdb eventbus
```

### Ejecutar migraciones

```bash
psql -d eventbus -f migrations/001_create_event_bus_tables.up.sql
```

### Rollback

```bash
psql -d eventbus -f migrations/001_create_event_bus_tables.down.sql
```

## ⚙️ Configuración

Copiar `.env.example` a `.env` y configurar:

```bash
cp .env.example .env
```

Variables de entorno:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres123
DB_NAME=eventbus
DB_SSL_MODE=disable

# Worker
WORKER_BATCH_SIZE=10
WORKER_MAX_RETRIES=3
WORKER_RETRY_DELAY=5s
WORKER_POLL_INTERVAL=5s

# Logging
LOG_LEVEL=INFO
SERVICE_NAME=eventbus-worker
```

## 🚀 Uso

### 1. Publicar Eventos

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    
    _ "github.com/lib/pq"
    
    "github.com/mercadocercano/eventbus/internal/application"
    "github.com/mercadocercano/eventbus/internal/infrastructure/config"
    "github.com/mercadocercano/eventbus/internal/infrastructure/persistence"
    "github.com/mercadocercano/eventbus/internal/shared"
)

type SaleCreatedPayload struct {
    SaleID      string  `json:"sale_id"`
    TotalAmount float64 `json:"total_amount"`
}

func main() {
    cfg, _ := config.Load()
    logger := shared.NewLogger(shared.LogLevel(cfg.LogLevel))
    
    db, _ := sql.Open("postgres", cfg.ConnectionString())
    defer db.Close()
    
    eventStore := persistence.NewSQLEventStore(db, logger)
    publishUseCase := application.NewPublishEventUseCase(eventStore, logger)
    
    payload := SaleCreatedPayload{
        SaleID:      "sale-123",
        TotalAmount: 1500.50,
    }
    
    payloadBytes, _ := json.Marshal(payload)
    
    ctx := context.Background()
    publishUseCase.Execute(
        ctx,
        "sale-123",        // aggregate_id
        "sale",            // aggregate_type
        "sale.created",    // event_type
        payloadBytes,      // payload
        "sales-service",   // published_by
    )
}
```

### 2. Consumir Eventos

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    
    "github.com/mercadocercano/eventbus/internal/application"
    "github.com/mercadocercano/eventbus/internal/domain"
    "github.com/mercadocercano/eventbus/internal/infrastructure/config"
    "github.com/mercadocercano/eventbus/internal/infrastructure/persistence"
    "github.com/mercadocercano/eventbus/internal/infrastructure/worker"
    "github.com/mercadocercano/eventbus/internal/shared"
)

// Implementar EventHandler
type LedgerEventHandler struct {
    logger *shared.Logger
}

func (h *LedgerEventHandler) ConsumerName() string {
    return "ledger-service"
}

func (h *LedgerEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    if event.EventType() == "sale.created" {
        var payload SaleCreatedPayload
        json.Unmarshal(event.Payload(), &payload)
        
        // Procesar evento
        fmt.Printf("Ledger: Registering sale %s\n", payload.SaleID)
    }
    return nil
}

func main() {
    cfg, _ := config.Load()
    logger := shared.NewLogger(shared.LogLevel(cfg.LogLevel))
    
    db, _ := sql.Open("postgres", cfg.ConnectionString())
    defer db.Close()
    
    eventStore := persistence.NewSQLEventStore(db, logger)
    processUseCase := application.NewProcessEventUseCase(eventStore, logger)
    
    eventWorker := worker.NewEventWorker(
        processUseCase,
        logger,
        cfg.Worker.BatchSize,
        cfg.Worker.PollInterval,
    )
    
    // Registrar handler
    ledgerHandler := &LedgerEventHandler{logger: logger}
    eventWorker.RegisterHandler(ledgerHandler)
    
    ctx := context.Background()
    eventWorker.Start(ctx)
    
    // Esperar señal de shutdown
    // ...
}
```

## 🧪 Ejemplos

### Ejecutar publisher de ejemplo

```bash
make run-example-publisher
```

### Ejecutar consumer de ejemplo

```bash
make run-example-consumer
```

### Ejecutar worker genérico

```bash
make run-worker
```

## 🔄 Flujo de Eventos

```
┌─────────────────┐
│  Sales Service  │
│  (Publisher)    │
└────────┬────────┘
         │ Publish Event
         ▼
┌─────────────────┐
│   EventStore    │
│  (SQL Append)   │
└────────┬────────┘
         │
         ├──────────────┬──────────────┬──────────────┐
         ▼              ▼              ▼              ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│Ledger Worker │ │Stock Worker  │ │Fiscal Worker │ │Report Worker │
│ (Consumer)   │ │ (Consumer)   │ │ (Consumer)   │ │ (Consumer)   │
└──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘
```

## 📊 Modelo de Datos

### event_bus

| Campo          | Tipo         | Descripción                        |
|----------------|--------------|-------------------------------------|
| id             | UUID         | ID único del evento                |
| aggregate_type | VARCHAR(100) | Tipo de agregado (sale, inventory) |
| aggregate_id   | UUID         | ID del agregado                    |
| event_type     | VARCHAR(100) | Tipo de evento (sale.created)      |
| payload_json   | JSONB        | Payload del evento                 |
| occurred_at    | TIMESTAMP    | Timestamp del evento               |
| published_by   | VARCHAR(100) | Servicio que publicó el evento     |

### event_consumers

| Campo         | Tipo         | Descripción                          |
|---------------|--------------|--------------------------------------|
| event_id      | UUID         | ID del evento                        |
| consumer_name | VARCHAR(100) | Nombre del consumidor                |
| processed_at  | TIMESTAMP    | Timestamp de procesamiento           |
| status        | VARCHAR(50)  | pending/processed/failed/retrying    |
| retry_count   | INT          | Contador de reintentos               |
| last_error    | TEXT         | Último error si falló                |

## 🔒 Garantías

- ✅ **At-least-once delivery**: Un evento puede procesarse más de una vez
- ✅ **Idempotencia obligatoria**: Los handlers deben ser idempotentes
- ✅ **Orden por agregado**: Eventos del mismo agregado se procesan en orden
- ✅ **Retry automático**: Máximo 3 reintentos por defecto
- ✅ **No se pierden eventos**: Append-only garantiza persistencia

## 🧩 Integración con Servicios

### Sales Service → Ledger Service

```go
// En sales-service (publisher)
publishUseCase.Execute(ctx, saleID, "sale", "sale.created", payload, "sales-service")

// En ledger-service (consumer)
type LedgerHandler struct {}
func (h *LedgerHandler) ConsumerName() string { return "ledger-service" }
func (h *LedgerHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    // Registrar en ledger
    return nil
}
```

## 📈 Métricas y Monitoreo

El worker emite logs estructurados en JSON:

```json
{
  "timestamp": "2025-02-19T10:30:00Z",
  "level": "INFO",
  "message": "Event processed successfully",
  "service": "eventbus",
  "event_id": "abc-123",
  "consumer": "ledger-service",
  "event_type": "sale.created"
}
```

## 🚀 Próximos Pasos

1. **Implementar Dead Letter Queue** para eventos con max retries excedido
2. **Agregar métricas Prometheus** (eventos procesados, latencia, errores)
3. **Implementar adaptador Redis** como alternativa a SQL
4. **Agregar soporte para Event Sourcing** completo
5. **Implementar snapshots** para agregados grandes

## 📝 Notas Importantes

- **Sin FK ni UNIQUE**: Control de integridad en código, no en DB
- **Sin triggers**: Toda la lógica en Go
- **Migrable**: Diseño permite cambiar a Redis/Kafka sin cambiar interfaces
- **Horizontal scaling**: Workers pueden correr en múltiples instancias
- **Idempotencia**: Responsabilidad de los handlers, no del eventbus

## 📄 Licencia

Propiedad de Mercado Cercano - Uso interno

---

**HITO 1 — Event Bus Persistente Compartido** ✅
