-- Example migration to demonstrate the migration system
-- This adds a simple user preferences table

CREATE TABLE IF NOT EXISTS user_preferences (
    user_id VARCHAR PRIMARY KEY,
    theme VARCHAR DEFAULT 'light',
    language VARCHAR DEFAULT 'en',
    notifications_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_preferences_created_at ON user_preferences(created_at);