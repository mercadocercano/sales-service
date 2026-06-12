# Documentación — sales-service

Índice navegable de la documentación del servicio de ventas (Mercado Cercano).
Para visión general, puertos, rutas y dependencias ver el [README raíz](../README.md).

## Architecture Decision Records

| ADR | Título | Estado | Fecha |
|-----|--------|--------|-------|
| [ADR-001](adr/ADR-001-operacion-atomica-stock.md) | Operación atómica de descuento de stock en creación de órdenes | Aceptado | 2026-02-20 |
| [ADR-002](adr/ADR-002-snapshot-historico-order-items.md) | Snapshot histórico inmutable de producto y variante en order_items | Aceptado | 2026-02-20 |

## API

El contrato de API se define en [`api-docs/openapi.yaml`](../api-docs/openapi.yaml).

## Hitos del servicio

Registros de implementación de cada hito (orden cronológico de versiones):

- [HITO v0.1 — Sales Skeleton (Event-Driven)](hitos/HITO_V0.1_IMPLEMENTATION.md)
- [HITO v0.2 — Renombramiento estructural order-service → sales-service](hitos/HITO_V0.2_RENAME_COMPLETE.md)
- [HITO v0.5 — HTTP Routes Alignment](hitos/HITO_V0.5_ROUTES_ALIGNMENT.md)
- [HITO v0.6 — Customer ID Real (integración customer-service)](hitos/HITO_V0.6_CUSTOMER_ID_REAL.md)
- [HITO D — Operación atómica de stock en order-service](hitos/HITO_D_INTEGRATION.md) · ver [ADR-001](adr/ADR-001-operacion-atomica-stock.md)
- [HITO ORD-02 — Snapshot histórico en órdenes](hitos/HITO_ORD-02_SNAPSHOT_HISTORICO.md) · ver [ADR-002](adr/ADR-002-snapshot-historico-order-items.md)
- [HITO ORD-02 — Resumen ejecutivo (Snapshot)](hitos/README_HITO_ORD-02.md)
- [HITO — POST /pos/sale devuelve DTO listo para imprimir](hitos/HITO_POS_TICKET_PRINTING.md)
- [POS Sale — Endpoint de venta directa](hitos/POS_SALE_README.md)
- [POS-SALE-02.BE — Progreso de implementación](hitos/POS_SALE_02_BE_PROGRESS.md)

## Guías

- [Guía de pruebas POS-SALE-02](guides/POS_SALE_02_TESTING_GUIDE.md)
- [Testing POS Sale Extended](guides/TEST_POS_SALE_EXTENDED.md)

## Runbooks

- [Migración order-service → sales-service](runbooks/SALES_SERVICE_MIGRATION.md)
