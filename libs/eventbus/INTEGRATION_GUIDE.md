# 🔌 Guía de Integración del EventBus

**Para**: Desarrolladores de microservicios del ERP Mercado Cercano  
**Propósito**: Integrar el EventBus compartido en servicios existentes

---

## 📋 Prerequisitos

1. EventBus ya instalado en `/libs/eventbus`
2. PostgreSQL accesible desde tu servicio
3. Go 1.22+ instalado

---

## 🚀 Integración en 5 Pasos

### Paso 1: Agregar Dependencia

En tu servicio (ej: `sales-service`):

```bash
cd services/sales-service
go get github.com/mercadocercano/eventbus
```

O agregar en `go.mod`:

```go
require (
    github.com/mercadocercano/eventbus v0.1.0
)

replace github.com/mercadocercano/eventbus => ../../libs/eventbus
```

### Paso 2: Publicar Eventos

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    
    "github.com/mercadocercano/eventbus/internal/application"
    "github.com/mercadocercano/eventbus/internal/infrastructure/persistence"
    "github.com/mercadocercano/eventbus/internal/shared"
)

func publishSaleCreated(saleID string, amount float64) error {
    // 1. Conectar a EventBus DB
    db, _ := sql.Open("postgres", "host=localhost dbname=eventbus...")
    defer db.Close()
    
    // 2. Crear use case
    logger := shared.NewLogger(shared.LevelInfo)
    eventStore := persistence.NewSQLEventStore(db, logger)
    publishUseCase := application.NewPublishEventUseCase(eventStore, logger)
    
    // 3. Preparar payload
    payload := map[string]interface{}{
        "sale_id": saleID,
        "amount":  amount,
        "items":   []string{"item1", "item2"},
    }
    payloadBytes, _ := json.Marshal(payload)
    
    // 4. Publicar evento
    return publishUseCase.Execute(
        context.Background(),
        saleID,              // aggregate_id
        "sale",              // aggregate_type
        "sale.created",      // event_type
        payloadBytes,        // payload
        "sales-service",     // published_by
    )
}
```

### Paso 3: Implementar Handler

```go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    
    "github.com/mercadocercano/eventbus/internal/domain"
    "github.com/mercadocercano/eventbus/internal/shared"
)

type LedgerEventHandler struct {
    logger *shared.Logger
    // Tus repositorios, servicios, etc.
}

func NewLedgerEventHandler(logger *shared.Logger) *LedgerEventHandler {
    return &LedgerEventHandler{logger: logger}
}

func (h *LedgerEventHandler) ConsumerName() string {
    return "ledger-service"
}

func (h *LedgerEventHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    h.logger.Info("Processing event", map[string]interface{}{
        "event_id":   event.ID(),
        "event_type": event.EventType(),
    })
    
    switch event.EventType() {
    case "sale.created":
        return h.handleSaleCreated(ctx, event)
    case "sale.cancelled":
        return h.handleSaleCancelled(ctx, event)
    default:
        h.logger.Warn("Unknown event type", map[string]interface{}{
            "event_type": event.EventType(),
        })
        return nil
    }
}

func (h *LedgerEventHandler) handleSaleCreated(ctx context.Context, event domain.DomainEvent) error {
    var payload struct {
        SaleID string  `json:"sale_id"`
        Amount float64 `json:"amount"`
    }
    
    if err := json.Unmarshal(event.Payload(), &payload); err != nil {
        return fmt.Errorf("failed to unmarshal payload: %w", err)
    }
    
    // IMPORTANTE: Implementar lógica idempotente
    // Verificar si ya fue procesado (por event.ID())
    
    h.logger.Info("Registering sale in ledger", map[string]interface{}{
        "sale_id": payload.SaleID,
        "amount":  payload.Amount,
    })
    
    // Tu lógica de negocio aquí
    // ...
    
    return nil
}
```

### Paso 4: Configurar Worker

```go
package main

import (
    "context"
    "database/sql"
    
    "github.com/mercadocercano/eventbus/internal/application"
    "github.com/mercadocercano/eventbus/internal/infrastructure/config"
    "github.com/mercadocercano/eventbus/internal/infrastructure/persistence"
    "github.com/mercadocercano/eventbus/internal/infrastructure/worker"
    "github.com/mercadocercano/eventbus/internal/shared"
    
    "your-service/handlers"
)

func main() {
    // 1. Cargar configuración
    cfg, _ := config.Load()
    logger := shared.NewLogger(shared.LogLevel(cfg.LogLevel))
    
    // 2. Conectar a EventBus DB
    db, _ := sql.Open("postgres", cfg.ConnectionString())
    defer db.Close()
    
    // 3. Crear worker
    eventStore := persistence.NewSQLEventStore(db, logger)
    processUseCase := application.NewProcessEventUseCase(eventStore, logger)
    
    eventWorker := worker.NewEventWorker(
        processUseCase,
        logger,
        cfg.Worker.BatchSize,
        cfg.Worker.PollInterval,
    )
    
    // 4. Registrar handlers
    ledgerHandler := handlers.NewLedgerEventHandler(logger)
    eventWorker.RegisterHandler(ledgerHandler)
    
    // Puedes registrar múltiples handlers si tu servicio
    // consume diferentes tipos de eventos
    
    // 5. Iniciar worker
    ctx := context.Background()
    eventWorker.Start(ctx)
    
    // 6. Esperar shutdown
    // ... (señales, graceful shutdown, etc.)
}
```

### Paso 5: Variables de Entorno

Agregar a tu `.env`:

```env
# EventBus Configuration
EVENTBUS_DB_HOST=localhost
EVENTBUS_DB_PORT=5432
EVENTBUS_DB_USER=postgres
EVENTBUS_DB_PASSWORD=postgres123
EVENTBUS_DB_NAME=eventbus

EVENTBUS_BATCH_SIZE=10
EVENTBUS_POLL_INTERVAL=5s
```

---

## ⚠️ Reglas Críticas de Integración

### 1. Idempotencia OBLIGATORIA

```go
func (h *Handler) Handle(ctx context.Context, event domain.DomainEvent) error {
    // ❌ MAL: Sin verificación de duplicados
    ledger.Create(payload)
    
    // ✅ BIEN: Verificar si ya fue procesado
    if ledger.ExistsByEventID(event.ID()) {
        return nil // Ya procesado, skip
    }
    ledger.CreateWithEventID(payload, event.ID())
    return nil
}
```

**Razón**: EventBus garantiza at-least-once, NO exactly-once.

### 2. No Modificar event_bus

```go
// ❌ NUNCA HACER ESTO
db.Exec("DELETE FROM event_bus WHERE ...")
db.Exec("UPDATE event_bus SET ...")

// ✅ CORRECTO: Solo lectura desde consumers
events, _ := eventStore.GetUnprocessed(...)
```

**Razón**: event_bus es append-only. Solo escritura en publish.

### 3. Manejo de Errores

```go
func (h *Handler) Handle(ctx context.Context, event domain.DomainEvent) error {
    // ❌ MAL: Panic sin manejar
    panic("something wrong")
    
    // ✅ BIEN: Retornar error para retry
    if err := processEvent(event); err != nil {
        return fmt.Errorf("failed to process: %w", err)
    }
    return nil
}
```

**Razón**: Errores retornados activan retry automático.

### 4. Payloads JSON Válidos

```go
// ❌ MAL: Payload inválido
payload := []byte("not a json")

// ✅ BIEN: JSON válido
payload := json.Marshal(map[string]interface{}{
    "sale_id": "123",
    "amount":  100.50,
})
```

**Razón**: SQLEventStore valida JSON antes de guardar.

---

## 📊 Patrones Comunes

### Patrón 1: Un Servicio, Múltiples Handlers

```go
// Stock service escucha múltiples eventos
stockHandler := NewStockEventHandler()  // sale.created, return.created
inventoryHandler := NewInventoryHandler() // inventory.low

eventWorker.RegisterHandler(stockHandler)
eventWorker.RegisterHandler(inventoryHandler)
```

### Patrón 2: Transacciones Locales

```go
func (h *Handler) Handle(ctx context.Context, event domain.DomainEvent) error {
    tx, _ := h.db.BeginTx(ctx, nil)
    defer tx.Rollback()
    
    // 1. Procesar evento
    if err := h.processInTransaction(tx, event); err != nil {
        return err
    }
    
    // 2. Guardar event_id para idempotencia
    if err := h.saveProcessedEventID(tx, event.ID()); err != nil {
        return err
    }
    
    return tx.Commit()
}
```

### Patrón 3: Publicar Eventos en Cadena

```go
// Sales service publica sale.created
publishUseCase.Execute(ctx, saleID, "sale", "sale.created", ...)

// Ledger handler consume y publica ledger.entry_created
func (h *LedgerHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
    // Registrar en ledger
    entryID := h.createLedgerEntry(event)
    
    // Publicar nuevo evento
    return h.publishUseCase.Execute(
        ctx, entryID, "ledger", "ledger.entry_created", ...
    )
}
```

---

## 🧪 Testing

### Test de Publisher

```go
func TestPublishSaleCreated(t *testing.T) {
    // 1. Setup testcontainer PostgreSQL
    db := setupTestDB(t)
    defer db.Close()
    
    // 2. Crear publisher
    eventStore := persistence.NewSQLEventStore(db, logger)
    publishUseCase := application.NewPublishEventUseCase(eventStore, logger)
    
    // 3. Publicar evento
    err := publishUseCase.Execute(
        context.Background(),
        "sale-123",
        "sale",
        "sale.created",
        []byte(`{"amount": 100}`),
        "sales-service",
    )
    
    // 4. Verificar
    assert.NoError(t, err)
    
    var count int
    db.QueryRow("SELECT COUNT(*) FROM event_bus WHERE aggregate_id = 'sale-123'").Scan(&count)
    assert.Equal(t, 1, count)
}
```

### Test de Handler

```go
func TestLedgerHandler(t *testing.T) {
    handler := NewLedgerEventHandler(logger)
    
    event := domain.NewEvent(
        "sale-123",
        "sale",
        "sale.created",
        []byte(`{"amount": 100}`),
        "sales-service",
    )
    
    err := handler.Handle(context.Background(), event)
    assert.NoError(t, err)
    
    // Verificar idempotencia
    err = handler.Handle(context.Background(), event)
    assert.NoError(t, err) // No debe fallar ni duplicar
}
```

---

## 🔍 Debugging

### Ver eventos pendientes

```sql
SELECT eb.*, ec.*
FROM event_bus eb
LEFT JOIN event_consumers ec ON eb.id = ec.event_id
WHERE ec.consumer_name = 'ledger-service'
AND ec.status != 'processed'
ORDER BY eb.occurred_at;
```

### Ver eventos fallidos

```sql
SELECT *
FROM event_consumers
WHERE status = 'failed'
AND consumer_name = 'ledger-service'
ORDER BY processed_at DESC;
```

### Reintentear evento fallido manualmente

```sql
-- Resetear retry count
UPDATE event_consumers
SET retry_count = 0, status = 'pending'
WHERE event_id = 'abc-123'
AND consumer_name = 'ledger-service';
```

---

## 📚 Recursos Adicionales

- **README.md**: Guía de uso general
- **ARCHITECTURE.md**: Detalles técnicos
- **examples/**: Código de ejemplo funcional

---

## 🆘 Problemas Comunes

### "Event already processed" pero no veo resultados

**Causa**: Handler retornó nil pero no completó procesamiento  
**Solución**: Implementar verificación de idempotencia real

### Worker no consume eventos

**Causa**: Handler no registrado o nombre incorrecto  
**Solución**: Verificar `ConsumerName()` y `RegisterHandler()`

### Eventos duplicados en destino

**Causa**: Handler no implementa idempotencia  
**Solución**: Verificar por event_id antes de procesar

---

**Happy Integration!** 🎉
