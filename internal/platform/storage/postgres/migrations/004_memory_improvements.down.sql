-- Drop indexes
DROP INDEX IF EXISTS idx_memory_entries_content_trgm;
DROP INDEX IF EXISTS idx_memory_entries_content_fts;
DROP INDEX IF EXISTS idx_memory_entries_expires;
DROP INDEX IF EXISTS idx_memory_entries_last_access;
DROP INDEX IF EXISTS idx_memory_entries_importance;
DROP INDEX IF EXISTS idx_memory_entries_session;
DROP INDEX IF EXISTS idx_memory_entries_user_type;

-- Drop memory_relations table and its indexes
DROP TABLE IF EXISTS memory_relations CASCADE;

-- Drop triggers
DROP TRIGGER IF EXISTS update_memory_entries_updated_at ON memory_entries;

-- Drop updated_at column
ALTER TABLE memory_entries DROP COLUMN IF EXISTS updated_at;

-- Restore original constraint
ALTER TABLE memory_entries 
DROP CONSTRAINT IF EXISTS memory_entries_memory_type_check;

ALTER TABLE memory_entries 
ADD CONSTRAINT memory_entries_memory_type_check 
CHECK (memory_type IN ('short_term', 'long_term', 'tool', 'personalization'));