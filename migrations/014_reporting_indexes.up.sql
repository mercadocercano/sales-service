-- Migration: Índices para reportes financieros (open-orders, aging)
-- Evita full scans en reporting; solo sales_orders + sales_payments

CREATE INDEX IF NOT EXISTS idx_sales_orders_tenant_status
ON sales_orders(tenant_id, status);

CREATE INDEX IF NOT EXISTS idx_sales_payments_order
ON sales_payments(sales_order_id);

COMMENT ON INDEX idx_sales_orders_tenant_status IS 'Reporting: filtro por tenant y status (CREATED/CONFIRMED)';
COMMENT ON INDEX idx_sales_payments_order IS 'Reporting: JOIN agregado de pagos por orden';
