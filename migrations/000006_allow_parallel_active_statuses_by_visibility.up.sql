-- Allow users to hold one active status per visibility tier.
-- This enables concurrent statuses, e.g. ALL_CONNECTIONS and CIRCLE_ONLY.

DROP INDEX IF EXISTS idx_statuses_user_active;

CREATE UNIQUE INDEX IF NOT EXISTS idx_statuses_user_visibility_active
    ON statuses(user_id, visibility_tier)
    WHERE is_active = TRUE;
