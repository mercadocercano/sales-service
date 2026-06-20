-- Migration: 005_pos_sales_multi_item
-- Description: Refactor pos_sales para soportar multi-item + descuentos
-- Author: System
-- Date: 2026-02-12
-- Hito: HITO B - POS Multi-Item + Descuento Global
--
-- ESTRATEGIA: DESTRUCTIVA (datos de testing, seguro eliminar)
--
-- CAMBIOS:
-- 1. TRUNCATE pos_sales (eliminar 8 registros de testing)
-- 2. Crear tabla pos_sale_items (patrón DDD: Aggregate + Entities)
-- 3. Modificar pos_sales (eliminar stock_entry_id, agregar discount/final_amount)

-- =====================================================
-- PASO 1: Eliminar datos de testing
-- =====================================================
TRUNCATE TABLE pos_sales CASCADE;

-- =====================================================
-- PASO 2: Crear tabla pos_sale_items
-- =====================================================
CREATE TABLE IF NOT EXISTS pos_sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pos_sale_id UUID NOT NULL REFERENCES pos_sales(id) ON DELETE CASCADE,
    
    -- Identificación del producto
    sku VARCHAR(255) NOT NULL,
    product_name VARCHAR(500) NOT NULL,
    
    -- Cantidades y precios
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(15, 2) NOT NULL CHECK (unit_price >= 0),
    subtotal NUMERIC(15, 2) NOT NULL CHECK (subtotal >= 0),
    
    -- Referencia al movimiento de stock
    stock_entry_id UUID NOT NULL,
    
    -- Auditoría
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Índices para consultas eficientes
CREATE INDEX idx_pos_sale_items_pos_sale_id ON pos_sale_items(pos_sale_id);
CREATE INDEX idx_pos_sale_items_sku ON pos_sale_items(sku);
CREATE INDEX idx_pos_sale_items_stock_entry_id ON pos_sale_items(stock_entry_id);

-- =====================================================
-- PASO 3: Modificar pos_sales (eliminar stock_entry_id)
-- =====================================================
ALTER TABLE pos_sales DROP COLUMN IF EXISTS stock_entry_id;

-- =====================================================
-- PASO 4: Agregar campos de descuento a pos_sales
-- =====================================================
ALTER TABLE pos_sales ADD COLUMN discount_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0);
ALTER TABLE pos_sales ADD COLUMN final_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (final_amount >= 0);

-- =====================================================
-- PASO 5: Renombrar total_amount para claridad
-- =====================================================
-- total_amount = suma de subtotales (antes de descuento)
-- final_amount = total_amount - discount_amount (lo que se cobra)
COMMENT ON COLUMN pos_sales.total_amount IS 'Suma de subtotales de items (antes de descuento)';
COMMENT ON COLUMN pos_sales.discount_amount IS 'Descuento aplicado (monto fijo, no porcentual)';
COMMENT ON COLUMN pos_sales.final_amount IS 'Monto final cobrado = total_amount - discount_amount';

-- =====================================================
-- PASO 6: Comentarios de documentación
-- =====================================================
COMMENT ON TABLE pos_sale_items IS 'Items de venta POS - Patrón DDD Aggregate (cada venta tiene N items)';
COMMENT ON COLUMN pos_sale_items.id IS 'ID único del item';
COMMENT ON COLUMN pos_sale_items.pos_sale_id IS 'ID de la venta POS padre (Aggregate Root)';
COMMENT ON COLUMN pos_sale_items.sku IS 'SKU de la variante vendida';
COMMENT ON COLUMN pos_sale_items.product_name IS 'Snapshot del nombre del producto (inmutable)';
COMMENT ON COLUMN pos_sale_items.quantity IS 'Cantidad vendida';
COMMENT ON COLUMN pos_sale_items.unit_price IS 'Precio unitario snapshot (inmutable)';
COMMENT ON COLUMN pos_sale_items.subtotal IS 'Subtotal del item = unit_price * quantity';
COMMENT ON COLUMN pos_sale_items.stock_entry_id IS 'Referencia al movimiento de stock asociado a este item';

-- =====================================================
-- Logging de migración
-- =====================================================
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Migration 005: POS Multi-Item';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Tabla creada: pos_sale_items';
    RAISE NOTICE 'Tabla modificada: pos_sales';
    RAISE NOTICE '  - Eliminado: stock_entry_id';
    RAISE NOTICE '  - Agregado: discount_amount';
    RAISE NOTICE '  - Agregado: final_amount';
    RAISE NOTICE '';
    RAISE NOTICE 'Datos eliminados: 8 registros de testing';
    RAISE NOTICE 'Patrón: DDD Aggregate + Entities';
    RAISE NOTICE 'Hito: HITO B - POS Multi-Item';
END $$;
