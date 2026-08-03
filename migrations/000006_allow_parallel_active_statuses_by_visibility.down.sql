-- Revert to single active status per user.

DROP INDEX IF EXISTS idx_statuses_user_visibility_active;

CREATE UNIQUE INDEX IF NOT EXISTS idx_statuses_user_active
    ON statuses(user_id)
    WHERE is_active = TRUE;
