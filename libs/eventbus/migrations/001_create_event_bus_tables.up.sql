-- Tabla principal de eventos (append-only)
CREATE TABLE IF NOT EXISTS event_bus (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(100) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload_json JSONB NOT NULL,
    occurred_at TIMESTAMP NOT NULL,
    published_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Índices para mejorar performance en queries comunes
CREATE INDEX IF NOT EXISTS idx_event_bus_aggregate ON event_bus(aggregate_type, aggregate_id);
CREATE INDEX IF NOT EXISTS idx_event_bus_event_type ON event_bus(event_type);
CREATE INDEX IF NOT EXISTS idx_event_bus_occurred_at ON event_bus(occurred_at);

-- Tabla de consumidores (tracking de procesamiento)
CREATE TABLE IF NOT EXISTS event_consumers (
    event_id UUID NOT NULL,
    consumer_name VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, consumer_name)
);

-- Índices para queries de eventos no procesados
CREATE INDEX IF NOT EXISTS idx_event_consumers_status ON event_consumers(consumer_name, status);
CREATE INDEX IF NOT EXISTS idx_event_consumers_retry ON event_consumers(consumer_name, retry_count);

-- Comentarios para documentación
COMMENT ON TABLE event_bus IS 'Almacenamiento append-only de eventos de dominio';
COMMENT ON TABLE event_consumers IS 'Tracking de procesamiento de eventos por consumidor';
COMMENT ON COLUMN event_bus.aggregate_type IS 'Tipo de agregado (ej: sale, inventory, ledger)';
COMMENT ON COLUMN event_bus.aggregate_id IS 'ID del agregado que generó el evento';
COMMENT ON COLUMN event_bus.event_type IS 'Tipo de evento (ej: sale.created, inventory.updated)';
COMMENT ON COLUMN event_bus.payload_json IS 'Payload del evento en formato JSON';
COMMENT ON COLUMN event_consumers.status IS 'Estado: pending, processed, failed, retrying';
COMMENT ON COLUMN event_consumers.retry_count IS 'Número de reintentos de procesamiento';
