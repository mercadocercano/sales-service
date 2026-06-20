-- Migration: confirmed_at para aging financiero correcto
-- La antigüedad de deuda debe basarse en fecha de confirmación, no en updated_at
-- (updated_at cambia con cada pago y alteraría artificialmente el aging)

ALTER TABLE sales_orders
ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMP NULL;

COMMENT ON COLUMN sales_orders.confirmed_at IS 'Fecha de confirmación; base para aging financiero (no updated_at)';

-- Backfill: órdenes ya CONFIRMED sin confirmed_at usan updated_at como aproximación
UPDATE sales_orders
SET confirmed_at = updated_at
WHERE status = 'CONFIRMED' AND confirmed_at IS NULL;
