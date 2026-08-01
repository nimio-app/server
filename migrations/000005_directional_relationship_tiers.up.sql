-- Add directional relationship tier fields
-- This enables per-user circle membership (A can have B in circle without B having A in circle)

-- Add new columns for directional tiers
ALTER TABLE connections 
    ADD COLUMN user_tier relationship_tier,
    ADD COLUMN friend_tier relationship_tier;

-- Migrate existing data:
-- - MUTUAL → ALL for both users
-- - ALL → ALL for both users  
-- - CIRCLE → CIRCLE for both users (will need manual adjustment for directional behavior later)
UPDATE connections 
SET 
    user_tier = CASE 
        WHEN relationship_tier = 'MUTUAL' THEN 'ALL'::relationship_tier
        ELSE relationship_tier 
    END,
    friend_tier = CASE 
        WHEN relationship_tier = 'MUTUAL' THEN 'ALL'::relationship_tier
        ELSE relationship_tier 
    END;

-- Make new columns NOT NULL after migration
ALTER TABLE connections 
    ALTER COLUMN user_tier SET NOT NULL,
    ALTER COLUMN friend_tier SET NOT NULL;

-- Set default for new connections to ALL
ALTER TABLE connections 
    ALTER COLUMN user_tier SET DEFAULT 'ALL'::relationship_tier,
    ALTER COLUMN friend_tier SET DEFAULT 'ALL'::relationship_tier;

-- Keep old column for backward compatibility during transition
-- (Can be dropped in a future migration after full rollout)
ALTER TABLE connections 
    ALTER COLUMN relationship_tier SET DEFAULT 'ALL'::relationship_tier;

-- Add comment explaining the new model
COMMENT ON COLUMN connections.user_tier IS 'Tier that user_id assigns to friend_id (directional)';
COMMENT ON COLUMN connections.friend_tier IS 'Tier that friend_id assigns to user_id (directional)';
COMMENT ON COLUMN connections.relationship_tier IS 'DEPRECATED: Use user_tier and friend_tier for directional tiers';

-- Remove MUTUAL from enum (after migration, clients should use ALL/CIRCLE only)
-- Note: Cannot remove enum value that's in use, but we've migrated all MUTUAL → ALL
-- Future migration can create new enum without MUTUAL and swap it
