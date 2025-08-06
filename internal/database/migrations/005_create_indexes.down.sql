-- Drop additional performance indexes
DROP INDEX IF EXISTS idx_audit_logs_recent_activity;
DROP INDEX IF EXISTS idx_audit_logs_retention;
DROP INDEX IF EXISTS idx_api_keys_validation;
DROP INDEX IF EXISTS idx_users_email_domain;
DROP INDEX IF EXISTS idx_feature_flags_recently_updated;
DROP INDEX IF EXISTS idx_feature_flags_active_recent;
DROP INDEX IF EXISTS idx_feature_flags_service_active;