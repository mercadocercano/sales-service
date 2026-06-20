-- Migration 002: Refactorizar Order para soportar múltiples items
-- Patrón DDD: Order (Aggregate Root) + OrderItem (Entity)

-- 1. Crear tabla order_items
CREATE TABLE IF NOT EXISTS order_items (
    item_id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(order_id) ON DELETE CASCADE,
    sku VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index para búsqueda por orden
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);

-- Index para búsqueda por SKU
CREATE INDEX IF NOT EXISTS idx_order_items_sku ON order_items(sku);

-- 2. Migrar datos existentes de orders a order_items
-- Cada orden existente con sku/quantity se convierte en 1 item
INSERT INTO order_items (item_id, order_id, sku, quantity)
SELECT 
    gen_random_uuid() as item_id,
    order_id,
    sku,
    quantity
FROM orders
WHERE sku IS NOT NULL AND quantity IS NOT NULL;

-- 3. Eliminar columnas sku y quantity de orders
-- (NO se puede hacer en PostgreSQL de forma segura, se dejan como nullable por compatibilidad)
ALTER TABLE orders ALTER COLUMN sku DROP NOT NULL;
ALTER TABLE orders ALTER COLUMN quantity DROP NOT NULL;

COMMENT ON TABLE order_items IS 'Items de una orden - Patrón DDD Aggregate';
COMMENT ON COLUMN order_items.item_id IS 'ID único del item';
COMMENT ON COLUMN order_items.order_id IS 'ID de la orden padre (Aggregate Root)';
COMMENT ON COLUMN order_items.sku IS 'SKU del producto (variant_sku)';
COMMENT ON COLUMN order_items.quantity IS 'Cantidad del item';
