-- Revoke access and refresh credentials atomically with password changes.
ALTER TABLE users ADD COLUMN auth_version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD CONSTRAINT users_auth_version_nonnegative CHECK (auth_version >= 0);
