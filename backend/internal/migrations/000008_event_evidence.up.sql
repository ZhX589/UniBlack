-- Phase 13B: Event-native evidence may not have a legacy Case relation.
ALTER TABLE evidence ALTER COLUMN case_id DROP NOT NULL;
ALTER TABLE evidence ADD CONSTRAINT evidence_requires_case_or_event
    CHECK (case_id IS NOT NULL OR event_id IS NOT NULL);
