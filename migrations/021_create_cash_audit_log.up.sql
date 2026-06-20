-- Migration: cash_audit_log (E18 Tramo A — ADR-003 gate de seguridad #5)
-- Bitácora INMUTABLE de toda operación de efectivo: quién (user_id del JWT), cuándo
-- (timestamp de servidor), qué acción y los montos relevantes. Append-only.

CREATE TABLE IF NOT EXISTS cash_audit_log (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID NOT NULL,
    cash_register_session_id  UUID NOT NULL,
    action                    VARCHAR(40) NOT NULL,   -- opened | movement_registered | closed | review_approved
    user_id                   UUID NOT NULL,          -- actor (del JWT), nunca del body
    point_of_sale_id          VARCHAR(120),
    -- Montos en el momento de la acción (NULL según corresponda)
    opening_amount            NUMERIC(15,2),
    expected_amount           NUMERIC(15,2),
    counted_amount            NUMERIC(15,2),
    difference                NUMERIC(15,2),
    movement_type             VARCHAR(20),
    movement_amount           NUMERIC(15,2),
    detail                    TEXT,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cash_audit_session ON cash_audit_log (cash_register_session_id);
CREATE INDEX IF NOT EXISTS idx_cash_audit_tenant ON cash_audit_log (tenant_id, created_at);
