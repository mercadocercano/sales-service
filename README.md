# Sales Service

**Version:** v0.5.0  
**HITO:** HTTP Routes Alignment  
**Anteriormente:** order-service (renombrado en HITO v0.2)  
**Puerto:** 8120 (externo) → 8080 (interno Docker)  
**Lenguaje:** Go 1.22+  
**Estado:** ✅ PRODUCCIÓN (H6-A.2) - Core Comercial Operativo

## Documentación

Ver [`docs/`](docs/README.md) para ADRs, hitos, guías de testing y runbooks operativos.

## Estado en producción (Mar 2026)

| Aspecto | Estado |
|---------|--------|
| **K8s** | ✅ Desplegado en `k8s/sales/` |
| **Kong** | Ruta `/sales/` |
| **DB** | `order_db`, `payment_method_db` |
| **Dependencias** | stock-service, customer-service (vía Kong) |
| **Endpoints clave** | `POST /api/v1/sales/pos`, `GET /api/v1/sales/pos` |

Ver: `documentation/PROJECT_STATUS_MAR_2026.md`  

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

### ✨ HITO v0.5 - Nuevas Rutas (Arquitectónicamente Alineadas)

#### Sales Orders

```bash
GET    /api/v1/sales/orders              # Listar órdenes
POST   /api/v1/sales/orders              # Crear orden
GET    /api/v1/sales/orders/:id          # Obtener orden
POST   /api/v1/sales/orders/:id/confirm  # Confirmar orden → publica evento
POST   /api/v1/sales/orders/:id/cancel   # Cancelar orden
```

#### POS Sales

```bash
POST   /api/v1/sales/pos                 # Crear venta POS → publica evento
GET    /api/v1/sales/pos                 # Listar ventas POS
```

### ⚠️ Rutas Legacy (Eliminadas en v0.5)

```bash
# ❌ DEPRECADAS - NO USAR
/api/v1/orders/*
/api/v1/pos/sale
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

**Database:** `order_db` (PostgreSQL)

### Tablas Actuales (v0.5)

```sql
-- Órdenes de venta
sales_orders (
    order_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    order_number VARCHAR(50),            -- Numeración secuencial (ORD-YYYYMMDD-NNNN)
    status VARCHAR(50),                  -- CREATED, CONFIRMED, CANCELED
    customer_id UUID,
    total_amount DECIMAL(15,2),
    currency VARCHAR(3) DEFAULT 'ARS',
    version INT DEFAULT 1,               -- Optimistic locking
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)

-- Items de órdenes
sales_order_items (
    item_id UUID PRIMARY KEY,
    order_id UUID REFERENCES sales_orders(order_id),
    sku VARCHAR(255) NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(15,2),
    subtotal DECIMAL(15,2),
    product_snapshot JSONB,              -- Snapshot inmutable de PIM
    variant_snapshot JSONB,              -- Snapshot inmutable de variante
    stock_entry_id UUID,
    created_at TIMESTAMP
)

-- Ventas POS
pos_sales (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    sale_number VARCHAR(50),             -- Numeración secuencial
    customer_id UUID,
    payment_method_id UUID NOT NULL,
    subtotal_amount DECIMAL(15,2),
    discount_amount DECIMAL(15,2),
    final_amount DECIMAL(15,2),
    amount_paid DECIMAL(15,2),
    change DECIMAL(15,2),
    currency VARCHAR(3) DEFAULT 'ARS',
    notes TEXT,
    created_at TIMESTAMP
)

-- Items de ventas POS
pos_sale_items (
    id UUID PRIMARY KEY,
    pos_sale_id UUID REFERENCES pos_sales(id),
    sku VARCHAR(255) NOT NULL,
    product_name VARCHAR(255),
    quantity DECIMAL(10,3) NOT NULL,
    unit_price DECIMAL(15,2) NOT NULL,
    subtotal DECIMAL(15,2) NOT NULL,
    stock_entry_id UUID,
    created_at TIMESTAMP
)

-- Secuencias de numeración
number_sequences (
    sequence_name VARCHAR(50) PRIMARY KEY,
    tenant_id UUID NOT NULL,
    current_value INT DEFAULT 0,
    prefix VARCHAR(20),
    last_updated TIMESTAMP
)
```

### Índices

```sql
CREATE INDEX idx_sales_orders_tenant ON sales_orders(tenant_id);
CREATE INDEX idx_sales_orders_number ON sales_orders(order_number);
CREATE INDEX idx_sales_orders_status ON sales_orders(status);
CREATE INDEX idx_pos_sales_tenant ON pos_sales(tenant_id);
CREATE INDEX idx_sales_order_items_product_snapshot ON sales_order_items USING GIN (product_snapshot);
CREATE INDEX idx_sales_order_items_variant_snapshot ON sales_order_items USING GIN (variant_snapshot);
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

### Test Manual (v0.5 - Nuevas Rutas)

```bash
# 1. Crear orden
curl -X POST http://localhost:8120/api/v1/sales/orders \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{
    "items": [
      {"sku": "TEST-SKU-001", "quantity": 2}
    ]
  }'

# 2. Confirmar orden
curl -X POST http://localhost:8120/api/v1/sales/orders/<ORDER_ID>/confirm \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{"reference": "REF-TEST-001"}'

# 3. Venta POS
curl -X POST http://localhost:8120/api/v1/sales/pos \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 550e8400-e29b-41d4-a716-446655440000" \
  -d '{
    "items": [
      {"sku": "COCA-1L", "quantity": 1, "unit_price": "1500.00"}
    ],
    "payment_method_id": "b0000000-0000-0000-0000-000000000001",
    "discount_amount": "0",
    "amount_paid": "2000.00"
  }'

# 4. Verificar evento publicado
docker exec mc-postgres psql -U postgres -d eventbus \
  -c "SELECT event_type, aggregate_id FROM event_bus ORDER BY occurred_at DESC LIMIT 1;"

# 5. Verificar ledger entry
docker exec mc-postgres psql -U postgres -d ledger_db \
  -c "SELECT document_type, document_number, debit_base FROM ledger_entries ORDER BY created_at DESC LIMIT 1;"
```

### Scripts de Testing

```bash
# Test snapshot feature
./scripts/test-snapshot-feature.sh

# Test POS sale completo
./scripts/test-pos-sale-complete-dto.sh

# Test rápido POS
./test-pos-sale.sh
```

---

## 📊 Hitos Completados

### HITO v0.5: HTTP Routes Alignment ⭐ ACTUAL

**Duración:** 1.5 horas  
**Fecha:** 2026-02-20  
**Estado:** ✅ CERRADO  

**Implementado:**
- ✅ Rutas HTTP alineadas: `/api/v1/sales/*`
- ✅ Hard cut: rutas legacy eliminadas (404)
- ✅ Kong Gateway actualizado
- ✅ Scripts de testing actualizados
- ✅ Coherencia transversal completa

**Evidencias:**
- `POST /api/v1/sales/orders` funciona
- `POST /api/v1/orders` → 404
- `POST /api/v1/sales/pos` funciona
- `POST /api/v1/pos/sale` → 404

**Documentación:** [`docs/hitos/HITO_V0.5_ROUTES_ALIGNMENT.md`](docs/hitos/HITO_V0.5_ROUTES_ALIGNMENT.md)

---

### HITO v0.4: Numeración Secuencial

**Duración:** 6 horas  
**Fecha:** 2026-02-19  
**Estado:** ✅ CERRADO  

**Implementado:**
- ✅ Tabla `number_sequences`
- ✅ `SequenceService` con concurrencia segura
- ✅ Numeración: `ORD-YYYYMMDD-NNNN`
- ✅ Campo `order_number` en `sales_orders`
- ✅ Asignación automática al confirmar orden

**Evidencias:**
- Numeración secuencial correcta
- Sin colisiones bajo concurrencia
- Optimistic locking validado

---

### HITO v0.3: DB Schema Alignment

**Duración:** 5 horas  
**Fecha:** 2026-02-18  
**Estado:** ✅ CERRADO  

**Implementado:**
- ✅ Migración: `orders` → `sales_orders`
- ✅ Migración: `order_items` → `sales_order_items`
- ✅ Campos agregados: `order_number`, `version`, `customer_id`
- ✅ Índices optimizados
- ✅ Repositorios actualizados

---

### HITO v0.2: Renombramiento Estructural

**Duración:** 3 horas  
**Fecha:** 2026-02-17  
**Estado:** ✅ CERRADO  

**Implementado:**
- ✅ Directorio: `order-service/` → `sales-service/`
- ✅ Module: `module order` → `module sales`
- ✅ Estructura: `src/order/` → `src/sales/`
- ✅ Imports actualizados (masivo)
- ✅ Docker Compose + Kong actualizados

**Documentación:** [`docs/hitos/HITO_V0.2_RENAME_COMPLETE.md`](docs/hitos/HITO_V0.2_RENAME_COMPLETE.md)

---

### HITO v0.1: EventBus Integration

**Duración:** 8 horas  
**Fecha:** 2026-02-16  
**Estado:** ✅ CERRADO  

**Implementado:**
- ✅ Librería `eventbus` integrada
- ✅ Publicación: `sales.order.confirmed`
- ✅ Publicación: `sales.pos.confirmed`
- ✅ Integración con ledger-service
- ✅ Ledger entries automáticos

**Documentación:** [`docs/hitos/HITO_V0.1_IMPLEMENTATION.md`](docs/hitos/HITO_V0.1_IMPLEMENTATION.md)

---

## 📚 Documentación Técnica

Índice completo en [`docs/`](docs/README.md): ADRs, hitos, guías de testing y runbooks.

### Hitos del Servicio
- **HITO v0.5:** [`docs/hitos/HITO_V0.5_ROUTES_ALIGNMENT.md`](docs/hitos/HITO_V0.5_ROUTES_ALIGNMENT.md)
- **HITO v0.2:** [`docs/hitos/HITO_V0.2_RENAME_COMPLETE.md`](docs/hitos/HITO_V0.2_RENAME_COMPLETE.md)
- **HITO v0.1:** [`docs/hitos/HITO_V0.1_IMPLEMENTATION.md`](docs/hitos/HITO_V0.1_IMPLEMENTATION.md)

### Arquitectura
- **Decisiones (ADR):** [`docs/adr/`](docs/adr/)
- **Arquitectura ERP:** `../../documentation/ERP_MERCADO_CERCANO_ARQUITECTURA_V1.md`
- **Ficha técnica:** `../../documentation/components/sales-service.md`

### Testing Guides
- **Snapshot Feature:** [`docs/hitos/README_HITO_ORD-02.md`](docs/hitos/README_HITO_ORD-02.md)
- **POS Testing:** [`docs/guides/POS_SALE_02_TESTING_GUIDE.md`](docs/guides/POS_SALE_02_TESTING_GUIDE.md)

---

## 🔗 Dependencias

### Servicios Externos
- **Stock Service** - Descuento atómico de inventario
- **PIM Service** - Snapshots de productos/variantes
- **Ledger Service** - Consumer de eventos de ventas

### Infraestructura
- **EventBus DB** (`eventbus`) - Persistencia de eventos
- **Payment Method DB** (`payment_method_db`) - Cache de métodos de pago
- **Kong Gateway** (puerto 8001) - Routing y autenticación

---

## 🎯 Estado Actual (v0.5)

### Coherencia Arquitectónica ✅

| Capa | Naming |
|------|--------|
| Código | `sales/` |
| Base de Datos | `sales_orders`, `pos_sales` |
| Eventos | `sales.order.confirmed` |
| Repositorio | `sales-service` |
| Rutas HTTP | `/api/v1/sales/*` |
| Kong Gateway | `/sales/` |

**Sin híbridos. Sin deuda técnica de naming.**

---

**Versión:** v0.5.0  
**Última actualización:** 2026-02-20  
**Mantenido por:** Backend Team  
**Estado:** ✅ STABLE - READY FOR INTEGRATION  
