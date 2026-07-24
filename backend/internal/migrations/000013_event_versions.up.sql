-- Append-only event version history for governance audit trail.

CREATE TABLE IF NOT EXISTS event_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    severity SMALLINT NOT NULL DEFAULT 1 CHECK (severity BETWEEN 1 AND 5),
    status VARCHAR(32) NOT NULL,
    change_type VARCHAR(20) NOT NULL DEFAULT 'create',
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(event_id, version_number)
);

CREATE INDEX IF NOT EXISTS idx_event_versions_event ON event_versions(event_id);
CREATE INDEX IF NOT EXISTS idx_event_versions_created ON event_versions(event_id, created_at DESC);

-- Backfill version 1 for existing events
INSERT INTO event_versions (event_id, version_number, title, details, severity, status, change_type, changed_by, created_at)
SELECT id, 1, title, details, severity, status, 'backfill', submitted_by, created_at
FROM events
WHERE NOT EXISTS (SELECT 1 FROM event_versions WHERE event_versions.event_id = events.id);
