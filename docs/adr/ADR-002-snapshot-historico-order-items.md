---
adr: ADR-002
status: accepted
skills:
  implement:
    - dev/hexagonal-go
  verify:
    - dev/code-reviewer
    - dev/postgres-data-modeling
---
# ADR-002: Snapshot histórico inmutable de producto y variante en order_items

**Estado**: Aceptado
**Fecha**: 2026-02-20
**Contexto**: `order_items` solo guardaba el SKU de la variante. Cuando el producto o la variante cambiaban en PIM (nombre, precio, categoría), las órdenes históricas perdían el contexto del momento de la venta y no era posible auditar a qué precio o con qué características se vendió.

## Decisión

Guardamos un snapshot inmutable del producto y la variante en cada `order_item` al momento de crear la orden. Agregamos las columnas `product_snapshot` y `variant_snapshot` (tipo `JSONB`, con índices GIN) vía migración evolutiva `003_add_snapshots_to_order_items.sql`. Al crear la orden, `CreateOrderUseCase` consulta PIM por SKU/ID a través de Kong, serializa producto y variante a JSON y los persiste junto con el ítem mediante el constructor `NewOrderItemWithSnapshots()`.

## Alternativas consideradas

| Opción | Por qué no |
|--------|-----------|
| Guardar solo el SKU y reconsultar PIM al leer la orden | PIM refleja el estado actual, no el del momento de la venta; rompe la auditoría histórica |
| Versionar productos/variantes en PIM | Mayor complejidad y acoplamiento; resuelve en PIM un requisito que es propio del dominio de ventas |
| Duplicar campos escalares (nombre, precio) en columnas planas | Rígido ante cambios de esquema de PIM; JSONB captura el objeto completo sin migraciones por campo |

## Consecuencias

**Positivas**: Auditoría histórica inmutable; las órdenes preservan el contexto exacto de la venta independientemente de cambios posteriores en PIM; JSONB + GIN permite consultar sobre los snapshots.
**Negativas / trade-offs**: Cada creación de orden agrega llamadas HTTP a PIM (latencia y dependencia en el camino crítico); mayor tamaño de almacenamiento por duplicación de datos; los snapshots no se actualizan por diseño.
**Neutral**: Migración solo aditiva — no rompe datos existentes; las órdenes previas quedan con snapshots nulos.
