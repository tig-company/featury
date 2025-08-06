-- Create audit_logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL, -- feature_flag, user, api_key
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- create, update, delete, view
    user_id UUID NOT NULL REFERENCES users(id),
    changes JSONB, -- Before/after diff for updates
    metadata JSONB, -- Request info, IP address, user agent, etc.
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create index on entity_type and entity_id for entity audit history
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);

-- Create index on user_id for user activity tracking
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);

-- Create index on action for filtering by action type
CREATE INDEX idx_audit_logs_action ON audit_logs(action);

-- Create index on created_at for time-based queries (most recent first)
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Create composite index for entity audit queries with time ordering
CREATE INDEX idx_audit_logs_entity_time ON audit_logs(entity_type, entity_id, created_at DESC);

-- Create GIN indexes for JSONB queries
CREATE INDEX idx_audit_logs_changes_gin ON audit_logs USING GIN(changes) WHERE changes IS NOT NULL;
CREATE INDEX idx_audit_logs_metadata_gin ON audit_logs USING GIN(metadata) WHERE metadata IS NOT NULL;