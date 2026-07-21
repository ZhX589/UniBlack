-- Dual-target rows were allowed before this migration. Prefer the Event target
-- for Event-first governance and clear the legacy Case foreign key.
UPDATE appeals SET case_id = NULL WHERE case_id IS NOT NULL AND event_id IS NOT NULL;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM appeals WHERE case_id IS NULL AND event_id IS NULL) THEN
        RAISE EXCEPTION 'cannot upgrade event governance while appeals lack both case_id and event_id';
    END IF;
END $$;

ALTER TABLE appeals ALTER COLUMN case_id DROP NOT NULL;
ALTER TABLE appeals DROP CONSTRAINT IF EXISTS appeals_target_check;
ALTER TABLE appeals ADD CONSTRAINT appeals_target_check CHECK (
    (CASE WHEN case_id IS NULL THEN 0 ELSE 1 END) + (CASE WHEN event_id IS NULL THEN 0 ELSE 1 END) = 1
);

ALTER TABLE submissions ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE appeals ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_submissions_active ON submissions(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_appeals_active ON appeals(status) WHERE deleted_at IS NULL;
