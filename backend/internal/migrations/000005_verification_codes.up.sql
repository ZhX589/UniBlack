-- Email verification codes for registration / identity verification

CREATE TABLE IF NOT EXISTS verification_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    code VARCHAR(32) NOT NULL,
    purpose VARCHAR(50) NOT NULL DEFAULT 'register',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_verification_codes_email_purpose
    ON verification_codes(email, purpose);

CREATE INDEX IF NOT EXISTS idx_verification_codes_expires
    ON verification_codes(expires_at);
