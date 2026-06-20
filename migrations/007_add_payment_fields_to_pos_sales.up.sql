-- Migration: 007_add_payment_fields_to_pos_sales
-- Description: Agrega amount_paid y change para POS ticket printing
-- Author: System
-- Date: 2025-02-17
-- HITO: POST /pos/sale devuelve DTO listo para imprimir

-- ========================================
-- AGREGAR CAMPOS TRANSACCIONALES
-- ========================================

-- amount_paid: Monto pagado por el cliente
ALTER TABLE pos_sales 
ADD COLUMN IF NOT EXISTS amount_paid DECIMAL(15,2) NOT NULL DEFAULT 0;

-- change: Vuelto calculado (amount_paid - final_amount)
ALTER TABLE pos_sales 
ADD COLUMN IF NOT EXISTS change DECIMAL(15,2) NOT NULL DEFAULT 0;

-- ========================================
-- COMENTARIOS
-- ========================================

COMMENT ON COLUMN pos_sales.amount_paid IS 'Monto pagado por el cliente (debe ser >= final_amount)';
COMMENT ON COLUMN pos_sales.change IS 'Vuelto calculado (amount_paid - final_amount)';

-- ========================================
-- CONSTRAINT DE VALIDACIÓN
-- ========================================

-- Validar que amount_paid >= final_amount (no puede ser negativo)
ALTER TABLE pos_sales 
ADD CONSTRAINT chk_amount_paid_sufficient 
CHECK (amount_paid >= final_amount);

-- Validar que change sea consistente con la operación
ALTER TABLE pos_sales 
ADD CONSTRAINT chk_change_non_negative 
CHECK (change >= 0);

-- ========================================
-- LOGGING
-- ========================================

DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Migration 007 completed successfully';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Added columns:';
    RAISE NOTICE '  - amount_paid (DECIMAL 15,2, NOT NULL, DEFAULT 0)';
    RAISE NOTICE '  - change (DECIMAL 15,2, NOT NULL, DEFAULT 0)';
    RAISE NOTICE '';
    RAISE NOTICE 'Added constraints:';
    RAISE NOTICE '  - chk_amount_paid_sufficient (amount_paid >= final_amount)';
    RAISE NOTICE '  - chk_change_non_negative (change >= 0)';
    RAISE NOTICE '';
    RAISE NOTICE 'HITO: POST /pos/sale now returns complete DTO for printing';
    RAISE NOTICE '========================================';
END $$;
