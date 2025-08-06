-- Additional performance indexes for common query patterns

-- Composite index for feature flag queries by service and status
CREATE INDEX idx_feature_flags_service_active 
ON feature_flags(service_name, created_at DESC) 
WHERE deleted_at IS NULL;

-- Index for active feature flags (non-deleted) ordered by most recent
CREATE INDEX idx_feature_flags_active_recent 
ON feature_flags(created_at DESC) 
WHERE deleted_at IS NULL;

-- Partial index for recently updated feature flags (last 30 days)
CREATE INDEX idx_feature_flags_recently_updated 
ON feature_flags(updated_at DESC) 
WHERE updated_at > NOW() - INTERVAL '30 days' AND deleted_at IS NULL;

-- Index for user email domain analysis (for multi-tenant scenarios)
CREATE INDEX idx_users_email_domain 
ON users((SPLIT_PART(email, '@', 2)));

-- Composite index for API key validation (hash + expiration check)
CREATE INDEX idx_api_keys_validation 
ON api_keys(key_hash, expires_at) 
WHERE expires_at IS NULL OR expires_at > NOW();

-- Index for audit log retention queries (for cleanup jobs)
CREATE INDEX idx_audit_logs_retention 
ON audit_logs(created_at) 
WHERE created_at < NOW() - INTERVAL '1 year';

-- Partial index for recent audit activity (last 7 days)
CREATE INDEX idx_audit_logs_recent_activity 
ON audit_logs(entity_type, created_at DESC) 
WHERE created_at > NOW() - INTERVAL '7 days';