-- Drop feature_flags table and associated objects
DROP TRIGGER IF EXISTS update_feature_flags_updated_at ON feature_flags;
DROP INDEX IF EXISTS idx_feature_flags_environments_gin;
DROP INDEX IF EXISTS idx_feature_flags_deleted_at;
DROP INDEX IF EXISTS idx_feature_flags_created_by;
DROP INDEX IF EXISTS idx_feature_flags_service;
DROP INDEX IF EXISTS idx_feature_flags_name_service_unique;
DROP TABLE IF EXISTS feature_flags;