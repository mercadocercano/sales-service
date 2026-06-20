-- Migration: vínculo venta -> caja (E18 Tramo A — ADR-003 §4, modo DEGRADADO)
-- FK NULLABLE y retrocompatible: las ventas existentes quedan con NULL y el flujo POS
-- NO se bloquea si no hay caja abierta. Si la venta trae una sesión abierta de su
-- terminal, se asocia; si no, procede con NULL.

ALTER TABLE pos_sales
    ADD COLUMN IF NOT EXISTS cash_register_session_id UUID
        REFERENCES cash_register_sessions(id);

CREATE INDEX IF NOT EXISTS idx_pos_sales_cash_session
    ON pos_sales (cash_register_session_id)
    WHERE cash_register_session_id IS NOT NULL;
