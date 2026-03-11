-- ============================================================================
-- Migration: Add unit_price to sales_order_items
-- Date: 2025-02-21
-- Purpose: Store unit_price as primary source of truth for item pricing
--         Snapshots become optional enrichment for traceability
-- ============================================================================

BEGIN;

-- Add unit_price column
ALTER TABLE sales_order_items 
ADD COLUMN unit_price NUMERIC(12, 2) NOT NULL DEFAULT 0.00;

-- Add check constraint
ALTER TABLE sales_order_items 
ADD CONSTRAINT chk_unit_price_positive CHECK (unit_price > 0);

-- Drop default after backfilling (if needed)
-- For new installations, this is safe
ALTER TABLE sales_order_items 
ALTER COLUMN unit_price DROP DEFAULT;

COMMIT;

-- ============================================================================
-- Notes:
-- - unit_price is now the primary source for total_amount calculation
-- - Snapshots (product_snapshot, variant_snapshot) remain for audit/traceability
-- - If PIM is unavailable, Sales can still create orders with request unit_price
-- ============================================================================
