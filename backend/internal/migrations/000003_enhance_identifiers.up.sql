-- Phase 3 Update: Enhance identifiers table for social accounts

-- Create new identifiers table with enhanced structure
CREATE TABLE IF NOT EXISTS identifiers_new (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    account_type VARCHAR(50) NOT NULL DEFAULT 'username',
    value VARCHAR(255) NOT NULL,
    label VARCHAR(100),
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(platform, value)
);

-- Copy data from old table (if exists)
INSERT INTO identifiers_new (id, subject_id, platform, account_type, value, is_primary, created_at)
SELECT 
    id, 
    subject_id,
    CASE 
        WHEN type = 'minecraft_uuid' THEN 'minecraft'
        ELSE type
    END as platform,
    'username' as account_type,
    value,
    is_primary,
    created_at
FROM identifiers
WHERE EXISTS (SELECT 1 FROM identifiers LIMIT 1);

-- Drop old table and indexes
DROP INDEX IF EXISTS idx_identifier_subject;
DROP INDEX IF EXISTS idx_identifier_type_value;
DROP TABLE IF EXISTS identifiers;

-- Rename new table
ALTER TABLE identifiers_new RENAME TO identifiers;

-- Create indexes
CREATE INDEX idx_identifier_subject ON identifiers(subject_id);
CREATE INDEX idx_identifier_platform ON identifiers(platform);
CREATE INDEX idx_identifier_platform_value ON identifiers(platform, value);
