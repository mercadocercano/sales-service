-- Migration: 018_add_sale_number_to_pos_sales
-- Description: Agrega numeración de comprobante INTERNO (no fiscal/AFIP) a ventas POS.
-- Author: technical-leader (E18 Tramo B)
-- Date: 2026-06-17
-- Hito: E18 Tramo B - Detalle de comprobante + numeración correlativa por tenant
--
-- CONTEXTO:
--   El número se asigna dentro de la MISMA transacción que inserta pos_sales,
--   tomando el siguiente correlativo desde document_sequences (document_type = 'POS_SALE')
--   con SELECT ... FOR UPDATE. Esta columna almacena el número ya asignado para
--   poder mostrarlo en el detalle del comprobante sin recalcular.
--
-- IMPORTANTE: sale_number es un número de comprobante INTERNO, NO es un número
--   fiscal AFIP ni una factura electrónica. La facturación electrónica queda
--   fuera de scope (ver E18, sección Riesgos).

BEGIN;

-- Aditivo: NULL permitido para no romper filas preexistentes (ventas anteriores
-- a esta migración no tienen número asignado).
ALTER TABLE pos_sales
    ADD COLUMN IF NOT EXISTS sale_number INT NULL;

COMMENT ON COLUMN pos_sales.sale_number IS
    'Número de comprobante INTERNO correlativo por tenant (no fiscal/AFIP). Asignado atómicamente desde document_sequences (POS_SALE).';

-- Unicidad por tenant: dos ventas del mismo tenant no pueden compartir número.
-- Índice parcial: ignora filas con sale_number NULL (ventas previas a la migración).
CREATE UNIQUE INDEX IF NOT EXISTS uq_pos_sales_tenant_sale_number
    ON pos_sales (tenant_id, sale_number)
    WHERE sale_number IS NOT NULL;

COMMIT;

-- ROLLBACK (ejecutar manualmente si se necesita revertir):
-- BEGIN;
-- DROP INDEX IF EXISTS uq_pos_sales_tenant_sale_number;
-- ALTER TABLE pos_sales DROP COLUMN IF EXISTS sale_number;
-- COMMIT;
