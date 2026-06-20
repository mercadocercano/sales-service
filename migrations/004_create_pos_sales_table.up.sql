-- Migration: 004_create_pos_sales_table
-- Description: Crear tabla de ventas POS (fuente de verdad comercial)
-- Author: System
-- Date: 2025-02-09
-- Hito: POS-SALE-02.BE
--
-- REGLAS:
-- - Sin FK (por ahora)
-- - Sin índices extra
-- - Sin estados
-- - Tabla mínima viable

CREATE TABLE IF NOT EXISTS pos_sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    customer_id UUID NULL,
    payment_method_id UUID NOT NULL,
    total_amount NUMERIC NOT NULL,
    currency TEXT NOT NULL DEFAULT 'ARS',
    stock_entry_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Comentarios de documentación
COMMENT ON TABLE pos_sales IS 'Ventas POS - Fuente de verdad comercial del punto de venta';
COMMENT ON COLUMN pos_sales.id IS 'Identificador único de la venta POS';
COMMENT ON COLUMN pos_sales.tenant_id IS 'ID del tenant (multi-tenant)';
COMMENT ON COLUMN pos_sales.customer_id IS 'ID del cliente (opcional, NULL = consumidor final)';
COMMENT ON COLUMN pos_sales.payment_method_id IS 'ID del método de pago (obligatorio)';
COMMENT ON COLUMN pos_sales.total_amount IS 'Monto total de la venta';
COMMENT ON COLUMN pos_sales.currency IS 'Moneda de la venta (default: ARS)';
COMMENT ON COLUMN pos_sales.stock_entry_id IS 'ID del movimiento de stock asociado';
COMMENT ON COLUMN pos_sales.created_at IS 'Fecha de creación de la venta';

-- Logging
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Migration 004: pos_sales table created';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Tabla: pos_sales';
    RAISE NOTICE 'Propósito: Fuente de verdad comercial POS';
    RAISE NOTICE 'Hito: POS-SALE-02.BE';
    RAISE NOTICE '';
    RAISE NOTICE 'Reglas aplicadas:';
    RAISE NOTICE '  - Sin FK (por diseño)';
    RAISE NOTICE '  - Sin índices adicionales';
    RAISE NOTICE '  - Sin estados';
    RAISE NOTICE '  - customer_id NULL permitido (consumidor final)';
    RAISE NOTICE '  - payment_method_id obligatorio';
END $$;
