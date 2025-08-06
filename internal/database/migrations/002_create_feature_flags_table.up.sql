-- Create feature_flags table
CREATE TABLE feature_flags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    description TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE NULL,
    
    -- Environment configurations stored as JSONB for flexibility
    environments JSONB NOT NULL DEFAULT '{}'::jsonb
);

-- Create unique constraint for name within service (excluding soft deleted)
CREATE UNIQUE INDEX idx_feature_flags_name_service_unique 
ON feature_flags(name, service_name) WHERE deleted_at IS NULL;

-- Create index for queries by service
CREATE INDEX idx_feature_flags_service ON feature_flags(service_name) WHERE deleted_at IS NULL;

-- Create index for created_by for filtering by creator
CREATE INDEX idx_feature_flags_created_by ON feature_flags(created_by);

-- Create index for soft deletion queries
CREATE INDEX idx_feature_flags_deleted_at ON feature_flags(deleted_at);

-- Create GIN index for JSONB environment queries
CREATE INDEX idx_feature_flags_environments_gin ON feature_flags USING GIN(environments);

-- Create trigger for updating updated_at timestamp
CREATE TRIGGER update_feature_flags_updated_at BEFORE UPDATE ON feature_flags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();