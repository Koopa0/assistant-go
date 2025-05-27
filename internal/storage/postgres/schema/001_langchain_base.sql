-- LangChain integration schema for GoAssistant
-- This schema supports the LangChain-Go integration with proper pgvector support

-- Enable pgvector extension for embeddings
CREATE EXTENSION IF NOT EXISTS vector;

-- Conversations table for chat history
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Messages table for individual chat messages
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Embeddings table for semantic search and RAG
CREATE TABLE IF NOT EXISTS embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_type VARCHAR(100) NOT NULL,
    content_id VARCHAR(255) NOT NULL,
    content_text TEXT NOT NULL,
    embedding vector(1536), -- OpenAI embedding dimension
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb,
    UNIQUE(content_type, content_id)
);

-- Memory entries table for LangChain memory management
CREATE TABLE IF NOT EXISTS memory_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_type VARCHAR(50) NOT NULL CHECK (memory_type IN ('short_term', 'long_term', 'tool', 'personalization')),
    user_id VARCHAR(255) NOT NULL,
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_type VARCHAR(50) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_type VARCHAR(50) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
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
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
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
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_embeddings_content_type ON embeddings(content_type);
CREATE INDEX IF NOT EXISTS idx_embeddings_content_id ON embeddings(content_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_vector ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE INDEX IF NOT EXISTS idx_memory_entries_user_id ON memory_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_memory_entries_type ON memory_entries(memory_type);
CREATE INDEX IF NOT EXISTS idx_memory_entries_session ON memory_entries(session_id);
CREATE INDEX IF NOT EXISTS idx_memory_entries_expires ON memory_entries(expires_at);

CREATE INDEX IF NOT EXISTS idx_agent_executions_user_id ON agent_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_agent_executions_type ON agent_executions(agent_type);
CREATE INDEX IF NOT EXISTS idx_agent_executions_created_at ON agent_executions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_chain_executions_user_id ON chain_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_chain_executions_type ON chain_executions(chain_type);
CREATE INDEX IF NOT EXISTS idx_chain_executions_created_at ON chain_executions(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_tool_cache_user_tool ON tool_cache(user_id, tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_cache_expires ON tool_cache(expires_at);

CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_user_preferences_category ON user_preferences(category);

CREATE INDEX IF NOT EXISTS idx_user_context_user_id ON user_context(user_id);
CREATE INDEX IF NOT EXISTS idx_user_context_type ON user_context(context_type);
CREATE INDEX IF NOT EXISTS idx_user_context_expires ON user_context(expires_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_conversations_updated_at BEFORE UPDATE ON conversations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_embeddings_updated_at BEFORE UPDATE ON embeddings FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON user_preferences FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_context_updated_at BEFORE UPDATE ON user_context FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
