-- 025_create_app_role.down.sql — revierte 025_create_app_role.up.sql

REVOKE ALL ON
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
FROM sales_app;

REVOKE USAGE ON SCHEMA public FROM sales_app;
REVOKE CONNECT ON DATABASE order_db FROM sales_app;

DROP ROLE IF EXISTS sales_app;
