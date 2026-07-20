ALTER TABLE evidence DROP CONSTRAINT IF EXISTS evidence_requires_case_or_event;
-- This rollback is safe only after Event-only evidence has been migrated or removed.
ALTER TABLE evidence ALTER COLUMN case_id SET NOT NULL;
