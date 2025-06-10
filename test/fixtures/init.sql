-- PostgreSQL initialization script for testing
-- This script sets up the test database with required extensions and basic schema

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Create basic schema for testing
CREATE SCHEMA IF NOT EXISTS assistant;

-- Set search path
SET search_path TO assistant, public;

-- Create conversations table
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Create messages table
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Create embeddings table for vector storage
CREATE TABLE IF NOT EXISTS embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_type VARCHAR(100) NOT NULL,
    content_id VARCHAR(255) NOT NULL,
    content_text TEXT NOT NULL,
    embedding vector(1536), -- OpenAI embedding dimension
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create tools table
CREATE TABLE IF NOT EXISTS tools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    category VARCHAR(100),
    parameters JSONB DEFAULT '{}'::jsonb,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create tool_executions table
CREATE TABLE IF NOT EXISTS tool_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tool_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    input_parameters JSONB DEFAULT '{}'::jsonb,
    output_result JSONB DEFAULT '{}'::jsonb,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    execution_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create agents table
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(100) NOT NULL,
    description TEXT,
    capabilities JSONB DEFAULT '[]'::jsonb,
    configuration JSONB DEFAULT '{}'::jsonb,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create memory_entries table for agent memory
CREATE TABLE IF NOT EXISTS memory_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID REFERENCES agents(id) ON DELETE CASCADE,
    memory_type VARCHAR(50) NOT NULL CHECK (memory_type IN ('working', 'episodic', 'semantic', 'procedural')),
    content TEXT NOT NULL,
    embedding vector(1536),
    importance_score FLOAT DEFAULT 0.5,
    access_count INTEGER DEFAULT 0,
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(role);

CREATE INDEX IF NOT EXISTS idx_embeddings_content_type ON embeddings(content_type);
CREATE INDEX IF NOT EXISTS idx_embeddings_content_id ON embeddings(content_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_created_at ON embeddings(created_at);

-- Vector similarity search index
CREATE INDEX IF NOT EXISTS idx_embeddings_vector ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE INDEX IF NOT EXISTS idx_tool_executions_tool_name ON tool_executions(tool_name);
CREATE INDEX IF NOT EXISTS idx_tool_executions_user_id ON tool_executions(user_id);
CREATE INDEX IF NOT EXISTS idx_tool_executions_created_at ON tool_executions(created_at);
CREATE INDEX IF NOT EXISTS idx_tool_executions_success ON tool_executions(success);

CREATE INDEX IF NOT EXISTS idx_agents_type ON agents(type);
CREATE INDEX IF NOT EXISTS idx_agents_enabled ON agents(enabled);

CREATE INDEX IF NOT EXISTS idx_memory_entries_agent_id ON memory_entries(agent_id);
CREATE INDEX IF NOT EXISTS idx_memory_entries_memory_type ON memory_entries(memory_type);
CREATE INDEX IF NOT EXISTS idx_memory_entries_importance_score ON memory_entries(importance_score);
CREATE INDEX IF NOT EXISTS idx_memory_entries_last_accessed ON memory_entries(last_accessed);

-- Vector similarity search index for memory
CREATE INDEX IF NOT EXISTS idx_memory_entries_vector ON memory_entries USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Insert test data
INSERT INTO tools (name, description, category, parameters) VALUES
('go-analyzer', 'Analyzes Go code for issues and improvements', 'development', '{"path": {"type": "string", "required": true}}'),
('go-tester', 'Runs Go tests with coverage analysis', 'development', '{"path": {"type": "string", "required": true}, "coverage": {"type": "boolean", "default": true}}'),
('go-formatter', 'Formats Go code according to standards', 'development', '{"path": {"type": "string", "required": true}}'),
('go-builder', 'Builds Go applications', 'development', '{"path": {"type": "string", "required": true}, "output": {"type": "string"}}')
ON CONFLICT (name) DO NOTHING;

INSERT INTO agents (name, type, description, capabilities) VALUES
('development-agent', 'development', 'Specialized agent for software development tasks', '["code_analysis", "testing", "debugging", "refactoring"]'),
('database-agent', 'database', 'Specialized agent for database operations', '["query_optimization", "schema_analysis", "migration_assistance"]'),
('infrastructure-agent', 'infrastructure', 'Specialized agent for infrastructure management', '["kubernetes", "docker", "monitoring"]')
ON CONFLICT (name) DO NOTHING;

-- Create functions for testing
CREATE OR REPLACE FUNCTION similarity_search_embeddings(
    query_embedding vector(1536),
    content_type_filter text DEFAULT NULL,
    similarity_threshold float DEFAULT 0.7,
    max_results integer DEFAULT 10
)
RETURNS TABLE (
    id uuid,
    content_type varchar(100),
    content_id varchar(255),
    content_text text,
    similarity float,
    metadata jsonb
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.id,
        e.content_type,
        e.content_id,
        e.content_text,
        1 - (e.embedding <=> query_embedding) as similarity,
        e.metadata
    FROM embeddings e
    WHERE 
        (content_type_filter IS NULL OR e.content_type = content_type_filter)
        AND (1 - (e.embedding <=> query_embedding)) >= similarity_threshold
    ORDER BY e.embedding <=> query_embedding
    LIMIT max_results;
END;
$$ LANGUAGE plpgsql;

-- Create function for memory similarity search
CREATE OR REPLACE FUNCTION similarity_search_memory(
    agent_id_param uuid,
    query_embedding vector(1536),
    memory_type_filter text DEFAULT NULL,
    similarity_threshold float DEFAULT 0.7,
    max_results integer DEFAULT 10
)
RETURNS TABLE (
    id uuid,
    memory_type varchar(50),
    content text,
    similarity float,
    importance_score float,
    metadata jsonb
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        m.id,
        m.memory_type,
        m.content,
        1 - (m.embedding <=> query_embedding) as similarity,
        m.importance_score,
        m.metadata
    FROM memory_entries m
    WHERE 
        m.agent_id = agent_id_param
        AND (memory_type_filter IS NULL OR m.memory_type = memory_type_filter)
        AND (1 - (m.embedding <=> query_embedding)) >= similarity_threshold
    ORDER BY m.embedding <=> query_embedding
    LIMIT max_results;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA assistant TO PUBLIC;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA assistant TO PUBLIC;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA assistant TO PUBLIC;
