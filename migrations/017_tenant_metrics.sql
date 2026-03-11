-- ============================================================================
-- Migration 017: tenant_metrics (H8 TTFS edge case fix)
-- Registra primera venta real por tenant. Evita TTFS duplicado cuando se
-- eliminan ventas y el comercio vuelve a vender.
-- ============================================================================

CREATE TABLE IF NOT EXISTS tenant_metrics (
    tenant_id UUID PRIMARY KEY,
    first_sale_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_metrics_first_sale_at
ON tenant_metrics (first_sale_at);

COMMENT ON TABLE tenant_metrics IS 'H8: Primera venta por tenant. INSERT ON CONFLICT DO NOTHING para TTFS consistente.';
