# Hito ORD-02: Snapshot Histórico en Órdenes

## 📋 Resumen Ejecutivo

**Objetivo**: Preservar auditoría histórica inmutable de productos y variantes al momento de crear órdenes.

**Status**: ✅ **COMPLETADO**

**Alcance**: Solo `order-service` y tabla `order_items`.

**Fuera de alcance**: POS, stock-service, eventos async, migraciones complejas.

## 🎯 Problema

Las órdenes solo guardaban el SKU de la variante. Si el producto cambiaba en PIM (precio, nombre, categoría), las órdenes históricas perdían contexto. No era posible saber:

- ¿Qué precio tenía el producto cuando se vendió?
- ¿Qué nombre tenía el producto en ese momento?
- ¿En qué categoría estaba clasificado?
- ¿Qué atributos tenía la variante?

## ✅ Solución Implementada

### Diseño MÁS Simple

1. **Agregar 2 campos JSONB a `order_items`**:
   - `product_snapshot`: JSON completo del producto
   - `variant_snapshot`: JSON completo de la variante

2. **Poblar snapshots al crear la orden**:
   - Consultar PIM service vía Kong
   - Serializar producto y variante a JSON
   - Guardar en base de datos

3. **Retornar snapshots en APIs de lectura**:
   - `GET /api/v1/orders/:id`
   - `GET /api/v1/orders` (listado)

## 🏗️ Arquitectura

```
CreateOrder Request
       ↓
   Controller
       ↓
CreateOrderUseCase ──┬──→ PIMClient.GetSnapshotForSKU()
       ↓             │         ↓
       │             │    Kong Gateway
       │             │         ↓
       │             │    pim-service
       │             │         ↓
       │             └────  Retorna producto + variante JSON
       ↓
NewOrderItemWithSnapshots(sku, qty, productJSON, variantJSON)
       ↓
OrderPostgresRepository.Save()
       ↓
INSERT order_items (
    product_snapshot JSONB,  -- Inmutable
    variant_snapshot JSONB   -- Inmutable
)
```

## 📁 Estructura de Archivos

### Nuevos
```
order-service/
├── migrations/
│   └── 003_add_snapshots_to_order_items.sql  ← Nueva migración
├── src/order/infrastructure/client/
│   └── pim_client.go                         ← Cliente HTTP para PIM
├── scripts/
│   ├── run-migration-003.sh                  ← Script de migración
│   └── test-snapshot-feature.sh              ← Tests E2E
├── HITO_ORD-02_SNAPSHOT_HISTORICO.md         ← Documentación del hito
└── README_HITO_ORD-02.md                     ← Este archivo
```

### Modificados
```
src/order/
├── domain/entity/
│   └── order_item.go                         ← Agregados campos snapshot
├── application/
│   ├── usecase/
│   │   ├── create_order.go                   ← Llama a PIMClient
│   │   ├── get_order.go                      ← Retorna snapshots
│   │   └── list_orders.go                    ← Retorna snapshots
│   └── response/
│       ├── get_order_response.go             ← DTOs con snapshots
│       └── list_orders_response.go
└── infrastructure/
    ├── persistence/
    │   └── order_postgres_repository.go      ← INSERT/SELECT con snapshots
    └── controller/
        └── order_controller.go               ← Pasa authToken
main.go                                        ← Inyecta PIMClient
```

## 🚀 Instalación y Deployment

### 1. Ejecutar Migración

```bash
# En desarrollo
cd services/order-service
./scripts/run-migration-003.sh

# En producción con Docker
docker exec -it order-service bash
cd /app
psql $DATABASE_URL -f migrations/003_add_snapshots_to_order_items.sql
```

### 2. Reconstruir Servicio

```bash
# Rebuild con Docker
cd services/order-service
docker build --no-cache -t order-service:latest .

# O usando Makefile del proyecto
make dev-restart
```

### 3. Verificar Deployment

```bash
# Health check
curl http://localhost:8001/order/api/v1/health

# Verificar tabla
docker exec -it order-db psql -U postgres -d order_db -c "\d order_items"
```

## 🧪 Testing

### Test Manual Básico

```bash
# 1. Crear orden
curl -X POST http://localhost:8001/order/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: <tenant_uuid>" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "items": [
      {"sku": "PROD-001-VAR-RED", "quantity": 2}
    ]
  }'

# 2. Obtener orden y verificar snapshots
curl http://localhost:8001/order/api/v1/orders/<order_id> \
  -H "X-Tenant-ID: <tenant_uuid>" \
  -H "Authorization: Bearer <token>" | jq '.items[0]'

# Verificar que existan:
# - product_snapshot
# - variant_snapshot
```

### Test E2E Automatizado

```bash
TENANT_ID=<uuid> \
AUTH_TOKEN=<token> \
TEST_SKU=PROD-001-VAR-RED \
./scripts/test-snapshot-feature.sh
```

### Casos de Prueba

1. ✅ Crear orden con SKU válido → Snapshots guardados
2. ✅ Crear orden con SKU inválido → Error 404 de PIM
3. ✅ Obtener orden → Snapshots retornados
4. ✅ Listar órdenes → Snapshots incluidos
5. ✅ Cambiar producto en PIM → Orden vieja mantiene snapshot original

## 📊 Ejemplo de Snapshot

### Request
```json
POST /api/v1/orders
{
  "items": [
    {"sku": "LENOVO-NB-15-RED", "quantity": 1}
  ]
}
```

### Response con Snapshots
```json
{
  "order_id": "550e8400-e29b-41d4-a716-446655440000",
  "items": [
    {
      "item_id": "650e8400-e29b-41d4-a716-446655440000",
      "sku": "LENOVO-NB-15-RED",
      "quantity": 1,
      "product_snapshot": {
        "product_id": "750e8400-e29b-41d4-a716-446655440000",
        "product_sku": "LENOVO-NB-15",
        "name": "Notebook Lenovo IdeaPad 15\"",
        "description": "Notebook i5 8GB RAM 256GB SSD",
        "category_id": "850e8400-e29b-41d4-a716-446655440000",
        "brand_id": "950e8400-e29b-41d4-a716-446655440000",
        "status": "active",
        "created_at": "2025-02-01T10:00:00Z"
      },
      "variant_snapshot": {
        "variant_id": "a50e8400-e29b-41d4-a716-446655440000",
        "product_id": "750e8400-e29b-41d4-a716-446655440000",
        "variant_sku": "LENOVO-NB-15-RED",
        "name": "Rojo",
        "price": 999.99,
        "cost_price": 750.00,
        "compare_price": 1199.99,
        "attributes": {
          "color": "Rojo",
          "size": "15 pulgadas"
        },
        "status": "active"
      }
    }
  ]
}
```

## 🔒 Variables de Entorno

```env
# Cliente PIM (usa misma configuración que Stock)
KONG_INTERNAL_URL=http://kong:8000
PIM_SERVICE_PATH=/pim
```

## ✅ Criterios de Cierre Cumplidos

- [x] Cada `order_item` guarda snapshot inmutable de producto y variante
- [x] Cambios futuros en PIM NO afectan órdenes viejas
- [x] No se rompe ningún flujo existente (migración evolutiva)
- [x] POS no se ve afectado (sin cambios en frontend)

## 🎁 Beneficios

1. **Auditoría Completa**: Cada orden preserva estado exacto del producto al venderse
2. **Reportes Precisos**: Informes históricos con precios y datos correctos
3. **Sin Dependencias Runtime**: No requiere PIM para consultar órdenes viejas
4. **Flexibilidad**: JSONB permite cualquier estructura de producto
5. **Retrocompatible**: Órdenes sin snapshots siguen funcionando

## 🚧 Limitaciones Conocidas

1. **Tamaño de snapshot**: Si el producto tiene mucha metadata, el JSONB puede crecer
   - Solución: Indices GIN ya agregados para optimizar queries
2. **Órdenes viejas sin snapshots**: Las creadas antes de esta migración tendrán snapshots NULL
   - Solución: Script de backfill (fuera de alcance de este hito)
3. **PIM service debe estar disponible**: Si PIM falla, la creación de orden falla
   - Solución: Manejo de errores ya implementado, retorna 502 al frontend

## 📈 Próximos Pasos (Sugeridos)

1. **ORD-03**: Calcular totales de orden usando `variant_snapshot.price`
2. **ORD-04**: Dashboard de reportes históricos con snapshots
3. **ORD-05**: Exportar órdenes con datos completos para contabilidad
4. **ORD-06**: Script de backfill para órdenes antiguas sin snapshots

## 🔧 Troubleshooting

### Error: "variant not found"
- Verificar que el SKU existe en PIM
- Verificar que PIM service está corriendo
- Verificar configuración de Kong para ruta `/pim`

### Error: "pim-service returned status 502"
- Verificar conectividad Kong → PIM service
- Revisar logs de Kong: `docker logs kong`
- Revisar logs de PIM: `docker logs pim-service`

### Snapshots NULL en base de datos
- Migración no aplicada: Ejecutar `003_add_snapshots_to_order_items.sql`
- PIM client no inyectado: Verificar `main.go` línea 112

### Compilación falla
```bash
cd services/order-service
go mod tidy
go build .
```

## 📞 Soporte

Para dudas o issues:
1. Revisar logs: `docker logs order-service`
2. Verificar health: `GET /api/v1/health`
3. Consultar documentación completa: `HITO_ORD-02_SNAPSHOT_HISTORICO.md`

---

**Implementado por**: PM Técnico  
**Fecha**: 2025-02-08  
**Versión**: 1.0.0
