-- Rollback: Restore original identifiers table

CREATE TABLE IF NOT EXISTS identifiers_old (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    value VARCHAR(255) NOT NULL,
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(type, value)
);

-- Copy data back (map platform to type)
INSERT INTO identifiers_old (id, subject_id, type, value, is_primary, created_at)
SELECT 
    id,
    subject_id,
    CASE 
        WHEN platform = 'minecraft' THEN 'minecraft_uuid'
        WHEN platform = 'custom' THEN 'other'
        ELSE platform
    END as type,
    value,
    is_primary,
    created_at
FROM identifiers;

-- Drop new table and indexes
DROP INDEX IF EXISTS idx_identifier_subject;
DROP INDEX IF EXISTS idx_identifier_platform;
DROP INDEX IF EXISTS idx_identifier_platform_value;
DROP TABLE IF EXISTS identifiers;

-- Rename old table back
ALTER TABLE identifiers_old RENAME TO identifiers;

-- Create original indexes
CREATE INDEX idx_identifier_subject ON identifiers(subject_id);
CREATE INDEX idx_identifier_type_value ON identifiers(type, value);
