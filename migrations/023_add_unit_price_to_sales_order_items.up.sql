-- ============================================================================
-- Migration: Add unit_price to sales_order_items
-- Date: 2025-02-21
-- Purpose: Store unit_price as primary source of truth for item pricing
--         Snapshots become optional enrichment for traceability
-- ============================================================================

BEGIN;

-- Add unit_price column (idempotent — 010 ya la crea en instalaciones nuevas)
ALTER TABLE sales_order_items
ADD COLUMN IF NOT EXISTS unit_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00;

-- Add check constraint (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_unit_price_positive'
    ) THEN
        ALTER TABLE sales_order_items
        ADD CONSTRAINT chk_unit_price_positive CHECK (unit_price > 0);
    END IF;
END;
$$;

-- Drop default after backfilling (if needed)
ALTER TABLE sales_order_items
ALTER COLUMN unit_price DROP DEFAULT;

COMMIT;

-- ============================================================================
-- Notes:
-- - unit_price is now the primary source for total_amount calculation
-- - Snapshots (product_snapshot, variant_snapshot) remain for audit/traceability
-- - If PIM is unavailable, Sales can still create orders with request unit_price
-- ============================================================================
