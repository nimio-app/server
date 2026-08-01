-- Remove the bidirectional constraint
DROP INDEX IF EXISTS idx_connections_canonical_pair;

-- Restore the old directional constraint
ALTER TABLE connections ADD CONSTRAINT unique_connection UNIQUE (user_id, friend_id);
