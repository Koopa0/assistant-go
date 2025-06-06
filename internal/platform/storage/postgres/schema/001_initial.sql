-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Conversations table
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content TEXT NOT NULL,
    token_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Embeddings table
CREATE TABLE embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content_type VARCHAR(100) NOT NULL,
    content_id VARCHAR(255) NOT NULL,
    content_text TEXT NOT NULL,
    embedding VECTOR(1536), -- Default OpenAI embedding dimension
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(content_type, content_id)
);

-- Indexes for performance
CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_created_at ON conversations(created_at DESC);
CREATE INDEX idx_conversations_updated_at ON conversations(updated_at DESC);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);
CREATE INDEX idx_messages_role ON messages(role);

CREATE INDEX idx_embeddings_content_type ON embeddings(content_type);
CREATE INDEX idx_embeddings_content_id ON embeddings(content_id);
CREATE INDEX idx_embeddings_created_at ON embeddings(created_at DESC);

-- Vector similarity search index (using HNSW for better performance)
CREATE INDEX idx_embeddings_vector_cosine ON embeddings 
USING hnsw (embedding vector_cosine_ops) 
WITH (m = 16, ef_construction = 64);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_conversations_updated_at 
    BEFORE UPDATE ON conversations 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE conversations IS 'Stores conversation contexts and metadata';
COMMENT ON TABLE messages IS 'Stores individual messages within conversations';
COMMENT ON TABLE embeddings IS 'Stores vector embeddings for semantic search';

COMMENT ON COLUMN conversations.user_id IS 'Identifier for the user who owns this conversation';
COMMENT ON COLUMN conversations.title IS 'Human-readable title for the conversation';
COMMENT ON COLUMN conversations.metadata IS 'Additional metadata stored as JSON';

COMMENT ON COLUMN messages.conversation_id IS 'Reference to the parent conversation';
COMMENT ON COLUMN messages.role IS 'Role of the message sender (user, assistant, system)';
COMMENT ON COLUMN messages.content IS 'The actual message content';
COMMENT ON COLUMN messages.token_count IS 'Number of tokens in the message content';
COMMENT ON COLUMN messages.metadata IS 'Additional message metadata stored as JSON';

COMMENT ON COLUMN embeddings.content_type IS 'Type of content (e.g., message, document, code)';
COMMENT ON COLUMN embeddings.content_id IS 'Unique identifier for the content within its type';
COMMENT ON COLUMN embeddings.content_text IS 'The original text that was embedded';
COMMENT ON COLUMN embeddings.embedding IS 'Vector embedding of the content';
COMMENT ON COLUMN embeddings.metadata IS 'Additional embedding metadata stored as JSON';
