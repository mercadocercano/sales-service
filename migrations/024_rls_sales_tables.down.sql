-- 024_rls_sales_tables.down.sql — revierte RLS fail-closed en las 12 tablas de sales-service

DROP POLICY IF EXISTS tenant_isolation ON sales_orders;
ALTER TABLE sales_orders NO FORCE ROW LEVEL SECURITY;
ALTER TABLE sales_orders DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON pos_sales;
ALTER TABLE pos_sales NO FORCE ROW LEVEL SECURITY;
ALTER TABLE pos_sales DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON sales_payments;
ALTER TABLE sales_payments NO FORCE ROW LEVEL SECURITY;
ALTER TABLE sales_payments DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON document_sequences;
ALTER TABLE document_sequences NO FORCE ROW LEVEL SECURITY;
ALTER TABLE document_sequences DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON customer_credits;
ALTER TABLE customer_credits NO FORCE ROW LEVEL SECURITY;
ALTER TABLE customer_credits DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON customer_credit_applications;
ALTER TABLE customer_credit_applications NO FORCE ROW LEVEL SECURITY;
ALTER TABLE customer_credit_applications DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON tenant_metrics;
ALTER TABLE tenant_metrics NO FORCE ROW LEVEL SECURITY;
ALTER TABLE tenant_metrics DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON cash_register_sessions;
ALTER TABLE cash_register_sessions NO FORCE ROW LEVEL SECURITY;
ALTER TABLE cash_register_sessions DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON cash_movements;
ALTER TABLE cash_movements NO FORCE ROW LEVEL SECURITY;
ALTER TABLE cash_movements DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON cash_audit_log;
ALTER TABLE cash_audit_log NO FORCE ROW LEVEL SECURITY;
ALTER TABLE cash_audit_log DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON sales_order_items;
ALTER TABLE sales_order_items NO FORCE ROW LEVEL SECURITY;
ALTER TABLE sales_order_items DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON pos_sale_items;
ALTER TABLE pos_sale_items NO FORCE ROW LEVEL SECURITY;
ALTER TABLE pos_sale_items DISABLE ROW LEVEL SECURITY;
