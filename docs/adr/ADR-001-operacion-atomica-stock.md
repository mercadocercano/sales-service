---
adr: ADR-001
status: accepted
skills:
  implement:
    - dev/hexagonal-go
  verify:
    - dev/go-hex-audit
    - dev/code-reviewer
    - dev/concurrency-transactions
---
# ADR-001: Operación atómica de descuento de stock en creación de órdenes

**Estado**: Aceptado
**Fecha**: 2026-02-20
**Contexto**: El flujo original validaba disponibilidad (`CheckAvailability`) y luego descontaba stock (`ProcessSale`) en pasos separados. Bajo concurrencia, dos transacciones podían leer el mismo stock disponible y vender ambas, produciendo sobreventa (stock negativo).

## Decisión

Adoptamos una operación atómica única por ítem, `ProcessSaleAtomic`, que dentro de una sola transacción ejecuta `SELECT ... FOR UPDATE` sobre la fila de stock, valida la cantidad disponible y registra el movimiento de venta. Si la validación falla, hace `ROLLBACK`. Ante fallo de un ítem o de la persistencia de la orden, se ejecuta compensación automática contra `POST /api/v1/compensate-sale` de stock-service. Los métodos previos (`CheckAvailability` + `ProcessSale`) quedan deprecados.

## Alternativas consideradas

| Opción | Por qué no |
|--------|-----------|
| Mantener check + process separados con reintentos | No elimina la race condition; solo reduce su probabilidad |
| Lock optimista (columna de versión) | Mayor tasa de reintentos bajo contención alta de POS; más complejo en el flujo multi-ítem |
| Lock distribuido (Redis) | Introduce dependencia y un punto de fallo extra para garantizar algo que el lock de fila de Postgres ya provee |

## Consecuencias

**Positivas**: Elimina la sobreventa garantizando consistencia de stock bajo concurrencia; el descuento es transaccional con rollback claro; compensación automática ante fallos parciales.
**Negativas / trade-offs**: `SELECT FOR UPDATE` serializa el acceso a la misma fila de stock, reduciendo concurrencia sobre un SKU caliente; obliga a mantener la lógica de compensación entre sales-service y stock-service.
**Neutral**: Los métodos antiguos permanecen marcados como deprecated para retrocompatibilidad durante la transición.
