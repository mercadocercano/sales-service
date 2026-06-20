-- Down: irreversible (ALTER/indice/datos). ADR-001 Param 1.
DO $$ BEGIN RAISE EXCEPTION 'Migration 003 is irreversible. Restore from backup if a rollback is required.'; END $$;
