-- Seed: Secuencias para tenant Demo (9a4c3eb9-2471-4688-bfc8-973e5b3e4ce8)
-- Necesario para E2E con customers de seed_e2e_customers.sql

INSERT INTO document_sequences (id, tenant_id, document_type, current_number, version, updated_at)
SELECT gen_random_uuid(), '9a4c3eb9-2471-4688-bfc8-973e5b3e4ce8'::uuid, 'SALES_ORDER', 0, 1, NOW()
WHERE NOT EXISTS (SELECT 1 FROM document_sequences WHERE tenant_id = '9a4c3eb9-2471-4688-bfc8-973e5b3e4ce8'::uuid AND document_type = 'SALES_ORDER');

INSERT INTO document_sequences (id, tenant_id, document_type, current_number, version, updated_at)
SELECT gen_random_uuid(), '9a4c3eb9-2471-4688-bfc8-973e5b3e4ce8'::uuid, 'POS_SALE', 0, 1, NOW()
WHERE NOT EXISTS (SELECT 1 FROM document_sequences WHERE tenant_id = '9a4c3eb9-2471-4688-bfc8-973e5b3e4ce8'::uuid AND document_type = 'POS_SALE');
