# Hito ORD-02: Snapshot Histórico en Órdenes

## Objetivo
Guardar snapshot inmutable de producto y variante al momento de crear la orden para preservar auditoría histórica.

## Problema Resuelto
Anteriormente, `order_items` solo guardaba el SKU de la variante. Si el producto o variante cambiaba en PIM (nombre, precio, categoría, etc.), las órdenes históricas perdían contexto. No se podía saber qué precio tenía el producto cuando se vendió, ni qué características tenía.

## Solución Implementada

### 1. Migración de Base de Datos
**Archivo**: `migrations/003_add_snapshots_to_order_items.sql`

```sql
ALTER TABLE order_items 
ADD COLUMN product_snapshot JSONB,
ADD COLUMN variant_snapshot JSONB;
```

- **Migración evolutiva**: Solo agrega columnas nuevas, no rompe datos existentes
- **Campos JSONB**: Permiten almacenar el objeto completo de producto y variante
- **Índices GIN**: Optimizan queries sobre los campos JSONB

### 2. Actualización de Entidad OrderItem
**Archivo**: `src/order/domain/entity/order_item.go`

Agregado:
- Campos `ProductSnapshot` y `VariantSnapshot` tipo `json.RawMessage`
- Constructor `NewOrderItemWithSnapshots()` para crear items con datos inmutables

### 3. Cliente PIM
**Archivo**: `src/order/infrastructure/client/pim_client.go` (NUEVO)

Cliente HTTP que se comunica con PIM service vía Kong Gateway para:
- Obtener variante por SKU: `GET /pim/api/v1/variants/by-sku/{sku}`
- Obtener producto por ID: `GET /pim/api/v1/products/{id}`
- Método helper `GetSnapshotForSKU()`: Obtiene ambos datos y retorna JSON serializado

### 4. Actualización de CreateOrderUseCase
**Archivo**: `src/order/application/usecase/create_order.go`

Flujo actualizado:
```
1. Por cada item en el request:
   a. Consultar PIM para obtener snapshot de producto y variante
   b. Serializar ambos a JSON
   c. Crear OrderItem con snapshots incluidos
2. Crear Order (aggregate root)
3. Persistir en base de datos con snapshots
```

### 5. Repository Actualizado
**Archivo**: `src/order/infrastructure/persistence/order_postgres_repository.go`

- `Save()`: Persiste snapshots en INSERT
- `FindByID()`: Carga snapshots en SELECT
- `List()`: Incluye snapshots en listado paginado

### 6. Respuestas de API
**Archivos actualizados**:
- `src/order/application/response/get_order_response.go`
- `src/order/application/response/list_orders_response.go`

Los endpoints GET ahora retornan los snapshots completos:

```json
{
  "order_id": "uuid",
  "items": [
    {
      "item_id": "uuid",
      "sku": "PROD-001-VAR-RED",
      "quantity": 2,
      "product_snapshot": {
        "product_id": "uuid",
        "name": "Notebook Lenovo",
        "category_id": "uuid",
        "brand_id": "uuid"
      },
      "variant_snapshot": {
        "variant_id": "uuid",
        "name": "Rojo 15 pulgadas",
        "price": 999.99,
        "cost_price": 750.00,
        "attributes": {...}
      }
    }
  ]
}
```

## Archivos Modificados

### Nuevos
1. `migrations/003_add_snapshots_to_order_items.sql`
2. `src/order/infrastructure/client/pim_client.go`

### Modificados
1. `src/order/domain/entity/order_item.go`
2. `src/order/application/usecase/create_order.go`
3. `src/order/infrastructure/persistence/order_postgres_repository.go`
4. `src/order/infrastructure/controller/order_controller.go`
5. `src/order/application/response/get_order_response.go`
6. `src/order/application/response/list_orders_response.go`
7. `src/order/application/usecase/get_order.go`
8. `src/order/application/usecase/list_orders.go`
9. `main.go`

## Variables de Entorno

El cliente PIM usa las mismas variables que el cliente Stock:

```env
KONG_INTERNAL_URL=http://kong:8000  # Default para Docker
PIM_SERVICE_PATH=/pim               # Default
```

## Flujo Completo

```
Frontend/POS
    ↓
POST /api/v1/orders
    ↓
Kong Gateway
    ↓
order-service: CreateOrderUseCase
    ↓
    ├─→ PIMClient.GetSnapshotForSKU() ──→ Kong ──→ pim-service
    │                                         ↓
    │                                    Retorna producto + variante
    │                                         ↓
    └─→ Crear OrderItem con snapshots ───────┘
    ↓
OrderPostgresRepository.Save()
    ↓
INSERT INTO order_items (
    item_id, order_id, sku, quantity,
    product_snapshot,  -- ← JSON completo del producto
    variant_snapshot   -- ← JSON completo de la variante
)
```

## Criterios de Cierre ✅

- [x] **Cada order_item guarda snapshot inmutable**: Campos JSONB agregados y poblados
- [x] **Cambios futuros en PIM no afectan órdenes viejas**: Los snapshots son inmutables al momento de creación
- [x] **No se rompe ningún flujo existente**: Migración evolutiva, campos opcionales en DTOs
- [x] **POS no se ve afectado**: Solo cambios en order-service, POS sigue llamando al mismo endpoint

## Beneficios

1. **Auditoría histórica completa**: Cada orden preserva exactamente cómo era el producto cuando se vendió
2. **Reportes precisos**: Se pueden generar reportes de ventas con precios y nombres históricos
3. **Sin dependencias runtime**: No requiere consultar PIM para ver órdenes viejas
4. **Flexibilidad**: El JSONB permite almacenar cualquier estructura de producto/variante
5. **Retrocompatibilidad**: Órdenes viejas sin snapshots siguen funcionando (campos NULL)

## Pruebas Sugeridas

### 1. Crear orden y verificar snapshots
```bash
curl -X POST http://localhost:8001/order/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <tenant_uuid>" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "items": [
      {"sku": "PROD-001-VAR-RED", "quantity": 2}
    ]
  }'
```

### 2. Obtener orden y verificar snapshots
```bash
curl http://localhost:8001/order/api/v1/orders/<order_id> \
  -H "X-Tenant-ID: <tenant_uuid>" \
  -H "Authorization: Bearer <token>"
```

Verificar que la respuesta incluya `product_snapshot` y `variant_snapshot`.

### 3. Cambiar producto en PIM
Actualizar nombre/precio del producto en PIM.

### 4. Verificar inmutabilidad
Volver a consultar la orden vieja y confirmar que los snapshots NO cambiaron.

## Próximos Pasos (Fuera de Alcance)

- ORD-03: Calcular totales de orden usando snapshots
- ORD-04: Dashboard de reportes históricos con snapshots
- ORD-05: Exportar órdenes con datos completos para contabilidad
