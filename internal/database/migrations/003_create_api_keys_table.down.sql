-- Drop api_keys table and associated objects
DROP INDEX IF EXISTS idx_api_keys_permissions_gin;
DROP INDEX IF EXISTS idx_api_keys_last_used_at;
DROP INDEX IF EXISTS idx_api_keys_expires_at;
DROP INDEX IF EXISTS idx_api_keys_user_id;
DROP INDEX IF EXISTS idx_api_keys_key_hash;
DROP TABLE IF EXISTS api_keys;