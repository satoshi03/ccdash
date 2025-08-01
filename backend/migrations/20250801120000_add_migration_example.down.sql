-- Rollback user preferences table

DROP INDEX IF EXISTS idx_user_preferences_created_at;
DROP TABLE IF EXISTS user_preferences;