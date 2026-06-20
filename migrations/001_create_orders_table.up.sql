-- Migration: Create orders table (minimal)
-- HITO: Creación mínima de orden

CREATE TABLE IF NOT EXISTS orders (
    order_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    sku VARCHAR(255) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status VARCHAR(50) NOT NULL DEFAULT 'CREATED',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index por tenant para consultas eficientes
CREATE INDEX IF NOT EXISTS idx_orders_tenant_id ON orders(tenant_id);

-- Index por tenant + created_at para ordenamiento
CREATE INDEX IF NOT EXISTS idx_orders_tenant_created ON orders(tenant_id, created_at DESC);

-- Constraint para validar status
ALTER TABLE orders ADD CONSTRAINT orders_status_check 
    CHECK (status IN ('CREATED'));

COMMENT ON TABLE orders IS 'Tabla de órdenes mínima - HITO creación básica';
COMMENT ON COLUMN orders.order_id IS 'ID único de la orden';
COMMENT ON COLUMN orders.tenant_id IS 'ID del tenant (multitenant)';
COMMENT ON COLUMN orders.sku IS 'SKU del producto (variant_sku)';
COMMENT ON COLUMN orders.quantity IS 'Cantidad solicitada';
COMMENT ON COLUMN orders.status IS 'Estado de la orden (solo CREATED por ahora)';
