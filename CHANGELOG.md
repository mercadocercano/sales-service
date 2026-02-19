# Changelog - Order Service

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] - 2025-02-08

### Added - Hito ORD-02: Snapshot Histórico
- Campo `product_snapshot` (JSONB) en tabla `order_items`
- Campo `variant_snapshot` (JSONB) en tabla `order_items`
- Cliente HTTP `PIMClient` para consultar productos y variantes vía Kong
- Constructor `NewOrderItemWithSnapshots()` en entidad `OrderItem`
- Método `GetSnapshotForSKU()` en `PIMClient` para obtener datos inmutables
- Índices GIN en campos JSONB para optimizar queries
- Migración `003_add_snapshots_to_order_items.sql`
- Scripts de utilidad:
  - `scripts/run-migration-003.sh`
  - `scripts/test-snapshot-feature.sh`
- Documentación completa del hito:
  - `HITO_ORD-02_SNAPSHOT_HISTORICO.md`
  - `README_HITO_ORD-02.md`

### Changed
- `CreateOrderUseCase` ahora obtiene snapshots de PIM antes de guardar
- `OrderPostgresRepository.Save()` persiste snapshots en INSERT
- `OrderPostgresRepository.FindByID()` carga snapshots en SELECT
- `OrderPostgresRepository.List()` incluye snapshots en listado paginado
- `GetOrderResponse` incluye campos `product_snapshot` y `variant_snapshot`
- `OrderItemResponse` incluye campos `product_snapshot` y `variant_snapshot`
- `OrderController.CreateOrder()` pasa `authToken` al use case
- `main.go` inyecta `PIMClient` en `CreateOrderUseCase`

### Fixed
- Duplicación de struct `OrderItemResponse` en `list_orders_response.go` (consolidado en `get_order_response.go`)

### Migration
```sql
-- Run this migration before deploying v1.1.0
ALTER TABLE order_items 
ADD COLUMN product_snapshot JSONB,
ADD COLUMN variant_snapshot JSONB;

CREATE INDEX idx_order_items_product_snapshot ON order_items USING GIN (product_snapshot);
CREATE INDEX idx_order_items_variant_snapshot ON order_items USING GIN (variant_snapshot);
```

### Breaking Changes
**NONE** - This release is 100% backward compatible.

- Old orders without snapshots will have NULL values (handled gracefully)
- API responses include new optional fields (clients can ignore them)
- No changes to existing endpoints or request formats

## [1.0.0] - 2025-01-XX

### Added - Hito 3.0: Order Service Bootstrap
- Arquitectura hexagonal (domain, application, infrastructure)
- Entidad `Order` (aggregate root)
- Entidad `OrderItem` (entity dentro del aggregate)
- Repository pattern con PostgreSQL
- Casos de uso:
  - `CreateOrderUseCase`
  - `ConfirmOrderUseCase`
  - `CancelOrderUseCase`
  - `ListOrdersUseCase`
  - `GetOrderUseCase`
  - `ValidateStockUseCase`
  - `ReserveStockUseCase`
  - `ReleaseStockUseCase`
  - `POSSaleUseCase`
- Cliente HTTP `StockClient` para comunicación con stock-service
- Endpoints REST:
  - `POST /api/v1/orders` - Crear orden
  - `GET /api/v1/orders` - Listar órdenes (paginado)
  - `GET /api/v1/orders/:id` - Obtener orden
  - `POST /api/v1/orders/:id/confirm` - Confirmar orden
  - `POST /api/v1/orders/:id/cancel` - Cancelar orden
  - `POST /api/v1/orders/validate-stock` - Validar stock
  - `POST /api/v1/orders/reserve-stock` - Reservar stock
  - `POST /api/v1/orders/release-stock` - Liberar stock
  - `POST /api/v1/pos/sale` - Venta directa POS
- Health check endpoint
- Métricas Prometheus
- Migraciones de base de datos:
  - `001_create_orders_table.sql`
  - `002_create_order_items_table.sql`
- Multi-tenancy con header `X-Tenant-ID`
- Middlewares: Logger, Recovery, GZIP

### Technical Debt
- [ ] Tests unitarios pendientes
- [ ] Tests de integración pendientes
- [ ] Backfill de snapshots para órdenes antiguas (ORD-06)

---

## Versioning Guide

- **MAJOR** version: Breaking changes in APIs or database schema
- **MINOR** version: New features, backward compatible
- **PATCH** version: Bug fixes, no new features

## Migration Notes

Always run migrations before deploying new versions:
```bash
cd services/order-service
./scripts/run-migration-XXX.sh
```

Check migration status:
```bash
docker exec -it order-db psql -U postgres -d order_db -c "SELECT * FROM schema_migrations;"
```
