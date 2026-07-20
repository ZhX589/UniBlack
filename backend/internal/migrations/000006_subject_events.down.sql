-- Reverse Phase 13A additions. public_id values are not restored to "absence of column"
-- without data loss; dropping the column is intentional rollback of structure only.

DROP INDEX IF EXISTS idx_evidence_event;
ALTER TABLE evidence DROP COLUMN IF EXISTS original_filename;
ALTER TABLE evidence DROP COLUMN IF EXISTS storage_key;
ALTER TABLE evidence DROP COLUMN IF EXISTS event_id;

DROP INDEX IF EXISTS idx_events_status;
DROP INDEX IF EXISTS idx_events_subject;
DROP TABLE IF EXISTS events;

DROP INDEX IF EXISTS idx_accounts_subject;
DROP INDEX IF EXISTS idx_accounts_platform_username;
DROP INDEX IF EXISTS idx_accounts_platform_id;
DROP TABLE IF EXISTS accounts;

DROP INDEX IF EXISTS idx_subjects_public_id;
ALTER TABLE subjects DROP COLUMN IF EXISTS public_id;
