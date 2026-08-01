-- Rollback directional relationship tiers

-- Remove directional tier columns
ALTER TABLE connections 
    DROP COLUMN IF EXISTS user_tier,
    DROP COLUMN IF EXISTS friend_tier;

-- Restore original default
ALTER TABLE connections 
    ALTER COLUMN relationship_tier SET DEFAULT 'MUTUAL'::relationship_tier;

-- Remove comments
COMMENT ON COLUMN connections.relationship_tier IS NULL;
