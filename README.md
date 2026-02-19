# Sales Service

**Version:** v0.2.0  
**Anteriormente:** order-service (renombrado en HITO v0.2)  
**Puerto:** 8120 (externo) → 8080 (interno Docker)  
**Lenguaje:** Go 1.22+  
**Estado:** ✅ DESARROLLO - EventBus Integration Completo  

---

## 📋 Descripción

Microservicio de gestión de ventas para el ERP "Mercado Cercano". Gestiona órdenes de venta (sales_orders) y ventas POS (pos_sales) con publicación de eventos al EventBus para integración con ledger-service.

**Responsabilidades:**

- Gestión de **Sales Orders** (órdenes diferidas)
- Gestión de **POS Sales** (ventas mostrador inmediatas)
- Publicación de eventos de ventas (`sales.order.confirmed`, `sales.pos.confirmed`)
- Integración con Stock Service (descuento atómico)
- Integración con PIM Service (snapshots de productos)
- Multi-tenant con validación estricta

---

## 🏗️ Arquitectura

### Hexagonal + DDD

```
src/
├── sales/               # Módulo principal (renombrado de order/)
│   ├── domain/
│   │   ├── entity/      # Order, OrderItem, PosSale, PosSaleItem
│   │   └── port/        # Interfaces de repositorios
│   ├── application/
│   │   ├── usecase/     # Casos de uso
│   │   ├── request/     # DTOs entrada
│   │   └── response/    # DTOs salida
│   └── infrastructure/
│       ├── controller/  # HTTP handlers
│       ├── persistence/ # PostgreSQL repositories
│       └── client/      # Clientes externos (Stock, PIM)
└── shared/             # Componentes compartidos
```

---

## 🔌 Endpoints

### Sales Orders

```bash
GET    /api/v1/orders              # Listar órdenes
POST   /api/v1/orders              # Crear orden
GET    /api/v1/orders/:id          # Obtener orden
POST   /api/v1/orders/:id/confirm  # Confirmar orden → publica evento
POST   /api/v1/orders/:id/cancel   # Cancelar orden
```

### POS Sales

```bash
POST   /api/v1/pos/sale            # Crear venta POS → publica evento
GET    /api/v1/pos/sales           # Listar ventas POS
```

### Reportes

```bash
GET    /api/v1/reports/daily?date=YYYY-MM-DD
```

---

## 🔗 Integraciones

### EventBus (Publisher)

**Eventos publicados:**
- `sales.order.confirmed` - Al confirmar orden
- `sales.pos.confirmed` - Al crear venta POS

**Formato:** EventEnvelope según contrato v1.0

**Implementación:** Librería `eventbus` compartida (`libs/eventbus`)

### Stock Service

**Operaciones:**
- `POST /api/v1/stock/process-sale-atomic` - Descuento atómico
- `POST /api/v1/stock/compensate/:id` - Reversión

**Vía:** Kong Gateway (puerto 8001)

### PIM Service

**Operaciones:**
- `GET /api/v1/products/:id/snapshot` - Snapshots inmutables
- `GET /api/v1/variants/by-sku/:sku` - Datos de variantes

**Vía:** Kong Gateway (puerto 8001)

---

## 💾 Base de Datos

### Tablas

```sql
-- Órdenes de venta (tabla legacy, será sales_orders en v0.3)
orders (
    order_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    sku VARCHAR(255),
    quantity INT,
    status VARCHAR(50),  -- CREATED, CONFIRMED, CANCELED
    created_at TIMESTAMP
)

-- Items de órdenes
order_items (
    item_id UUID PRIMARY KEY,
    order_id UUID,
    sku VARCHAR(255),
    quantity INT,
    product_snapshot JSONB,
    variant_snapshot JSONB
)

-- Ventas POS
pos_sales (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    customer_id UUID,
    payment_method_id UUID NOT NULL,
    total_amount DECIMAL,
    discount_amount DECIMAL,
    final_amount DECIMAL,
    amount_paid DECIMAL,
    change DECIMAL,
    currency VARCHAR(3),
    created_at TIMESTAMP
)

-- Items de ventas POS
pos_sale_items (
    id UUID PRIMARY KEY,
    pos_sale_id UUID,
    sku VARCHAR(255),
    product_name VARCHAR(255),
    quantity DECIMAL,
    unit_price DECIMAL,
    subtotal DECIMAL,
    stock_entry_id UUID
)
```

---

## 🚀 Desarrollo

### Compilar

```bash
cd services/sales-service
go mod tidy
go build .
```

### Ejecutar localmente

```bash
DB_HOST=localhost \
DB_PORT=5432 \
DB_USER=postgres \
DB_PASSWORD=postgres \
DB_NAME=order_db \
EVENTBUS_DB_HOST=localhost \
EVENTBUS_DB_PORT=5432 \
EVENTBUS_DB_USER=postgres \
EVENTBUS_DB_PASSWORD=postgres \
EVENTBUS_DB_NAME=eventbus \
PORT=8123 \
./sales
```

### Ejecutar con Docker

```bash
docker-compose up sales-service
```

---

## 🧪 Testing

### Test Manual

```bash
# 1. Crear orden
curl -X POST http://localhost:8123/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "items": [
      {"sku": "TEST-SKU", "quantity": 1}
    ]
  }'

# 2. Confirmar orden
curl -X POST http://localhost:8123/api/v1/orders/<ORDER_ID>/confirm \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{"reference": "TEST"}'

# 3. Verificar evento publicado
docker exec mc-postgres psql -U postgres -d eventbus \
  -c "SELECT event_type, aggregate_id FROM event_bus ORDER BY occurred_at DESC LIMIT 1;"

# 4. Verificar ledger entry
docker exec mc-postgres psql -U postgres -d ledger_db \
  -c "SELECT document_type, debit_base FROM ledger_entries ORDER BY created_at DESC LIMIT 1;"
```

---

## 📊 Hitos Completados

### HITO v0.1: EventBus Integration

**Duración:** 8 horas  
**Estado:** ✅ CERRADO  

**Implementado:**
- ✅ Librería `eventbus` integrada
- ✅ `PublishEventUseCase` inicializado
- ✅ Publicación de `sales.order.confirmed`
- ✅ Publicación de `sales.pos.confirmed`
- ✅ Ledger-service consume eventos
- ✅ Ledger entries creados correctamente

**Evidencias:**
- Eventos en `event_bus` table
- Entries en `ledger_entries` table
- Balance correcto

### HITO v0.2: Renombramiento Estructural

**Duración:** 3 horas  
**Estado:** ✅ CERRADO  

**Implementado:**
- ✅ Directorio renombrado: `order-service/` → `sales-service/`
- ✅ Module renombrado: `module order` → `module sales`
- ✅ Estructura renombrada: `src/order/` → `src/sales/`
- ✅ Imports actualizados (masivo)
- ✅ Docker Compose actualizado
- ✅ Kong Gateway actualizado
- ✅ Compilación exitosa
- ✅ Flujo E2E validado post-rename

**Evidencias:**
- ✅ `go build` sin errores
- ✅ Health endpoint OK
- ✅ Confirm order funcional
- ✅ Evento publicado
- ✅ Ledger entry con monto 250.00

---

## 🔜 Próximos Hitos (Backlog)

### HITO v0.3: DB Schema Alignment

- Migración 008: `orders` → `sales_orders`
- Migración 009: Extender `pos_sales`
- Agregar campos: `order_number`, `customer_id`, `fiscal_status`, `invoice_id`, `version`

**Estimación:** 4-6 horas

### HITO v0.4: Rutas HTTP

- `/api/v1/orders` → `/api/v1/sales/orders`
- `/api/v1/pos/sale` → `/api/v1/sales/pos`
- Kong routes update

**Estimación:** 2-3 horas

### HITO v1.0: Production Ready

- Numeración secuencial
- Points of Sale
- Customer ID real
- Optimistic locking
- Fiscal integration

**Estimación:** 2-3 semanas

---

## 📚 Documentación

- **Ficha técnica:** `documentation/components/sales-service.md`
- **Implementación v0.1:** `HITO_V0.1_IMPLEMENTATION.md`
- **Cierre v0.2:** `HITO_V0.2_RENAME_COMPLETE.md`
- **Arquitectura ERP:** `documentation/ERP_MERCADO_CERCANO_ARQUITECTURA_V1.md`

---

## ⚠️ Notas Importantes

### Nombres Legacy (temporales)

Por retrocompatibilidad, se mantienen hasta v0.3:

- Tablas: `orders`, `order_items` (serán `sales_orders`, `sales_order_items`)
- Rutas: `/api/v1/orders` (será `/api/v1/sales/orders`)
- DB: `order_db` (será `sales_db`)

### EventBus Dependency

Este servicio depende de:
- `libs/eventbus` - Librería compartida
- `eventbus` DB - Persistencia de eventos
- `ledger-service` - Consumer de eventos

---

**Última actualización:** 2026-02-20  
**Mantenido por:** Backend Team  
**Estado:** ✅ READY FOR DEVELOPMENT  
