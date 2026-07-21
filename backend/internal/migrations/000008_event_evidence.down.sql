-- Refuse before changing schema if Phase 13 Event-only evidence exists.
-- This avoids a partial rollback that would orphan stored evidence.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM evidence WHERE case_id IS NULL AND event_id IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot roll back 000008 while Event-only evidence exists';
    END IF;
END $$;
ALTER TABLE evidence DROP CONSTRAINT IF EXISTS evidence_requires_case_or_event;
ALTER TABLE evidence ALTER COLUMN case_id SET NOT NULL;
