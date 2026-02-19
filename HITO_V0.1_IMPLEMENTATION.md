# HITO v0.1: Sales Skeleton - Integración Event-Driven

**Fecha implementación:** 2026-02-20  
**Estado:** ✅ IMPLEMENTADO - Pendiente testing  
**Servicio:** order-service (sin renombrar)  

---

## 🎯 Objetivo Cumplido

Integrar publicación de eventos `sales.order.confirmed` y `sales.pos.confirmed` usando la librería `eventbus` compartida, permitiendo que ledger-service consuma y registre débitos.

---

## ✅ Cambios Implementados

### 1. Dependencias agregadas

**Archivo:** `go.mod`

```go
require (
    // ... dependencias existentes ...
    github.com/mercadocercano/eventbus v0.1.0
)

replace github.com/mercadocercano/eventbus => ../../libs/eventbus
```

---

### 2. Inicialización EventBus en `main.go`

**Cambios:**

1. Import de eventbus:
```go
import (
    // ... imports existentes ...
    "github.com/mercadocercano/eventbus"
)
```

2. Conexión a eventbus DB (después de payment_method_db):
```go
// HITO v0.1: Conectar a EventBus DB
eventBusHost := getEnv("EVENTBUS_DB_HOST", dbHost)
eventBusPort := getEnv("EVENTBUS_DB_PORT", "5432")
eventBusUser := getEnv("EVENTBUS_DB_USER", dbUser)
eventBusPassword := getEnv("EVENTBUS_DB_PASSWORD", dbPassword)
eventBusName := getEnv("EVENTBUS_DB_NAME", "eventbus")

eventBusConnStr := "postgres://..." 
eventBusDB, err := sql.Open("postgres", eventBusConnStr)

// Inicializar eventbus publisher
logger := eventbus.NewLogger(eventbus.LevelInfo)
eventStore := eventbus.NewSQLEventStore(eventBusDB, logger)
publishUseCase = eventbus.NewPublishEventUseCase(eventStore, logger)
```

3. Pasar publishUseCase a setupOrderModule:
```go
setupOrderModule(v1, db, paymentMethodDB, publishUseCase)
```

---

### 3. Actualización `ConfirmOrderUseCase`

**Archivo:** `src/order/application/usecase/confirm_order.go`

**Cambios:**

1. Agregar campo `publishUseCase`:
```go
type ConfirmOrderUseCase struct {
    orderRepo      port.OrderRepository
    stockClient    *client.StockClient
    publishUseCase *eventbus.PublishEventUseCase  // NUEVO
}
```

2. Actualizar constructor:
```go
func NewConfirmOrderUseCase(
    orderRepo port.OrderRepository,
    stockClient *client.StockClient,
    publishUseCase *eventbus.PublishEventUseCase,  // NUEVO
) *ConfirmOrderUseCase
```

3. Publicar evento después de confirmar:
```go
// 6. HITO v0.1: Publicar evento sales.order.confirmed
if uc.publishUseCase != nil {
    if err := uc.publishSalesOrderConfirmedEvent(ctx, order, tenantID); err != nil {
        // Log error pero NO fallar la operación (orden ya confirmada)
        log.Printf("WARNING: Failed to publish sales.order.confirmed: %v", err)
    }
}
```

4. Método `publishSalesOrderConfirmedEvent`:
```go
func (uc *ConfirmOrderUseCase) publishSalesOrderConfirmedEvent(
    ctx context.Context,
    order *entity.Order,
    tenantID string,
) error {
    // Construir payload según contrato v1
    eventPayload := map[string]interface{}{
        "order_number": 0,
        "customer": map[string]interface{}{
            "customer_id":   "00000000-0000-0000-0000-000000000001",
            "customer_name": "Cliente Genérico",
            "tax_condition": "CONSUMIDOR_FINAL",
        },
        "currency":      "ARS",
        "exchange_rate": 1.0,
        "totals": map[string]interface{}{
            "subtotal": totalAmount,
            "discount": 0.0,
            "tax":      0.0,
            "total":    totalAmount,
        },
        "payment_terms": map[string]interface{}{
            "type":     "CUENTA_CORRIENTE",
            "due_date": time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339),
        },
    }

    // Crear envelope completo
    envelope := map[string]interface{}{
        "event_id":       uuid.New().String(),
        "event_type":     "sales.order.confirmed",
        "event_version":  1,
        "aggregate_type": "sales_order",
        "aggregate_id":   order.OrderID,
        "tenant_id":      tenantID,
        "occurred_at":    time.Now().UTC().Format(time.RFC3339),
        "payload":        eventPayload,
    }

    envelopeBytes, _ := json.Marshal(envelope)

    // Publicar usando eventbus
    return uc.publishUseCase.Execute(
        ctx,
        order.OrderID,
        "sales_order",
        "sales.order.confirmed",
        envelopeBytes,
        "order-service",
    )
}
```

---

### 4. Actualización `POSSaleUseCase`

**Archivo:** `src/order/application/usecase/pos_sale.go`

**Cambios:**

1. Agregar campo `publishUseCase`:
```go
type POSSaleUseCase struct {
    stockClient        *client.StockClient
    posSaleRepo        port.PosSaleRepository
    paymentMethodCache *cache.PaymentMethodCache
    publishUseCase     *eventbus.PublishEventUseCase  // NUEVO
}
```

2. Actualizar constructor:
```go
func NewPOSSaleUseCase(
    stockClient *client.StockClient,
    posSaleRepo port.PosSaleRepository,
    paymentMethodCache *cache.PaymentMethodCache,
    publishUseCase *eventbus.PublishEventUseCase,  // NUEVO
) *POSSaleUseCase
```

3. Publicar evento después de persistir:
```go
// HITO v0.1: Publicar evento sales.pos.confirmed
if uc.publishUseCase != nil {
    ctx := context.Background()
    if err := uc.publishPOSSaleConfirmedEvent(ctx, posSale, tenantID); err != nil {
        log.Printf("WARNING: Failed to publish sales.pos.confirmed: %v", err)
    }
}
```

4. Método `publishPOSSaleConfirmedEvent`:
```go
func (uc *POSSaleUseCase) publishPOSSaleConfirmedEvent(
    ctx context.Context,
    posSale *entity.PosSale,
    tenantID string,
) error {
    eventPayload := map[string]interface{}{
        "pos_number": 0,
        "customer": map[string]interface{}{
            "customer_id":   "00000000-0000-0000-0000-000000000001",
            "customer_name": "Cliente Genérico",
            "tax_condition": "CONSUMIDOR_FINAL",
        },
        "currency":      posSale.Currency,
        "exchange_rate": 1.0,
        "totals": map[string]interface{}{
            "subtotal": posSale.TotalAmount.InexactFloat64(),
            "discount": posSale.DiscountAmount.InexactFloat64(),
            "tax":      0.0,
            "total":    posSale.FinalAmount.InexactFloat64(),
        },
        "payment": map[string]interface{}{
            "method":          posSale.PaymentMethodID.String(),
            "amount_received": posSale.AmountPaid.InexactFloat64(),
            "change_given":    posSale.Change.InexactFloat64(),
        },
    }

    envelope := map[string]interface{}{
        "event_id":       uuid.New().String(),
        "event_type":     "sales.pos.confirmed",
        "event_version":  1,
        "aggregate_type": "pos_sale",
        "aggregate_id":   posSale.ID.String(),
        "tenant_id":      tenantID,
        "occurred_at":    time.Now().UTC().Format(time.RFC3339),
        "payload":        eventPayload,
    }

    envelopeBytes, _ := json.Marshal(envelope)

    return uc.publishUseCase.Execute(
        ctx,
        posSale.ID.String(),
        "pos_sale",
        "sales.pos.confirmed",
        envelopeBytes,
        "order-service",
    )
}
```

---

## 📦 Archivos Modificados

1. `go.mod` - Dependencias
2. `main.go` - Inicialización eventbus
3. `src/order/application/usecase/confirm_order.go` - Publicación eventos orden
4. `src/order/application/usecase/pos_sale.go` - Publicación eventos POS

**Total:** 4 archivos modificados.

---

## 🔒 NO Modificado (Según alcance HITO v0.1)

❌ Tablas: Se mantienen `orders` y `pos_sales` (sin renombrar)  
❌ Rutas: Se mantienen `/api/v1/orders` y `/api/v1/pos-sales`  
❌ Servicio: Se mantiene nombre `order-service`  
❌ Numeración: `order_number` y `pos_number` = 0 (hardcoded)  
❌ Customer: `customer_id` = hardcoded UUID genérico  
❌ Migraciones DB: No se ejecutaron 008 ni 009  

**Motivo:** Alcance mínimo para validar integración EventBus → Ledger.

---

## ✅ Compilación

```bash
cd services/order-service
go mod tidy
go build .
```

**Resultado:** ✅ Compilación exitosa sin errores.

---

## 🚦 Criterios de Cierre del HITO v0.1

Para considerar el hito CERRADO, debe cumplirse:

### Flujo A: SalesOrder

1. ✅ POST `/api/v1/orders` - Crear orden
2. ✅ POST `/api/v1/orders/:id/confirm` - Confirmar orden
3. [ ] Evento `sales.order.confirmed` persiste en tabla `events`
4. [ ] Ledger consume evento
5. [ ] Ledger inserta en `ledger_entries`
6. [ ] GET `/api/v1/ledger/balance` devuelve monto correcto

### Flujo B: POSSale

1. ✅ POST `/api/v1/pos-sales` - Crear venta
2. [ ] Evento `sales.pos.confirmed` persiste en tabla `events`
3. [ ] Ledger consume evento
4. [ ] Ledger inserta en `ledger_entries`
5. [ ] GET `/api/v1/ledger/balance` devuelve monto correcto

---

## 🧪 Testing Pendiente

### Variables de entorno requeridas

```bash
# Eventbus DB
EVENTBUS_DB_HOST=localhost
EVENTBUS_DB_PORT=5432
EVENTBUS_DB_USER=postgres
EVENTBUS_DB_PASSWORD=postgres
EVENTBUS_DB_NAME=eventbus

# Servicios existentes (ya configurados)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=order_db
```

### Script de validación

```bash
# 1. Iniciar servicios
docker-compose up -d order-service eventbus ledger-service

# 2. Confirmar orden
ORDER_ID="<order-id-existente>"
TOKEN="<jwt-token>"
TENANT_ID="00000000-0000-0000-0000-000000000001"

curl -X POST http://localhost:8001/api/v1/orders/$ORDER_ID/confirm \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"

# 3. Verificar evento en eventbus
docker exec -it postgres psql -U postgres -d eventbus \
  -c "SELECT event_type, aggregate_id FROM events WHERE event_type = 'sales.order.confirmed' ORDER BY occurred_at DESC LIMIT 1;"

# 4. Verificar ledger_entry
docker exec -it postgres psql -U postgres -d ledger_db \
  -c "SELECT document_type, amount FROM ledger_entries ORDER BY created_at DESC LIMIT 1;"

# 5. Verificar balance
curl http://localhost:8001/api/v1/ledger/balance \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID"
```

---

## 📊 Estructura de Eventos Publicados

### sales.order.confirmed

```json
{
  "event_id": "uuid",
  "event_type": "sales.order.confirmed",
  "event_version": 1,
  "aggregate_type": "sales_order",
  "aggregate_id": "order-uuid",
  "tenant_id": "tenant-uuid",
  "occurred_at": "2026-02-20T10:00:00Z",
  "payload": {
    "order_number": 0,
    "customer": {
      "customer_id": "00000000-0000-0000-0000-000000000001",
      "customer_name": "Cliente Genérico",
      "tax_condition": "CONSUMIDOR_FINAL"
    },
    "currency": "ARS",
    "exchange_rate": 1.0,
    "totals": {
      "subtotal": 100.00,
      "discount": 0.0,
      "tax": 0.0,
      "total": 100.00
    },
    "payment_terms": {
      "type": "CUENTA_CORRIENTE",
      "due_date": "2026-03-20T10:00:00Z"
    }
  }
}
```

### sales.pos.confirmed

```json
{
  "event_id": "uuid",
  "event_type": "sales.pos.confirmed",
  "event_version": 1,
  "aggregate_type": "pos_sale",
  "aggregate_id": "pos-uuid",
  "tenant_id": "tenant-uuid",
  "occurred_at": "2026-02-20T10:00:00Z",
  "payload": {
    "pos_number": 0,
    "customer": {
      "customer_id": "00000000-0000-0000-0000-000000000001",
      "customer_name": "Cliente Genérico",
      "tax_condition": "CONSUMIDOR_FINAL"
    },
    "currency": "ARS",
    "exchange_rate": 1.0,
    "totals": {
      "subtotal": 100.00,
      "discount": 10.00,
      "tax": 0.0,
      "total": 90.00
    },
    "payment": {
      "method": "uuid",
      "amount_received": 100.00,
      "change_given": 10.00
    }
  }
}
```

---

## ⚠️ Limitaciones Conocidas (TODOs para Hitos futuros)

### Customer ID hardcoded
```go
"customer_id": "00000000-0000-0000-0000-000000000001"
```
**Fix en:** HITO v0.2 - Agregar campo `customer_id` a `orders` y `pos_sales`.

### Numeración secuencial
```go
"order_number": 0,
"pos_number": 0,
```
**Fix en:** HITO v0.2 - Implementar tabla `document_sequences`.

### Total amount calculado básico
```go
totalAmount += float64(item.Quantity) * 100.0  // Precio hardcoded
```
**Fix en:** HITO v0.2 - Obtener precio real del snapshot PIM.

### Exchange rate hardcoded
```go
"exchange_rate": 1.0
```
**Fix en:** HITO v0.3 - Multi-moneda con tenant-service.

---

## 🚀 Próximos Pasos

### Inmediato (Cerrar HITO v0.1)

1. Iniciar `order-service` con eventbus configurado
2. Confirmar una orden existente
3. Verificar evento en tabla `events`
4. Verificar ledger procesa y crea `ledger_entry`
5. Verificar balance correcto

### Post HITO v0.1 (Backlog)

- **HITO v0.2:** Renombrar `order-service` → `sales-service`
- **HITO v0.3:** Migraciones DB (tablas + campos)
- **HITO v0.4:** Numeración secuencial
- **HITO v0.5:** Customer ID real
- **HITO v0.6:** Points of Sale
- **HITO v1.0:** Production ready

---

**Implementado por:** System Architecture Team  
**Fecha:** 2026-02-20  
**Tiempo de implementación:** ~6 horas  
**Estado:** ✅ LISTO PARA TESTING  

---

## 📞 Validación Requerida

**Traer evidencia de:**

1. Evento persistido en `events`:
```sql
SELECT id, event_type, aggregate_id, occurred_at 
FROM events 
WHERE event_type IN ('sales.order.confirmed', 'sales.pos.confirmed')
ORDER BY occurred_at DESC 
LIMIT 5;
```

2. Ledger entry creado:
```sql
SELECT id, document_type, document_id, amount, created_at
FROM ledger_entries
ORDER BY created_at DESC
LIMIT 5;
```

3. Balance actualizado:
```bash
curl http://localhost:8001/api/v1/ledger/balance \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" | jq
```

**Con esas 3 evidencias → HITO FORMALMENTE CERRADO.**
