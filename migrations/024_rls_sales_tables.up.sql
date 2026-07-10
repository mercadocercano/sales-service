-- 024_rls_sales_tables.up.sql — Aislamiento fail-closed (Devy RULE-09/RULE-10, roadmap PLAT-E25)
--
-- El aislamiento NO depende del WHERE de cada *PostgresRepository: si una query olvidara
-- filtrar por tenant_id, la base la filtra igual. La app hace SET LOCAL app.tenant_id por
-- transacción (go-shared postgres.WithRLSInTransaction), leído siempre del claim JWT ya
-- validado por tenantmw.TenantValidation — nunca de un header crudo (ver T12).
--
-- Dos patrones:
--   (1) 10 tablas con tenant_id propio (incluye tenant_metrics, donde tenant_id es la PK):
--       policy directa por igualdad.
--   (2) 2 tablas hijas sin tenant_id propio (sales_order_items, pos_sale_items): policy por
--       subquery contra el padre — primer caso de este tipo en el parque MC (sin precedente
--       en E24/E26). Índices que soportan la subquery ya existen:
--       idx_sales_order_items_order (sales_order_id) e idx_pos_sale_items_pos_sale_id
--       (pos_sale_id) — confirmados en 010_clean_migration_to_sales_schema.up.sql y
--       005_pos_sales_multi_item.up.sql, no se crean de nuevo acá.

-- ── Patrón 1: tablas con tenant_id propio ──────────────────────────────────────────────

ALTER TABLE sales_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_orders FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON sales_orders;
CREATE POLICY tenant_isolation ON sales_orders
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE pos_sales ENABLE ROW LEVEL SECURITY;
ALTER TABLE pos_sales FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON pos_sales;
CREATE POLICY tenant_isolation ON pos_sales
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE sales_payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_payments FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON sales_payments;
CREATE POLICY tenant_isolation ON sales_payments
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE document_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_sequences FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON document_sequences;
CREATE POLICY tenant_isolation ON document_sequences
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE customer_credits ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_credits FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON customer_credits;
CREATE POLICY tenant_isolation ON customer_credits
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE customer_credit_applications ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_credit_applications FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON customer_credit_applications;
CREATE POLICY tenant_isolation ON customer_credit_applications
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- tenant_id es la PRIMARY KEY (una fila por tenant) — la policy sigue siendo una igualdad simple.
ALTER TABLE tenant_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_metrics FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON tenant_metrics;
CREATE POLICY tenant_isolation ON tenant_metrics
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE cash_register_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_register_sessions FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON cash_register_sessions;
CREATE POLICY tenant_isolation ON cash_register_sessions
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE cash_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_movements FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON cash_movements;
CREATE POLICY tenant_isolation ON cash_movements
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

ALTER TABLE cash_audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE cash_audit_log FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON cash_audit_log;
CREATE POLICY tenant_isolation ON cash_audit_log
    USING      (tenant_id = current_setting('app.tenant_id')::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid);

-- ── Patrón 2: tablas hijas sin tenant_id propio — policy por subquery contra el padre ──────

ALTER TABLE sales_order_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE sales_order_items FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON sales_order_items;
CREATE POLICY tenant_isolation ON sales_order_items
    USING (sales_order_id IN (
        SELECT id FROM sales_orders WHERE tenant_id = current_setting('app.tenant_id')::uuid
    ))
    WITH CHECK (sales_order_id IN (
        SELECT id FROM sales_orders WHERE tenant_id = current_setting('app.tenant_id')::uuid
    ));

ALTER TABLE pos_sale_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE pos_sale_items FORCE  ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON pos_sale_items;
CREATE POLICY tenant_isolation ON pos_sale_items
    USING (pos_sale_id IN (
        SELECT id FROM pos_sales WHERE tenant_id = current_setting('app.tenant_id')::uuid
    ))
    WITH CHECK (pos_sale_id IN (
        SELECT id FROM pos_sales WHERE tenant_id = current_setting('app.tenant_id')::uuid
    ));
