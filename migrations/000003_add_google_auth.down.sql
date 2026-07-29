-- Reverse Google OAuth support
DROP INDEX IF EXISTS idx_users_google_id;

ALTER TABLE users 
DROP COLUMN IF EXISTS google_id,
DROP COLUMN IF EXISTS auth_provider;

-- Restore password_hash NOT NULL constraint
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
