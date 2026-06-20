-- Migration: cash_register_sessions (E18 Tramo A — caja POS, ADR-003)
-- Raíz de agregado de la sesión de caja. Maneja efectivo real (L4).
-- Estados: open -> closing -> closed | pending_review (unidireccional, validado en Go).

CREATE TABLE IF NOT EXISTS cash_register_sessions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    point_of_sale_id     VARCHAR(120) NOT NULL,           -- terminal/caja física (opaco, lo provee el cliente)
    status               VARCHAR(20) NOT NULL DEFAULT 'open',

    -- Apertura
    opened_by            UUID NOT NULL,                   -- user_id del JWT (cajero)
    opening_amount       NUMERIC(15,2) NOT NULL,
    opened_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Umbral de descuadre vigente al abrir (snapshot inmutable; tolerancia configurable por tenant)
    review_threshold     NUMERIC(15,2) NOT NULL DEFAULT 100.00,

    -- Cierre (snapshot inmutable, NULL hasta cerrar)
    closed_by            UUID,                            -- user_id que ejecutó el cierre
    counted_amount       NUMERIC(15,2),                   -- efectivo contado por el cajero
    expected_amount      NUMERIC(15,2),                   -- calculado server-side
    difference           NUMERIC(15,2),                   -- counted - expected
    closed_at            TIMESTAMPTZ,

    -- Aprobación de descuadre (PENDING_REVIEW -> closed), separación de funciones
    approved_by          UUID,                            -- supervisor (debe ser != opened_by y != closed_by)
    approved_at          TIMESTAMPTZ,

    version              INTEGER NOT NULL DEFAULT 1,      -- optimistic lock (patrón sales_orders)
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT cash_sessions_status_check
        CHECK (status IN ('open', 'closing', 'closed', 'pending_review')),
    CONSTRAINT cash_sessions_opening_amount_nonneg CHECK (opening_amount >= 0)
);

-- Invariante: UNA caja abierta por terminal dentro del tenant (ADR-003 §3).
CREATE UNIQUE INDEX IF NOT EXISTS idx_cash_sessions_one_open_per_terminal
    ON cash_register_sessions (tenant_id, point_of_sale_id)
    WHERE status = 'open';

CREATE INDEX IF NOT EXISTS idx_cash_sessions_tenant ON cash_register_sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cash_sessions_status ON cash_register_sessions (tenant_id, status);
