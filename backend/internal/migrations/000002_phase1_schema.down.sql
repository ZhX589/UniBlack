-- Phase 1: Rollback complete database schema

-- Drop indexes
DROP INDEX IF EXISTS idx_auditlog_created;
DROP INDEX IF EXISTS idx_auditlog_resource;
DROP INDEX IF EXISTS idx_auditlog_user;
DROP INDEX IF EXISTS idx_appeal_status;
DROP INDEX IF EXISTS idx_appeal_case;
DROP INDEX IF EXISTS idx_submission_submitted_by;
DROP INDEX IF EXISTS idx_submission_status;
DROP INDEX IF EXISTS idx_evidence_case;
DROP INDEX IF EXISTS idx_case_submitted_by;
DROP INDEX IF EXISTS idx_case_status;
DROP INDEX IF EXISTS idx_case_subject;
DROP INDEX IF EXISTS idx_identifier_type_value;
DROP INDEX IF EXISTS idx_identifier_subject;

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS appeals;
DROP TABLE IF EXISTS submissions;
DROP TABLE IF EXISTS evidence;
DROP TABLE IF EXISTS cases;
DROP TABLE IF EXISTS identifiers;
DROP TABLE IF EXISTS subjects;

-- Remove org_id from users
ALTER TABLE users DROP COLUMN IF EXISTS org_id;

DROP TABLE IF EXISTS organizations;

-- Remove added columns from permissions
ALTER TABLE permissions DROP COLUMN IF EXISTS action;
ALTER TABLE permissions DROP COLUMN IF EXISTS resource;

-- Remove added columns from roles
ALTER TABLE roles DROP COLUMN IF EXISTS is_system;

-- Remove added columns from users
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;
ALTER TABLE users DROP COLUMN IF EXISTS is_active;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users DROP COLUMN IF EXISTS display_name;
