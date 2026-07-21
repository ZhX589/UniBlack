-- Normalize common account identity values before enforcing normalized uniqueness.
UPDATE accounts
SET platform = lower(btrim(platform)),
    username = CASE WHEN username IS NULL THEN NULL ELSE lower(btrim(username)) END,
    account_id = CASE WHEN account_id IS NULL THEN NULL ELSE lower(btrim(account_id)) END;

DROP INDEX IF EXISTS idx_accounts_platform_id;
DROP INDEX IF EXISTS idx_accounts_platform_username;
CREATE UNIQUE INDEX idx_accounts_platform_id
    ON accounts (lower(btrim(platform)), lower(btrim(account_id)))
    WHERE account_id IS NOT NULL AND btrim(account_id) <> '';
CREATE UNIQUE INDEX idx_accounts_platform_username
    ON accounts (lower(btrim(platform)), lower(btrim(username)))
    WHERE (account_id IS NULL OR btrim(account_id) = '') AND username IS NOT NULL;
