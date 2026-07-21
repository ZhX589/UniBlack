DROP INDEX IF EXISTS idx_sanctions_user_active;
DROP TABLE IF EXISTS sanctions;
DROP INDEX IF EXISTS idx_appeals_event;
ALTER TABLE appeals DROP COLUMN IF EXISTS resolution_reason;
ALTER TABLE appeals DROP COLUMN IF EXISTS outcome;
ALTER TABLE appeals DROP COLUMN IF EXISTS event_id;
