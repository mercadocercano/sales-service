-- ============================================================================
-- Migración 010: Clean Migration to Sales Schema
-- Fecha: 2026-02-20
-- Hito: v0.3 - DB Schema Alignment
-- Estrategia: DROP + CREATE (no backward compatibility)
-- ============================================================================

BEGIN;

-- ============================================================================
-- PASO 1: Eliminar tablas legacy (sin migración de datos)
-- ============================================================================

DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS orders CASCADE;

DO $$ BEGIN RAISE NOTICE 'Tablas legacy eliminadas: orders, order_items'; END $$;

-- ============================================================================
-- PASO 2: Crear tabla sales_orders (alineada a arquitectura v1)
-- ============================================================================

CREATE TABLE sales_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    
    -- Numeración (futuro - por ahora NULL permitido)
    order_number INT,
    
    -- Estado comercial
    status VARCHAR(30) NOT NULL CHECK (status IN ('CREATED', 'CONFIRMED', 'CANCELED')),
    
    -- Fiscal (futuro)
    fiscal_status VARCHAR(30) DEFAULT 'PENDING',
    invoice_id UUID,
    
    -- Montos
    total_amount DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    
    -- Auditoría
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1
);

-- Índices de performance
CREATE INDEX idx_sales_orders_tenant ON sales_orders(tenant_id);
CREATE INDEX idx_sales_orders_customer ON sales_orders(customer_id);
CREATE INDEX idx_sales_orders_status ON sales_orders(tenant_id, status);
CREATE INDEX idx_sales_orders_created ON sales_orders(tenant_id, created_at DESC);

-- Comentarios
COMMENT ON TABLE sales_orders IS 'Órdenes de venta - fuente de verdad comercial (HITO v0.3)';
COMMENT ON COLUMN sales_orders.order_number IS 'Número secuencial (futuro HITO v0.4)';
COMMENT ON COLUMN sales_orders.fiscal_status IS 'Estado fiscal (PENDING, PROCESSING, APPROVED)';
COMMENT ON COLUMN sales_orders.version IS 'Versión para optimistic locking (futuro)';

DO $$ BEGIN RAISE NOTICE 'Tabla sales_orders creada'; END $$;

-- ============================================================================
-- PASO 3: Crear tabla sales_order_items
-- ============================================================================

CREATE TABLE sales_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sales_order_id UUID NOT NULL,
    
    -- Producto
    sku VARCHAR(255) NOT NULL,
    
    -- Cantidades y precios
    quantity DECIMAL(15,2) NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    subtotal DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    
    -- Snapshots (inmutables)
    product_snapshot JSONB,
    variant_snapshot JSONB,
    
    -- Auditoría
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Índice por orden (DDD: cargar aggregate)
CREATE INDEX idx_sales_order_items_order ON sales_order_items(sales_order_id);

COMMENT ON TABLE sales_order_items IS 'Items de órdenes de venta (DDD: entities dentro del aggregate)';
COMMENT ON COLUMN sales_order_items.product_snapshot IS 'Snapshot inmutable del producto al momento de crear la orden';
COMMENT ON COLUMN sales_order_items.variant_snapshot IS 'Snapshot inmutable de la variante al momento de crear la orden';

DO $$ BEGIN RAISE NOTICE 'Tabla sales_order_items creada'; END $$;

-- ============================================================================
-- PASO 4: Actualizar pos_sales (extender sin romper)
-- ============================================================================

ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS pos_number INT;
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS point_of_sale_id UUID;
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS fiscal_status VARCHAR(30) DEFAULT 'PENDING';
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS invoice_id UUID;
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();
ALTER TABLE pos_sales ADD COLUMN IF NOT EXISTS version INT DEFAULT 1;

-- Índices adicionales para pos_sales
CREATE INDEX IF NOT EXISTS idx_pos_sales_fiscal ON pos_sales(fiscal_status);
CREATE INDEX IF NOT EXISTS idx_pos_sales_created ON pos_sales(tenant_id, created_at DESC);

COMMENT ON COLUMN pos_sales.pos_number IS 'Número secuencial POS (futuro HITO v0.4)';
COMMENT ON COLUMN pos_sales.fiscal_status IS 'Estado fiscal de la venta';

DO $$ BEGIN RAISE NOTICE 'Tabla pos_sales extendida con campos adicionales'; END $$;

-- ============================================================================
-- FIN
-- ============================================================================

COMMIT;

DO $$ 
BEGIN 
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Migración 010 completada exitosamente';
    RAISE NOTICE 'Tablas creadas:';
    RAISE NOTICE '  - sales_orders (nueva)';
    RAISE NOTICE '  - sales_order_items (nueva)';
    RAISE NOTICE 'Tablas extendidas:';
    RAISE NOTICE '  - pos_sales (campos adicionales)';
    RAISE NOTICE '========================================';
END $$;
