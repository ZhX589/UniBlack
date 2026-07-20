ALTER TABLE appeals ADD COLUMN IF NOT EXISTS event_id UUID REFERENCES events(id) ON DELETE SET NULL;
ALTER TABLE appeals ADD COLUMN IF NOT EXISTS outcome VARCHAR(32);
ALTER TABLE appeals ADD COLUMN IF NOT EXISTS resolution_reason TEXT;
CREATE INDEX IF NOT EXISTS idx_appeals_event ON appeals(event_id);

CREATE TABLE IF NOT EXISTS sanctions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL CHECK (type IN ('warning', 'submission_suspension', 'submission_ban')),
    reason TEXT NOT NULL,
    related_event_id UUID REFERENCES events(id) ON DELETE SET NULL,
    related_appeal_id UUID REFERENCES appeals(id) ON DELETE SET NULL,
    starts_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ends_at TIMESTAMPTZ,
    imposed_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES users(id) ON DELETE SET NULL,
    revoke_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (ends_at IS NULL OR ends_at > starts_at),
    CHECK (type <> 'submission_suspension' OR ends_at IS NOT NULL)
);
CREATE INDEX IF NOT EXISTS idx_sanctions_user_active ON sanctions(user_id, starts_at, ends_at) WHERE revoked_at IS NULL;
