-- Create api_keys table
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash VARCHAR(255) NOT NULL UNIQUE, -- Stores hashed API key for security
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]'::jsonb,
    expires_at TIMESTAMP WITH TIME ZONE NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE NULL
);

-- Create unique index on key_hash for fast authentication lookups
CREATE UNIQUE INDEX idx_api_keys_key_hash ON api_keys(key_hash);

-- Create index on user_id for filtering by user
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

-- Create index on expires_at for cleanup of expired keys
CREATE INDEX idx_api_keys_expires_at ON api_keys(expires_at) WHERE expires_at IS NOT NULL;

-- Create index on last_used_at for analytics
CREATE INDEX idx_api_keys_last_used_at ON api_keys(last_used_at);

-- Create GIN index for permissions queries
CREATE INDEX idx_api_keys_permissions_gin ON api_keys USING GIN(permissions);