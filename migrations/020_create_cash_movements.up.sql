-- Migration: cash_movements (E18 Tramo A — ADR-003 §2, §gate #4)
-- Movimientos MANUALES de efectivo (ingresos/egresos/retiros/correcciones).
-- Las VENTAS NO se duplican acá: se derivan de pos_sales (single source of truth).
-- APPEND-ONLY: sin UPDATE/DELETE; las correcciones son contra-asientos (otro INSERT).

CREATE TABLE IF NOT EXISTS cash_movements (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID NOT NULL,
    cash_register_session_id  UUID NOT NULL REFERENCES cash_register_sessions(id),
    type                      VARCHAR(20) NOT NULL,   -- income | expense | withdrawal | correction
    amount                    NUMERIC(15,2) NOT NULL, -- siempre > 0; el signo lo da el type
    reason                    TEXT NOT NULL,
    created_by                UUID NOT NULL,          -- user_id del JWT
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT cash_movements_type_check
        CHECK (type IN ('income', 'expense', 'withdrawal', 'correction')),
    CONSTRAINT cash_movements_amount_positive CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_cash_movements_session ON cash_movements (cash_register_session_id);
CREATE INDEX IF NOT EXISTS idx_cash_movements_tenant ON cash_movements (tenant_id);
