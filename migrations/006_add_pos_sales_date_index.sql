-- Migration: 006_add_pos_sales_date_index
-- Description: Crear índice compuesto para queries de reportes por fecha
-- Author: System
-- Date: 2026-02-12
-- Hito: HITO C - Reportes Diarios
--
-- OBJETIVO:
-- Optimizar queries de reportes diarios que filtran por tenant_id + created_at
--
-- IMPACTO:
-- - Mejora performance de queries por rango de fecha
-- - Permite ORDER BY created_at DESC eficiente
-- - Esencial para endpoint GET /api/v1/reports/daily

-- =====================================================
-- PASO 1: Crear índice compuesto en pos_sales
-- =====================================================
CREATE INDEX IF NOT EXISTS idx_pos_sales_tenant_created 
ON pos_sales(tenant_id, created_at DESC);

-- =====================================================
-- PASO 2: Comentario de documentación
-- =====================================================
COMMENT ON INDEX idx_pos_sales_tenant_created IS 
'HITO C - Índice para reportes diarios - Optimiza queries por tenant y fecha';

-- =====================================================
-- Logging de migración
-- =====================================================
DO $$
BEGIN
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Migration 006: Índice Reportes Diarios';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Índice creado: idx_pos_sales_tenant_created';
    RAISE NOTICE 'Columnas: (tenant_id, created_at DESC)';
    RAISE NOTICE 'Tabla: pos_sales';
    RAISE NOTICE 'Hito: HITO C - Reportes Diarios';
    RAISE NOTICE '';
    RAISE NOTICE 'Performance esperada:';
    RAISE NOTICE '  - Queries por fecha: O(log n)';
    RAISE NOTICE '  - Agregaciones diarias: Eficientes';
END $$;
