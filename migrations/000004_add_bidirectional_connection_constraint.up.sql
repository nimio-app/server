-- Drop the old directional constraint
ALTER TABLE connections DROP CONSTRAINT IF EXISTS unique_connection;

-- Add a unique constraint on canonical pair (bidirectional)
-- This prevents (A,B) and (B,A) from both existing
CREATE UNIQUE INDEX idx_connections_canonical_pair 
ON connections(LEAST(user_id, friend_id), GREATEST(user_id, friend_id));

-- Add comment for clarity
COMMENT ON INDEX idx_connections_canonical_pair IS 'Ensures bidirectional uniqueness: prevents both (A,B) and (B,A) from existing';
