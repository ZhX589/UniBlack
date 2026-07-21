CREATE TABLE IF NOT EXISTS sanction_appeals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sanction_id UUID NOT NULL REFERENCES sanctions(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected')),
    submitted_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    review_notes TEXT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (sanction_id)
);

CREATE INDEX IF NOT EXISTS idx_sanction_appeals_status ON sanction_appeals(status);
CREATE INDEX IF NOT EXISTS idx_sanction_appeals_user ON sanction_appeals(submitted_by);
