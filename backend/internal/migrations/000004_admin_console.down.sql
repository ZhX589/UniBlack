-- Rollback Phase 11

DROP INDEX IF EXISTS idx_access_lists_value;
DROP INDEX IF EXISTS idx_access_lists_target;
DROP INDEX IF EXISTS idx_access_lists_type;
DROP TABLE IF EXISTS access_lists;
DROP TABLE IF EXISTS system_settings;

ALTER TABLE users DROP COLUMN IF EXISTS email_verification_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_token;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
