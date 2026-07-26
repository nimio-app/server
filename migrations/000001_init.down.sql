-- Drop triggers
DROP TRIGGER IF EXISTS update_statuses_updated_at ON statuses;
DROP TRIGGER IF EXISTS update_connections_updated_at ON connections;
DROP TRIGGER IF EXISTS update_profiles_updated_at ON profiles;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop functions
DROP FUNCTION IF EXISTS expire_old_statuses();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (in reverse order of dependencies)
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS status_visibility_lists;
DROP TABLE IF EXISTS statuses;
DROP TABLE IF EXISTS connections;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS users;

-- Drop enums
DROP TYPE IF EXISTS visibility_tier;
DROP TYPE IF EXISTS availability_type;
DROP TYPE IF EXISTS connection_status;
DROP TYPE IF EXISTS relationship_tier;

-- Drop extensions
DROP EXTENSION IF EXISTS "uuid-ossp";
