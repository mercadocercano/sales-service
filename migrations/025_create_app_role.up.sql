-- 025_create_app_role.up.sql — PLAT-E25 T3 (RULE-09: rol de aplicación sin DDL)
--
-- El runtime de sales-service conectaba como `postgres` (superuser, rolbypassrls=t) —
-- el mismo default inseguro que ledger-service tenía antes de PLAT-E24 T3. Un rol
-- BYPASSRLS anula por completo las policies de 024_rls_sales_tables: sin este rol, T2
-- no protege nada en runtime.
--
-- Idempotente: seguro de re-aplicar (CREATE ROLE guardado en IF NOT EXISTS vía DO block;
-- GRANT es siempre idempotente en Postgres).
--
-- Nota (@dev-architect, revisión 2026-07-08): SELECT explícito sobre sales_orders y
-- pos_sales es requerido por las policies-subquery de sales_order_items/pos_sale_items
-- (024), aunque el rol ya las toque por otras rutas de GRANT de abajo.
--
-- Alcance: solo order_db (las 12 tablas de esta épica). payment_method_db es scope de
-- PLAT-E26; eventbus queda fuera de esta épica (ver Riesgos de la épica).
--
-- Seguridad (hallazgo de review automático, 2026-07-09): esta migración NO fija password —
-- un literal acá quedaría en git history para siempre en cuanto se commitee. El rol se crea
-- sin password (login deshabilitado hasta setearla) y la password real se fija out-of-band
-- vía `ALTER ROLE sales_app PASSWORD '...'` corrido a mano contra lab-postgres, nunca
-- versionado. `.env.example` documenta el valor dev/lab vigente (no de producción).

DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'sales_app') THEN
    CREATE ROLE sales_app LOGIN
      NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
END
$$;

GRANT CONNECT ON DATABASE order_db TO sales_app;
GRANT USAGE ON SCHEMA public TO sales_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON
  sales_orders,
  sales_order_items,
  pos_sales,
  pos_sale_items,
  sales_payments,
  document_sequences,
  customer_credits,
  customer_credit_applications,
  tenant_metrics,
  cash_register_sessions,
  cash_movements,
  cash_audit_log
TO sales_app;

-- Requerido por las policies-subquery de las tablas hijas (WITH CHECK evalúa
-- sales_order_id IN (SELECT id FROM sales_orders WHERE ...), idem pos_sale_items).
GRANT SELECT ON sales_orders, pos_sales TO sales_app;
