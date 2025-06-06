-- Add missing columns to memory_entries table
ALTER TABLE memory_entries 
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_memory_entries_updated_at 
BEFORE UPDATE ON memory_entries 
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

-- Create memory_relations table for relationships between memories
CREATE TABLE IF NOT EXISTS memory_relations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_memory_id UUID NOT NULL REFERENCES memory_entries(id) ON DELETE CASCADE,
    to_memory_id UUID NOT NULL REFERENCES memory_entries(id) ON DELETE CASCADE,
    relation_type VARCHAR(50) NOT NULL CHECK (relation_type IN (
        'cause', 'sequence', 'similar', 'contains', 'related'
    )),
    weight FLOAT DEFAULT 0.5 CHECK (weight >= 0 AND weight <= 1),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(from_memory_id, to_memory_id, relation_type)
);

-- Add trigger for memory_relations updated_at
CREATE TRIGGER update_memory_relations_updated_at 
BEFORE UPDATE ON memory_relations 
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();

-- Indexes for memory_entries performance
CREATE INDEX IF NOT EXISTS idx_memory_entries_user_type 
ON memory_entries(user_id, memory_type);

CREATE INDEX IF NOT EXISTS idx_memory_entries_session 
ON memory_entries(session_id) 
WHERE session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_memory_entries_importance 
ON memory_entries(importance DESC);

CREATE INDEX IF NOT EXISTS idx_memory_entries_last_access 
ON memory_entries(last_access DESC);

CREATE INDEX IF NOT EXISTS idx_memory_entries_expires 
ON memory_entries(expires_at) 
WHERE expires_at IS NOT NULL;

-- Full-text search index on content
CREATE INDEX IF NOT EXISTS idx_memory_entries_content_fts 
ON memory_entries USING GIN (to_tsvector('english', content));

-- Indexes for memory_relations
CREATE INDEX IF NOT EXISTS idx_memory_relations_from 
ON memory_relations(from_memory_id);

CREATE INDEX IF NOT EXISTS idx_memory_relations_to 
ON memory_relations(to_memory_id);

CREATE INDEX IF NOT EXISTS idx_memory_relations_type 
ON memory_relations(relation_type);

CREATE INDEX IF NOT EXISTS idx_memory_relations_weight 
ON memory_relations(weight DESC);

-- Enable pg_trgm extension for similarity search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Index for similarity search
CREATE INDEX IF NOT EXISTS idx_memory_entries_content_trgm 
ON memory_entries USING GIN (content gin_trgm_ops);

-- Update memory_type constraint to include new types
ALTER TABLE memory_entries 
DROP CONSTRAINT IF EXISTS memory_entries_memory_type_check;

ALTER TABLE memory_entries 
ADD CONSTRAINT memory_entries_memory_type_check 
CHECK (memory_type IN (
    'short_term', 'long_term', 'tool', 'personalization',
    'working', 'episodic', 'semantic', 'procedural'
));