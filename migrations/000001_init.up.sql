-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table: Core authentication data
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT email_valid CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Profiles table: Public user information
CREATE TABLE profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    username VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    avatar_url VARCHAR(500),
    bio TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT username_valid CHECK (username ~* '^[a-z0-9_]{3,50}$')
);

CREATE INDEX idx_profiles_username ON profiles(username);

-- Connection relationship tiers enum
CREATE TYPE relationship_tier AS ENUM ('ALL', 'CIRCLE', 'MUTUAL');

-- Connection status enum
CREATE TYPE connection_status AS ENUM ('PENDING', 'ACCEPTED', 'BLOCKED');

-- Connections table: Friend relationships with privacy tiers
CREATE TABLE connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    relationship_tier relationship_tier NOT NULL DEFAULT 'MUTUAL',
    status connection_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT no_self_connection CHECK (user_id != friend_id),
    CONSTRAINT unique_connection UNIQUE (user_id, friend_id)
);

CREATE INDEX idx_connections_user_id ON connections(user_id);
CREATE INDEX idx_connections_friend_id ON connections(friend_id);
CREATE INDEX idx_connections_status ON connections(status);
CREATE INDEX idx_connections_user_status ON connections(user_id, status);

-- Availability type enum
CREATE TYPE availability_type AS ENUM ('FREE', 'BUSY', 'FOCUS', 'DRIVING', 'WANT_TO_TALK');

-- Visibility tier enum
CREATE TYPE visibility_tier AS ENUM ('ALL_CONNECTIONS', 'CIRCLE_ONLY', 'CUSTOM_LIST');

-- Statuses table: User availability states with privacy controls
CREATE TABLE statuses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    availability_type availability_type NOT NULL,
    note TEXT,
    visibility_tier visibility_tier NOT NULL DEFAULT 'ALL_CONNECTIONS',
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT TRUE,
    CONSTRAINT note_length CHECK (LENGTH(note) <= 500)
);

-- One active status per user per visibility tier (enables ALL + CIRCLE in parallel)
CREATE UNIQUE INDEX idx_statuses_user_visibility_active
    ON statuses(user_id, visibility_tier)
    WHERE is_active = TRUE;
CREATE INDEX idx_statuses_user_id ON statuses(user_id);
CREATE INDEX idx_statuses_expires_at ON statuses(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_statuses_created_at ON statuses(created_at);

-- Status visibility custom lists (for CUSTOM_LIST visibility tier)
CREATE TABLE status_visibility_lists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    status_id UUID NOT NULL REFERENCES statuses(id) ON DELETE CASCADE,
    visible_to_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT unique_visibility UNIQUE (status_id, visible_to_user_id)
);

CREATE INDEX idx_status_visibility_status_id ON status_visibility_lists(status_id);
CREATE INDEX idx_status_visibility_user_id ON status_visibility_lists(visible_to_user_id);

-- Refresh tokens table: JWT refresh token management
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT token_not_expired CHECK (expires_at > created_at)
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic updated_at management
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_profiles_updated_at BEFORE UPDATE ON profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_connections_updated_at BEFORE UPDATE ON connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_statuses_updated_at BEFORE UPDATE ON statuses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to auto-expire statuses
CREATE OR REPLACE FUNCTION expire_old_statuses()
RETURNS void AS $$
BEGIN
    UPDATE statuses
    SET is_active = FALSE
    WHERE is_active = TRUE 
    AND expires_at IS NOT NULL 
    AND expires_at <= NOW();
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE users IS 'Core user authentication data';
COMMENT ON TABLE profiles IS 'User profile information visible to connections';
COMMENT ON TABLE connections IS 'Friend relationships with privacy tier controls';
COMMENT ON TABLE statuses IS 'Intentional availability states - tracks emotional readiness, not passive activity';
COMMENT ON TABLE status_visibility_lists IS 'Custom visibility controls for granular status sharing';
COMMENT ON TABLE refresh_tokens IS 'JWT refresh token storage for secure session management';
