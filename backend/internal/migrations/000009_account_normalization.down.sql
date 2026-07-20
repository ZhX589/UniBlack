DROP INDEX IF EXISTS idx_accounts_platform_id;
DROP INDEX IF EXISTS idx_accounts_platform_username;
CREATE UNIQUE INDEX idx_accounts_platform_id ON accounts(platform, account_id) WHERE account_id IS NOT NULL AND account_id <> '';
CREATE UNIQUE INDEX idx_accounts_platform_username ON accounts(platform, username) WHERE (account_id IS NULL OR account_id = '') AND username IS NOT NULL;
