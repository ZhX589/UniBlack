DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM appeals WHERE case_id IS NULL) THEN
        RAISE EXCEPTION 'cannot downgrade event governance while event-only appeals exist; remove or migrate them to cases first';
    END IF;
    IF EXISTS (SELECT 1 FROM appeals WHERE deleted_at IS NOT NULL)
       OR EXISTS (SELECT 1 FROM submissions WHERE deleted_at IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot downgrade event governance while retired appeals or submissions exist; restore or hard-delete them first';
    END IF;
END $$;

DROP INDEX IF EXISTS idx_appeals_active;
DROP INDEX IF EXISTS idx_submissions_active;
ALTER TABLE appeals DROP CONSTRAINT IF EXISTS appeals_target_check;
ALTER TABLE appeals ALTER COLUMN case_id SET NOT NULL;
ALTER TABLE appeals DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE submissions DROP COLUMN IF EXISTS deleted_at;
