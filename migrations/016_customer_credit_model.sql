-- ============================================================================
-- Migration 016: Customer Credit Model (pagos a cuenta del cliente)
-- Fase 1: Introducción paralela. No modifica sales_payments.
-- Orden de locks en aplicación: 1) sales_orders FOR UPDATE, 2) customer_credits FOR UPDATE
-- ============================================================================

-- 1. Tabla: customer_credits
-- Crédito disponible del cliente (pago "a cuenta"). Se aplica luego a órdenes.
CREATE TABLE IF NOT EXISTS customer_credits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    customer_id UUID NOT NULL,

    total_amount NUMERIC(15,2) NOT NULL CHECK (total_amount > 0),
    applied_amount NUMERIC(15,2) NOT NULL DEFAULT 0 CHECK (applied_amount >= 0),
    remaining_amount NUMERIC(15,2) NOT NULL CHECK (remaining_amount >= 0),

    status VARCHAR(20) NOT NULL CHECK (status IN ('AVAILABLE', 'FULLY_APPLIED')),

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Invariante en app: remaining_amount = total_amount - applied_amount; applied_amount <= total_amount
CREATE INDEX IF NOT EXISTS idx_customer_credits_tenant_customer
ON customer_credits (tenant_id, customer_id);

CREATE INDEX IF NOT EXISTS idx_customer_credits_status
ON customer_credits (tenant_id, status);

COMMENT ON TABLE customer_credits IS 'Créditos a cuenta del cliente (Fase 1 credit-based). No reemplaza sales_payments aún.';
COMMENT ON COLUMN customer_credits.remaining_amount IS 'total_amount - applied_amount; actualizar en cada aplicación';
COMMENT ON COLUMN customer_credits.status IS 'AVAILABLE = remaining > 0; FULLY_APPLIED = remaining = 0';

-- 2. Tabla: customer_credit_applications
-- Aplicación de crédito a una orden (parcial o total). Auditoría y soporte multi-orden.
CREATE TABLE IF NOT EXISTS customer_credit_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,

    customer_credit_id UUID NOT NULL,
    sales_order_id UUID NOT NULL,

    amount NUMERIC(15,2) NOT NULL CHECK (amount > 0),

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_credit
        FOREIGN KEY (customer_credit_id)
        REFERENCES customer_credits(id)
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_credit_applications_credit
ON customer_credit_applications (customer_credit_id);

CREATE INDEX IF NOT EXISTS idx_credit_applications_order
ON customer_credit_applications (sales_order_id);

CREATE INDEX IF NOT EXISTS idx_credit_applications_tenant
ON customer_credit_applications (tenant_id);

COMMENT ON TABLE customer_credit_applications IS 'Aplicación de crédito a orden. Validar en app: credit.customer_id = order.customer_id, credit.tenant_id = order.tenant_id';

-- Nota: FK a sales_orders no añadida para no acoplar a schema de órdenes en esta migración;
-- la aplicación valida order existe y pertenece al mismo tenant/customer.
