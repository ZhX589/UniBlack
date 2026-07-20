-- Phase 13A: subject public_id, accounts, events (compatible with existing cases/identifiers)

-- 1) Subject public ID
-- Historical UUID backfill needs 36 characters including UBS_. New IDs use 30.
ALTER TABLE subjects ADD COLUMN IF NOT EXISTS public_id VARCHAR(40);
UPDATE subjects
SET public_id = 'UBS_' || upper(replace(id::text, '-', ''))
WHERE public_id IS NULL OR public_id = '';
-- Historical backfill is intentionally retained; future writes use UBS_<ULID>.
ALTER TABLE subjects ALTER COLUMN public_id SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_subjects_public_id ON subjects(public_id);

-- 2) Accounts table (new model; identifiers kept for compatibility)
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    platform_label VARCHAR(100),
    account_type VARCHAR(20) NOT NULL DEFAULT 'username',
    username VARCHAR(255),
    account_id VARCHAR(255),
    custom_attributes JSONB NOT NULL DEFAULT '{}',
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (username IS NOT NULL OR account_id IS NOT NULL)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_platform_id
    ON accounts(platform, account_id)
    WHERE account_id IS NOT NULL AND account_id <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_platform_username
    ON accounts(platform, username)
    WHERE (account_id IS NULL OR account_id = '') AND username IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_subject ON accounts(subject_id);

-- Best-effort backfill from identifiers into accounts (username as value)
INSERT INTO accounts (subject_id, platform, platform_label, account_type, username, account_id, is_primary, created_at)
SELECT
    i.subject_id,
    lower(i.platform),
    i.label,
    COALESCE(NULLIF(i.account_type, ''), 'username'),
    CASE WHEN i.account_type = 'id' THEN NULL ELSE i.value END,
    CASE WHEN i.account_type = 'id' THEN i.value ELSE NULL END,
    i.is_primary,
    i.created_at
FROM identifiers i
WHERE NOT EXISTS (
    SELECT 1 FROM accounts a
    WHERE a.subject_id = i.subject_id
      AND a.platform = lower(i.platform)
      AND (
        (a.account_id IS NOT NULL AND a.account_id = i.value)
        OR (a.username IS NOT NULL AND a.username = i.value)
      )
)
ON CONFLICT DO NOTHING;

-- 3) Events table (new model; cases kept for compatibility)
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    legacy_case_id UUID UNIQUE REFERENCES cases(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    occurred_from TIMESTAMPTZ,
    occurred_to TIMESTAMPTZ,
    details TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'published',
    severity SMALLINT NOT NULL DEFAULT 1 CHECK (severity BETWEEN 1 AND 5),
    submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    correction_note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (occurred_to IS NULL OR occurred_from IS NULL OR occurred_to >= occurred_from)
);

CREATE INDEX IF NOT EXISTS idx_events_subject ON events(subject_id);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);

-- Backfill events from cases
INSERT INTO events (id, subject_id, legacy_case_id, title, details, status, severity, submitted_by, created_at, updated_at)
SELECT
    c.id,
    c.subject_id,
    c.id,
    c.title,
    COALESCE(c.description, ''),
    CASE
        WHEN c.status IN ('approved', 'closed') THEN 'published'
        WHEN c.status = 'rejected' THEN 'withdrawn'
        WHEN c.status = 'draft' THEN 'draft'
        ELSE 'published'
    END,
    GREATEST(1, LEAST(5, COALESCE(c.severity, 1))),
    c.submitted_by,
    c.created_at,
    c.updated_at
FROM cases c
WHERE NOT EXISTS (SELECT 1 FROM events e WHERE e.legacy_case_id = c.id)
ON CONFLICT DO NOTHING;

-- 4) Evidence extensions for archive keys (keep case_id for compatibility)
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES events(id) ON DELETE SET NULL;
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS storage_key VARCHAR(512);
ALTER TABLE evidence ADD COLUMN IF NOT EXISTS original_filename VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_evidence_event ON evidence(event_id);

UPDATE evidence e
SET event_id = c.id
FROM cases c
WHERE e.case_id = c.id AND e.event_id IS NULL;
