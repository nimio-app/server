-- Add email verification fields to users table
ALTER TABLE users 
ADD COLUMN email_verified BOOLEAN DEFAULT FALSE,
ADD COLUMN verification_token VARCHAR(255),
ADD COLUMN verification_token_expires_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN verified_at TIMESTAMP WITH TIME ZONE;

-- Create index on verification token for faster lookups
CREATE INDEX idx_users_verification_token ON users(verification_token) WHERE verification_token IS NOT NULL;

-- Add comment
COMMENT ON COLUMN users.email_verified IS 'Whether the user has verified their email address';
COMMENT ON COLUMN users.verification_token IS 'Token for email verification (hashed)';
COMMENT ON COLUMN users.verification_token_expires_at IS 'Expiration time for verification token (24 hours)';
