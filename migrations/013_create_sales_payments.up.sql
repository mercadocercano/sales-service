-- Migration: Create sales_payments table
-- HITO: Cobranza mínima integrada E2E

CREATE TABLE IF NOT EXISTS sales_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    sales_order_id UUID NOT NULL,
    amount DECIMAL(18,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'ARS',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_sales_order 
        FOREIGN KEY (sales_order_id) 
        REFERENCES sales_orders(id) 
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_payments_order ON sales_payments(sales_order_id);
CREATE INDEX IF NOT EXISTS idx_payments_tenant ON sales_payments(tenant_id, created_at DESC);

COMMENT ON TABLE sales_payments IS 'Pagos registrados contra órdenes confirmadas - HITO Cobranza';
COMMENT ON COLUMN sales_payments.sales_order_id IS 'FK a sales_orders - solo órdenes confirmadas';
COMMENT ON COLUMN sales_payments.amount IS 'Monto del pago (> 0, suma <= total orden)';
COMMENT ON COLUMN sales_payments.currency IS 'Moneda del pago (ARS hardcoded para HITO)';
