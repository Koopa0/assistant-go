-- Drop user_api_keys table and related indexes
DROP TABLE IF EXISTS user_api_keys;

-- Drop additional indexes added in this migration
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_preferences_gin;