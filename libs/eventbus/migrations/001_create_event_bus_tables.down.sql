-- Rollback: eliminar tablas e índices
DROP INDEX IF EXISTS idx_event_consumers_retry;
DROP INDEX IF EXISTS idx_event_consumers_status;
DROP TABLE IF EXISTS event_consumers;

DROP INDEX IF EXISTS idx_event_bus_occurred_at;
DROP INDEX IF EXISTS idx_event_bus_event_type;
DROP INDEX IF EXISTS idx_event_bus_aggregate;
DROP TABLE IF EXISTS event_bus;
