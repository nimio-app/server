-- Add Google OAuth support to users table
ALTER TABLE users 
ADD COLUMN google_id VARCHAR(255) UNIQUE,
ADD COLUMN auth_provider VARCHAR(20) DEFAULT 'email' CHECK (auth_provider IN ('email', 'google'));

-- Index for Google ID lookups
CREATE INDEX idx_users_google_id ON users(google_id);

-- Make password_hash nullable for OAuth users
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
