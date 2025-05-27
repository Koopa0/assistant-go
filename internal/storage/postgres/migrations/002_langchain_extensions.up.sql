-- LangChain extensions migration for GoAssistant
-- Adds additional tables for comprehensive LangChain-Go integration

-- Memory entries table for LangChain memory management
CREATE TABLE IF NOT EXISTS memory_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    memory_type VARCHAR(50) NOT NULL CHECK (memory_type IN ('short_term', 'long_term', 'tool', 'personalization')),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255),
    content TEXT NOT NULL,
    importance DECIMAL(3,2) DEFAULT 0.5 CHECK (importance >= 0 AND importance <= 1),
    access_count INTEGER DEFAULT 0,
    last_access TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Agent executions table for tracking agent runs
CREATE TABLE IF NOT EXISTS agent_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_type VARCHAR(50) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    query TEXT NOT NULL,
    response TEXT,
    steps JSONB DEFAULT '[]'::jsonb,
    execution_time_ms INTEGER,
    success BOOLEAN DEFAULT false,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Chain executions table for tracking chain runs
CREATE TABLE IF NOT EXISTS chain_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chain_type VARCHAR(50) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    input TEXT NOT NULL,
    output TEXT,
    steps JSONB DEFAULT '[]'::jsonb,
    execution_time_ms INTEGER,
    tokens_used INTEGER DEFAULT 0,
    success BOOLEAN DEFAULT false,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Tool cache table for caching tool execution results
CREATE TABLE IF NOT EXISTS tool_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tool_name VARCHAR(100) NOT NULL,
    input_hash VARCHAR(64) NOT NULL,
    input_data JSONB NOT NULL,
    output_data JSONB,
    execution_time_ms INTEGER,
    success BOOLEAN DEFAULT false,
    error_message TEXT,
    hit_count INTEGER DEFAULT 0,
    last_hit TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (NOW() + INTERVAL '6 hours'),
    metadata JSONB DEFAULT '{}'::jsonb,
    UNIQUE(user_id, tool_name, input_hash)
);

-- User preferences table for personalization
CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL,
    preference_key VARCHAR(200) NOT NULL,
    preference_value JSONB NOT NULL,
    value_type VARCHAR(50) NOT NULL CHECK (value_type IN ('string', 'number', 'boolean', 'object', 'array')),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb,
    UNIQUE(user_id, category, preference_key)
);

-- User context table for contextual information
CREATE TABLE IF NOT EXISTS user_context (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    context_type VARCHAR(100) NOT NULL,
    context_key VARCHAR(200) NOT NULL,
    context_value JSONB NOT NULL,
    importance DECIMAL(3,2) DEFAULT 0.5 CHECK (importance >= 0 AND importance <= 1),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb,
    UNIQUE(user_id, context_type, context_key)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_memory_entries_user_id ON memory_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_memory_entries_type ON memory_entries(memory_type);
CREATE INDEX IF NOT EXISTS idx_memory_entries_session ON memory_entries(session_id);
CREATE INDEX IF NOT EXISTS idx_memory_entries_expires ON memory_entries(expires_at);
CREATE INDEX IF NOT EXISTS idx_memory_entries_importance ON memory_entries(importance DESC);

CREATE INDEX IF NOT EXISTS idx_agent_executions_user_id ON agent_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_type ON agent_executions(agent_type);
CREATE INDEX IF NOT EXISTS idx_agent_executions_conversation ON agent_executions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_created_at ON agent_executions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_executions_success ON agent_executions(success);

CREATE INDEX IF NOT EXISTS idx_chain_executions_user_id ON chain_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_chain_executions_type ON chain_executions(chain_type);
CREATE INDEX IF NOT EXISTS idx_chain_executions_conversation ON chain_executions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_chain_executions_created_at ON chain_executions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chain_executions_success ON chain_executions(success);

CREATE INDEX IF NOT EXISTS idx_tool_cache_user_tool ON tool_cache(user_id, tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_cache_hash ON tool_cache(input_hash);
CREATE INDEX IF NOT EXISTS idx_tool_cache_expires ON tool_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_tool_cache_hit_count ON tool_cache(hit_count DESC);

CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_user_preferences_category ON user_preferences(category);
CREATE INDEX IF NOT EXISTS idx_user_preferences_key ON user_preferences(preference_key);

CREATE INDEX IF NOT EXISTS idx_user_context_user_id ON user_context(user_id);
CREATE INDEX IF NOT EXISTS idx_user_context_type ON user_context(context_type);
CREATE INDEX IF NOT EXISTS idx_user_context_key ON user_context(context_key);
CREATE INDEX IF NOT EXISTS idx_user_context_expires ON user_context(expires_at);
CREATE INDEX IF NOT EXISTS idx_user_context_importance ON user_context(importance DESC);

-- Create triggers for updated_at columns
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON user_preferences FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_context_updated_at BEFORE UPDATE ON user_context FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
