-- Rollback removes password-change session revocation. Do not deploy older
-- authentication code while revoked credentials are still within their TTL.
ALTER TABLE users DROP CONSTRAINT users_auth_version_nonnegative;
ALTER TABLE users DROP COLUMN auth_version;
