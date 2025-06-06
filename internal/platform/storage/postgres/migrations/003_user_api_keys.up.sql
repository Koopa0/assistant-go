-- Create user_api_keys table for API key management
CREATE TABLE user_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    permissions JSONB NOT NULL DEFAULT '["read"]',
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired')),
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, name)
);

-- Create indexes for API keys
CREATE INDEX idx_user_api_keys_user_id ON user_api_keys(user_id);
CREATE INDEX idx_user_api_keys_key_hash ON user_api_keys(key_hash);
CREATE INDEX idx_user_api_keys_status ON user_api_keys(status);
CREATE INDEX idx_user_api_keys_expires_at ON user_api_keys(expires_at);

-- Add trigger for updated_at
CREATE TRIGGER update_user_api_keys_updated_at 
    BEFORE UPDATE ON user_api_keys 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add some additional indexes to users table for better performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

-- Add GIN index for user preferences JSONB
CREATE INDEX IF NOT EXISTS idx_users_preferences_gin ON users USING GIN(preferences);