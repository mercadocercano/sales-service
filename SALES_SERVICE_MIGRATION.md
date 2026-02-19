# Migración order-service → sales-service

**Fecha:** 2026-02-20  
**Estado:** PLANIFICADO - Pendiente ejecución  
**Responsable:** Equipo Backend  

---

## 🎯 Objetivo

Evolucionar el `order-service` existente a `sales-service` completo, integrando con EventBus para cerrar flujo E2E:

```
SalesOrder → Confirm → Event → Ledger → Balance
POSSale → Create → Event → Ledger → Balance
```

---

## ✅ Estado Actual: Lo que YA tenemos

| Componente | Implementado | Detalle |
|------------|--------------|---------|
| **Tabla orders** | ✅ | Necesita migrar a `sales_orders` |
| **Tabla pos_sales** | ✅ | Necesita agregar campos fiscales |
| **POST /orders** | ✅ | Multi-item + snapshots PIM |
| **POST /orders/:id/confirm** | ✅ | Cambia estado a CONFIRMED |
| **POST /pos-sales** | ✅ | Multi-item + descuentos + vuelto |
| **Integración Stock** | ✅ | ProcessSaleAtomic + compensación |
| **Integración PIM** | ✅ | Snapshots inmutables |
| **Cache Payment Methods** | ✅ | Cache de payment_method_db |

**Código base:** ~80% implementado. Solo falta EventBus.

---

## ❌ Lo que falta (Hito v0.1)

| Componente | Estimación | Bloqueante |
|------------|------------|------------|
| EventBus client | 2h | ✅ |
| Publicar `sales.order.created` | 1h | ✅ |
| Publicar `sales.order.confirmed` | 1h | ✅ |
| Publicar `sales.pos.created` | 30min | ✅ |
| Publicar `sales.pos.confirmed` | 30min | ✅ |
| Migración DB (008, 009) | 2h | ✅ |
| Renombrar servicio completo | 3h | ✅ |
| Tests E2E | 2h | ✅ |
| **TOTAL** | **12-15h** | **1-2 días** |

---

## 📋 Plan de Ejecución

### Fase 1: Pre-requisitos (30 min)

```bash
# 1. Backup completo de order_db
pg_dump -U postgres order_db > backup_order_db_$(date +%Y%m%d).sql

# 2. Verificar servicios dependientes corriendo
docker ps | grep -E "eventbus|ledger|stock|pim"

# 3. Verificar EventBus operativo
curl http://localhost:8300/health
```

### Fase 2: Renombramiento (2-3h)

**Objetivo:** Cambiar nombres sin romper funcionalidad.

```bash
# 1. Renombrar directorio
cd /Users/hornosg/MyProjects/saas-mt/services
mv order-service sales-service

# 2. Actualizar go.mod
cd sales-service
sed -i '' 's/module order/module sales/g' go.mod

# 3. Actualizar imports
find . -name "*.go" -exec sed -i '' 's|"order/|"sales/|g' {} +

# 4. Actualizar Dockerfile
sed -i '' 's/order-service/sales-service/g' Dockerfile

# 5. Actualizar docker-compose
cd ../..
sed -i '' 's/order-service/sales-service/g' docker-compose.yml
sed -i '' 's/order-service/sales-service/g' docker-compose.services.yml

# 6. Actualizar Kong
cd services/api-gateway
# Editar kong.yml manualmente (ver abajo)

# 7. Compilar y verificar
cd ../sales-service
go build .
```

**Kong routes (editar manualmente):**

```yaml
# services/api-gateway/kong.yml
services:
  - name: sales-service  # antes: order-service
    url: http://sales-service:8080
    
routes:
  - name: sales-orders-route
    service: sales-service
    paths:
      - /api/v1/sales/orders  # antes: /orders
      
  - name: sales-pos-route
    service: sales-service
    paths:
      - /api/v1/sales/pos  # antes: /pos-sales
```

**Validación Fase 2:**

```bash
# Compilar
cd services/sales-service && go build .

# Verificar sin errores
echo $?  # Debe ser 0

# Verificar imports
grep -r "\"order/" src/  # No debe devolver nada
```

---

### Fase 3: EventBus Integration (4-6h)

**Objetivo:** Publicar eventos según contrato v1.0.

#### 3.1 Crear cliente EventBus

```bash
# Crear estructura
mkdir -p src/sales/infrastructure/eventbus
touch src/sales/infrastructure/eventbus/client.go
```

**Código:** `src/sales/infrastructure/eventbus/client.go`

```go
package eventbus

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"
)

type EventBusClient struct {
    baseURL string
    client  *http.Client
}

func NewEventBusClient() *EventBusClient {
    baseURL := os.Getenv("EVENTBUS_URL")
    if baseURL == "" {
        baseURL = "http://eventbus:8300"
    }
    return &EventBusClient{
        baseURL: baseURL,
        client:  &http.Client{Timeout: 5 * time.Second},
    }
}

func (c *EventBusClient) Publish(eventType string, payload interface{}) error {
    event := map[string]interface{}{
        "type":       eventType,
        "version":    1,
        "payload":    payload,
        "timestamp":  time.Now().UTC(),
    }
    
    body, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }
    
    req, err := http.NewRequest("POST", c.baseURL+"/publish", bytes.NewBuffer(body))
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("failed to publish event: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("eventbus returned status %d", resp.StatusCode)
    }
    
    return nil
}
```

#### 3.2 Actualizar main.go

```go
// main.go - setupOrderModule()

import (
    // ... imports existentes ...
    orderEventBus "sales/src/sales/infrastructure/eventbus"  // NUEVO
)

func setupOrderModule(router *gin.RouterGroup, db *sql.DB, paymentMethodDB *sql.DB) {
    // ... código existente ...
    
    // NUEVO: Crear cliente EventBus
    eventBusClient := orderEventBus.NewEventBusClient()
    
    // Actualizar casos de uso con EventBus
    if orderRepo != nil {
        createOrderUC = orderUseCase.NewCreateOrderUseCase(
            orderRepo, 
            pimClient, 
            stockClient,
            eventBusClient,  // NUEVO
        )
        confirmOrderUC = orderUseCase.NewConfirmOrderUseCase(
            orderRepo, 
            stockClient,
            eventBusClient,  // NUEVO
        )
        // ...
    }
    
    // Actualizar POS Sale UseCase
    if posSaleRepo != nil {
        posSaleUC = orderUseCase.NewPOSSaleUseCase(
            stockClient, 
            posSaleRepo, 
            pmCache,
            eventBusClient,  // NUEVO
        )
    }
}
```

#### 3.3 Actualizar CreateOrderUseCase

```go
// src/sales/application/usecase/create_order.go

import (
    // ... imports existentes ...
    "sales/src/sales/infrastructure/eventbus"
)

type CreateOrderUseCase struct {
    orderRepo   port.OrderRepository
    pimClient   *client.PIMClient
    stockClient *client.StockClient
    eventBus    *eventbus.EventBusClient  // NUEVO
}

func NewCreateOrderUseCase(
    orderRepo port.OrderRepository, 
    pimClient *client.PIMClient, 
    stockClient *client.StockClient,
    eventBus *eventbus.EventBusClient,  // NUEVO
) *CreateOrderUseCase {
    return &CreateOrderUseCase{
        orderRepo:   orderRepo,
        pimClient:   pimClient,
        stockClient: stockClient,
        eventBus:    eventBus,
    }
}

func (uc *CreateOrderUseCase) Execute(...) (*response.CreateOrderResponse, error) {
    // ... código existente de crear orden ...
    
    // Persistir
    if err := uc.orderRepo.Save(ctx, order); err != nil {
        uc.compensateProcessedStock(...)
        return nil, fmt.Errorf("error saving order (stock compensated): %w", err)
    }
    
    // NUEVO: Publicar sales.order.created
    eventPayload := map[string]interface{}{
        "order_id":     order.OrderID,
        "tenant_id":    order.TenantID,
        "customer_id":  order.CustomerID,  // puede ser nil
        "total_amount": calculateTotalAmount(order.Items),
        "currency":     "ARS",
        "status":       "CREATED",
        "items":        mapItemsToEvent(order.Items),
        "created_at":   order.CreatedAt,
    }
    
    if err := uc.eventBus.Publish("sales.order.created", eventPayload); err != nil {
        // NO fallar la operación, solo log
        log.Printf("WARNING: Failed to publish sales.order.created for order %s: %v", 
            order.OrderID, err)
    }
    
    return response, nil
}

// Helper para mapear items
func mapItemsToEvent(items []entity.OrderItem) []map[string]interface{} {
    result := make([]map[string]interface{}, len(items))
    for i, item := range items {
        result[i] = map[string]interface{}{
            "item_id":     item.ItemID,
            "sku":         item.SKU,
            "quantity":    item.Quantity,
            "unit_price":  item.UnitPrice,  // desde snapshot
            "subtotal":    item.Subtotal,
        }
    }
    return result
}

func calculateTotalAmount(items []entity.OrderItem) float64 {
    total := 0.0
    for _, item := range items {
        // Asumimos que OrderItem tiene campo UnitPrice del snapshot
        total += float64(item.Quantity) * item.UnitPrice
    }
    return total
}
```

#### 3.4 Actualizar ConfirmOrderUseCase

```go
// src/sales/application/usecase/confirm_order.go (CREAR SI NO EXISTE)

package usecase

import (
    "context"
    "fmt"
    "log"
    "sales/src/sales/domain/port"
    "sales/src/sales/infrastructure/client"
    "sales/src/sales/infrastructure/eventbus"
    "time"
)

type ConfirmOrderUseCase struct {
    orderRepo   port.OrderRepository
    stockClient *client.StockClient
    eventBus    *eventbus.EventBusClient
}

func NewConfirmOrderUseCase(
    orderRepo port.OrderRepository,
    stockClient *client.StockClient,
    eventBus *eventbus.EventBusClient,
) *ConfirmOrderUseCase {
    return &ConfirmOrderUseCase{
        orderRepo:   orderRepo,
        stockClient: stockClient,
        eventBus:    eventBus,
    }
}

func (uc *ConfirmOrderUseCase) Execute(ctx context.Context, tenantID, orderID string) error {
    // 1. Obtener orden
    order, err := uc.orderRepo.GetByID(ctx, orderID, tenantID)
    if err != nil {
        return fmt.Errorf("order not found: %w", err)
    }
    
    // 2. Confirmar (cambia estado)
    if err := order.Confirm(); err != nil {
        return fmt.Errorf("cannot confirm order: %w", err)
    }
    
    // 3. Persistir cambio
    if err := uc.orderRepo.Update(ctx, order); err != nil {
        return fmt.Errorf("failed to update order: %w", err)
    }
    
    // 4. NUEVO: Publicar sales.order.confirmed (contrato v1.0)
    eventPayload := map[string]interface{}{
        "order_id":      order.OrderID,
        "tenant_id":     order.TenantID,
        "customer_id":   order.CustomerID,
        "total_amount":  calculateTotalAmount(order.Items),
        "currency":      "ARS",
        "confirmed_at":  time.Now().UTC(),
        "items":         mapItemsToEvent(order.Items),
    }
    
    if err := uc.eventBus.Publish("sales.order.confirmed", eventPayload); err != nil {
        log.Printf("WARNING: Failed to publish sales.order.confirmed for order %s: %v", 
            order.OrderID, err)
    }
    
    return nil
}
```

#### 3.5 Actualizar POSSaleUseCase

```go
// src/sales/application/usecase/pos_sale.go

type POSSaleUseCase struct {
    stockClient *client.StockClient
    posSaleRepo port.PosSaleRepository
    pmCache     *cache.PaymentMethodCache
    eventBus    *eventbus.EventBusClient  // NUEVO
}

func NewPOSSaleUseCase(
    stockClient *client.StockClient,
    posSaleRepo port.PosSaleRepository,
    pmCache *cache.PaymentMethodCache,
    eventBus *eventbus.EventBusClient,  // NUEVO
) *POSSaleUseCase {
    return &POSSaleUseCase{
        stockClient: stockClient,
        posSaleRepo: posSaleRepo,
        pmCache:     pmCache,
        eventBus:    eventBus,
    }
}

func (uc *POSSaleUseCase) Execute(...) (*response.POSSaleResponse, error) {
    // ... código existente de crear POS sale ...
    
    // Persistir
    if err := uc.posSaleRepo.Save(ctx, posSale); err != nil {
        // compensar stock...
        return nil, err
    }
    
    // NUEVO: Publicar sales.pos.created
    eventPayload := mapPosSaleToEvent(posSale)
    
    if err := uc.eventBus.Publish("sales.pos.created", eventPayload); err != nil {
        log.Printf("WARNING: Failed to publish sales.pos.created: %v", err)
    }
    
    // NUEVO: Publicar sales.pos.confirmed (POS es auto-confirmado)
    if err := uc.eventBus.Publish("sales.pos.confirmed", eventPayload); err != nil {
        log.Printf("WARNING: Failed to publish sales.pos.confirmed: %v", err)
    }
    
    return response, nil
}

func mapPosSaleToEvent(ps *entity.PosSale) map[string]interface{} {
    return map[string]interface{}{
        "id":                ps.ID.String(),
        "tenant_id":         ps.TenantID.String(),
        "customer_id":       getCustomerIDOrNil(ps.CustomerID),
        "payment_method_id": ps.PaymentMethodID.String(),
        "total_amount":      ps.TotalAmount.InexactFloat64(),
        "discount_amount":   ps.DiscountAmount.InexactFloat64(),
        "final_amount":      ps.FinalAmount.InexactFloat64(),
        "amount_paid":       ps.AmountPaid.InexactFloat64(),
        "change":            ps.Change.InexactFloat64(),
        "currency":          ps.Currency,
        "items":             mapPosSaleItems(ps.Items),
        "created_at":        ps.CreatedAt,
    }
}

func getCustomerIDOrNil(id *uuid.UUID) interface{} {
    if id == nil {
        return nil
    }
    return id.String()
}

func mapPosSaleItems(items []entity.PosSaleItem) []map[string]interface{} {
    result := make([]map[string]interface{}, len(items))
    for i, item := range items {
        result[i] = map[string]interface{}{
            "id":          item.ID.String(),
            "sku":         item.SKU,
            "quantity":    item.Quantity.InexactFloat64(),
            "unit_price":  item.UnitPrice.InexactFloat64(),
            "subtotal":    item.Subtotal.InexactFloat64(),
        }
    }
    return result
}
```

**Validación Fase 3:**

```bash
# Compilar
go build .

# Ejecutar tests unitarios
go test ./... -v

# Iniciar servicio
docker-compose up sales-service

# Verificar logs (debe mostrar conexión a EventBus)
docker logs sales-service | grep -i eventbus

# Crear orden y verificar evento
curl -X POST http://localhost:8001/api/v1/sales/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"items":[{"sku":"TEST-001","quantity":1}]}'

# Verificar en EventBus
curl http://localhost:8300/events | jq '.[] | select(.type == "sales.order.created")'
```

---

### Fase 4: Migraciones DB (2-3h)

**Objetivo:** Alinear schema con contrato v1.0.

#### 4.1 Migración 008: orders → sales_orders

```sql
-- migrations/008_migrate_orders_to_sales_orders.sql

BEGIN;

-- Agregar columnas faltantes
ALTER TABLE orders ADD COLUMN IF NOT EXISTS order_number INT;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_id UUID;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS fiscal_status VARCHAR(50) DEFAULT 'PENDING';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS invoice_id UUID;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS total_amount DECIMAL;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS version INT DEFAULT 1;

-- Calcular total_amount para registros existentes
UPDATE orders SET total_amount = 0 WHERE total_amount IS NULL;

-- Hacer total_amount NOT NULL
ALTER TABLE orders ALTER COLUMN total_amount SET NOT NULL;

-- Renombrar tabla
ALTER TABLE orders RENAME TO sales_orders;
ALTER TABLE order_items RENAME TO sales_order_items;

-- Actualizar FK en items
ALTER TABLE sales_order_items RENAME COLUMN order_id TO sales_order_id;

-- Actualizar constraint de status
ALTER TABLE sales_orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE sales_orders ADD CONSTRAINT sales_orders_status_check 
    CHECK (status IN ('CREATED', 'CONFIRMED', 'CANCELED'));

-- Índices adicionales
CREATE INDEX IF NOT EXISTS idx_sales_orders_tenant_status ON sales_orders(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_sales_orders_customer ON sales_orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_fiscal_status ON sales_orders(fiscal_status);

-- Comentarios
COMMENT ON TABLE sales_orders IS 'Órdenes de venta - fuente de verdad comercial';
COMMENT ON COLUMN sales_orders.order_number IS 'Número secuencial de orden (futuro)';
COMMENT ON COLUMN sales_orders.fiscal_status IS 'Estado fiscal (PENDING, PROCESSING, APPROVED)';
COMMENT ON COLUMN sales_orders.invoice_id IS 'Referencia a factura generada';
COMMENT ON COLUMN sales_orders.version IS 'Versión para optimistic locking';

COMMIT;
```

#### 4.2 Migración 009: Extender pos_sales

```sql
-- migrations/009_extend_pos_sales.sql

BEGIN;

ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS pos_number INT;
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS point_of_sale_id UUID;
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS fiscal_status VARCHAR(50) DEFAULT 'PENDING';
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS invoice_id UUID;
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS version INT DEFAULT 1;

-- Índices
CREATE INDEX IF NOT EXISTS idx_pos_sales_tenant_created ON pos_sales(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pos_sales_fiscal_status ON pos_sales(fiscal_status);
CREATE INDEX IF NOT EXISTS idx_pos_sales_invoice ON pos_sales(invoice_id) WHERE invoice_id IS NOT NULL;

COMMENT ON TABLE pos_sales IS 'Ventas POS - fuente de verdad comercial mostrador';
COMMENT ON COLUMN pos_sales.pos_number IS 'Número secuencial de venta (futuro)';
COMMENT ON COLUMN pos_sales.point_of_sale_id IS 'Punto de venta fiscal (futuro)';

COMMIT;
```

**Ejecutar migraciones:**

```bash
# Conectar al contenedor de DB
docker exec -it postgres psql -U postgres -d sales_db

# Ejecutar migración 008
\i /migrations/008_migrate_orders_to_sales_orders.sql

# Ejecutar migración 009
\i /migrations/009_extend_pos_sales.sql

# Verificar
\d sales_orders
\d pos_sales

# Salir
\q
```

**Validación Fase 4:**

```bash
# Verificar estructura
docker exec -it postgres psql -U postgres -d sales_db -c "\d sales_orders"

# Verificar datos migrados
docker exec -it postgres psql -U postgres -d sales_db -c "SELECT COUNT(*) FROM sales_orders;"

# Verificar índices
docker exec -it postgres psql -U postgres -d sales_db -c "
  SELECT indexname, indexdef 
  FROM pg_indexes 
  WHERE tablename = 'sales_orders';
"
```

---

### Fase 5: Validación E2E (2-3h)

**Objetivo:** Verificar flujo completo funciona.

#### 5.1 Script de prueba E2E

Ver: `/documentation/SALES_SERVICE_MIGRATION_PLAN.md` sección 4.1

```bash
chmod +x test-sales-skeleton-e2e.sh
./test-sales-skeleton-e2e.sh
```

#### 5.2 Checklist de validación

- [ ] POST /sales/orders crea orden
- [ ] Evento `sales.order.created` visible en EventBus
- [ ] POST /sales/orders/:id/confirm funciona
- [ ] Evento `sales.order.confirmed` visible en EventBus
- [ ] Ledger consume evento y crea `ledger_entry`
- [ ] GET /ledger/balance devuelve monto correcto
- [ ] POST /sales/pos crea venta
- [ ] Eventos `sales.pos.created` + `sales.pos.confirmed` publicados
- [ ] Ledger procesa venta POS correctamente
- [ ] Logs sin errores críticos
- [ ] Performance aceptable (<500ms por request)

---

## ⚠️ Rollback Plan

Si algo falla:

```bash
# 1. Detener sales-service
docker-compose stop sales-service

# 2. Restaurar backup DB
docker exec -i postgres psql -U postgres -c "DROP DATABASE sales_db;"
docker exec -i postgres psql -U postgres -c "CREATE DATABASE sales_db;"
cat backup_order_db_YYYYMMDD.sql | docker exec -i postgres psql -U postgres sales_db

# 3. Revertir renombramiento
cd services
mv sales-service order-service

# 4. Revertir git (si committeado)
git revert <commit-hash>

# 5. Reiniciar order-service
docker-compose up order-service
```

---

## 📝 Post-Migración

### Actualizar documentación

- [ ] Actualizar README.md del servicio
- [ ] Actualizar API docs (OpenAPI)
- [ ] Actualizar diagrama de arquitectura
- [ ] Actualizar /documentation/ERP_MERCADO_CERCANO_ARQUITECTURA_V1.md

### Actualizar CI/CD

- [ ] Actualizar pipelines (si existen)
- [ ] Actualizar docker-compose.prod.yml
- [ ] Actualizar scripts de deployment

### Comunicar cambios

- [ ] Notificar equipo frontend (cambio de endpoints)
- [ ] Actualizar Postman collection
- [ ] Actualizar variables de entorno en producción

---

## 🔒 Fuera de Alcance v0.1

❌ Numeración secuencial (order_number, pos_number)  
❌ Points of Sale (point_of_sale_id)  
❌ Fiscal integration (fiscal_status usable pero sin integración)  
❌ Optimistic locking (version existe pero no se valida)  
❌ Multi-moneda (solo ARS)  

Estos se abordarán en Hitos v0.2 - v1.0.

---

**Última actualización:** 2026-02-20  
**Próxima revisión:** Post-ejecución migración  
