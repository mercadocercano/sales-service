-- ============================================================================
-- Migración 011: Document Sequences (Optimistic Locking)
-- Fecha: 2026-02-20
-- Hito: v0.4 - Numeración Secuencial
-- Estrategia: Control de concurrencia desde código (no DB sequences)
-- ============================================================================

BEGIN;

-- ============================================================================
-- Crear tabla document_sequences
-- ============================================================================

CREATE TABLE document_sequences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    document_type VARCHAR(50) NOT NULL,
    current_number INT NOT NULL DEFAULT 0,
    version INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Índice compuesto para queries frecuentes
CREATE INDEX idx_document_sequences_tenant_type ON document_sequences(tenant_id, document_type);

-- Comentarios
COMMENT ON TABLE document_sequences IS 'Secuencias de numeración por tenant y tipo de documento (HITO v0.4)';
COMMENT ON COLUMN document_sequences.document_type IS 'Tipo: SALES_ORDER, POS_SALE, INVOICE, CREDIT_NOTE';
COMMENT ON COLUMN document_sequences.current_number IS 'Último número asignado';
COMMENT ON COLUMN document_sequences.version IS 'Versión para optimistic locking';

DO $$ 
BEGIN 
    RAISE NOTICE 'Tabla document_sequences creada';
END $$;

-- ============================================================================
-- Inicializar secuencias para tenant de testing
-- ============================================================================

INSERT INTO document_sequences (id, tenant_id, document_type, current_number, version, updated_at)
VALUES 
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'SALES_ORDER', 0, 1, NOW()),
    (gen_random_uuid(), '00000000-0000-0000-0000-000000000001', 'POS_SALE', 0, 1, NOW());

DO $$ 
BEGIN 
    RAISE NOTICE 'Secuencias inicializadas para tenant 00000000-0000-0000-0000-000000000001';
END $$;

COMMIT;

DO $$ 
BEGIN 
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Migración 011 completada exitosamente';
    RAISE NOTICE 'Tabla: document_sequences';
    RAISE NOTICE 'Secuencias inicializadas: SALES_ORDER, POS_SALE';
    RAISE NOTICE '========================================';
END $$;
