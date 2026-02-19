-- Migration 003: Agregar snapshots inmutables a order_items
-- Objetivo: Guardar snapshot de producto y variante al momento de crear la orden
-- Esto previene que cambios futuros en PIM afecten órdenes históricas

-- 1. Agregar columnas JSONB para snapshots
ALTER TABLE order_items 
ADD COLUMN product_snapshot JSONB,
ADD COLUMN variant_snapshot JSONB;

-- 2. Comentarios de documentación
COMMENT ON COLUMN order_items.product_snapshot IS 'Snapshot inmutable del producto al momento de crear la orden (previene cambios en PIM)';
COMMENT ON COLUMN order_items.variant_snapshot IS 'Snapshot inmutable de la variante al momento de crear la orden (previene cambios en PIM)';

-- 3. Índices GIN para queries eficientes sobre JSONB (opcional, útil para analytics)
CREATE INDEX IF NOT EXISTS idx_order_items_product_snapshot ON order_items USING GIN (product_snapshot);
CREATE INDEX IF NOT EXISTS idx_order_items_variant_snapshot ON order_items USING GIN (variant_snapshot);
