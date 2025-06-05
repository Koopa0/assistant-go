-- Intelligent features migration for Assistant
-- Adds tables for learning system, advanced memory, knowledge graph, and agent collaboration

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "ltree";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- =====================================================
-- LEARNING SYSTEM TABLES
-- =====================================================

-- Learning events capture all interactions for pattern detection
CREATE TABLE IF NOT EXISTS learning_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN (
        'code_completion', 'refactoring', 'debugging', 'query_response',
        'tool_usage', 'error_recovery', 'preference_detected'
    )),
    context JSONB NOT NULL, -- Full context at time of event
    input_data JSONB NOT NULL,
    output_data JSONB,
    outcome VARCHAR(50) CHECK (outcome IN ('success', 'failure', 'partial', 'abandoned')),
    confidence FLOAT CHECK (confidence >= 0 AND confidence <= 1),
    feedback_score INTEGER CHECK (feedback_score >= -1 AND feedback_score <= 1), -- -1: negative, 0: neutral, 1: positive
    learning_metadata JSONB DEFAULT '{}', -- Features extracted, patterns detected
    duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    session_id VARCHAR(255),
    correlation_id UUID -- Links related events
);

-- Learned patterns represent discovered regularities
CREATE TABLE IF NOT EXISTS learned_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pattern_type VARCHAR(50) NOT NULL CHECK (pattern_type IN (
        'coding_style', 'tool_preference', 'workflow_sequence',
        'error_pattern', 'time_pattern', 'context_preference'
    )),
    pattern_name VARCHAR(200) NOT NULL,
    pattern_signature JSONB NOT NULL, -- Unique pattern identifier
    pattern_data JSONB NOT NULL, -- Full pattern details
    confidence FLOAT NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    occurrence_count INTEGER DEFAULT 1,
    positive_outcomes INTEGER DEFAULT 0,
    negative_outcomes INTEGER DEFAULT 0,
    last_observed TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, pattern_type, pattern_signature)
);

-- User skills track proficiency evolution
CREATE TABLE IF NOT EXISTS user_skills (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_category VARCHAR(100) NOT NULL,
    skill_name VARCHAR(200) NOT NULL,
    proficiency_level FLOAT NOT NULL CHECK (proficiency_level >= 0 AND proficiency_level <= 1),
    experience_points INTEGER DEFAULT 0,
    successful_uses INTEGER DEFAULT 0,
    total_uses INTEGER DEFAULT 0,
    last_used TIMESTAMP WITH TIME ZONE,
    learning_curve JSONB DEFAULT '[]', -- Historical proficiency data points
    related_patterns UUID[] DEFAULT '{}', -- Links to learned_patterns
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, skill_category, skill_name)
);

-- =====================================================
-- ADVANCED MEMORY SYSTEM TABLES
-- =====================================================

-- Episodic memories store specific experiences
CREATE TABLE IF NOT EXISTS episodic_memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    episode_type VARCHAR(50) NOT NULL,
    episode_summary TEXT NOT NULL,
    full_context JSONB NOT NULL,
    emotional_valence FLOAT CHECK (emotional_valence >= -1 AND emotional_valence <= 1),
    importance FLOAT NOT NULL CHECK (importance >= 0 AND importance <= 1),
    vividness FLOAT DEFAULT 1.0 CHECK (vividness >= 0 AND vividness <= 1), -- Decays over time
    embedding vector(1536), -- For similarity search
    temporal_context JSONB, -- Time of day, day of week patterns
    spatial_context JSONB, -- Project, file, function context
    social_context JSONB, -- Collaboration, team context
    causal_links UUID[], -- Links to other memories
    access_count INTEGER DEFAULT 0,
    last_accessed TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    decay_rate FLOAT DEFAULT 0.01 -- How fast vividness decreases
);

-- Semantic memories store factual knowledge
CREATE TABLE IF NOT EXISTS semantic_memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    knowledge_type VARCHAR(50) NOT NULL CHECK (knowledge_type IN (
        'concept', 'fact', 'rule', 'relationship', 'definition'
    )),
    subject VARCHAR(500) NOT NULL,
    predicate VARCHAR(500) NOT NULL,
    object JSONB NOT NULL,
    confidence FLOAT NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    source_type VARCHAR(50), -- 'learned', 'told', 'inferred', 'external'
    source_references UUID[], -- Links to episodic memories or learning events
    embedding vector(1536),
    contradiction_count INTEGER DEFAULT 0,
    confirmation_count INTEGER DEFAULT 0,
    last_confirmed TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Procedural memories store how-to knowledge
CREATE TABLE IF NOT EXISTS procedural_memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    procedure_name VARCHAR(500) NOT NULL,
    procedure_type VARCHAR(50) NOT NULL,
    trigger_conditions JSONB NOT NULL, -- When to apply this procedure
    steps JSONB NOT NULL, -- Ordered list of actions
    prerequisites JSONB DEFAULT '[]', -- Required conditions
    expected_outcomes JSONB,
    success_rate FLOAT DEFAULT 0.0,
    execution_count INTEGER DEFAULT 0,
    average_duration_ms INTEGER,
    optimization_history JSONB DEFAULT '[]', -- How procedure improved over time
    is_automated BOOLEAN DEFAULT false,
    automation_confidence FLOAT DEFAULT 0.0,
    last_executed TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Working memory for current context
CREATE TABLE IF NOT EXISTS working_memory (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NOT NULL,
    memory_slot INTEGER NOT NULL CHECK (memory_slot >= 1 AND memory_slot <= 7), -- Miller's 7±2
    content_type VARCHAR(50) NOT NULL,
    content JSONB NOT NULL,
    activation_level FLOAT DEFAULT 1.0 CHECK (activation_level >= 0 AND activation_level <= 1),
    reference_count INTEGER DEFAULT 1,
    linked_memories UUID[], -- Links to other memory types
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (NOW() + INTERVAL '2 hours'),
    UNIQUE(user_id, session_id, memory_slot)
);

-- =====================================================
-- KNOWLEDGE GRAPH TABLES
-- =====================================================

-- Knowledge nodes represent entities in the knowledge graph
CREATE TABLE IF NOT EXISTS knowledge_nodes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    node_type VARCHAR(50) NOT NULL CHECK (node_type IN (
        'project', 'technology', 'concept', 'person', 'organization',
        'pattern', 'problem', 'solution', 'tool', 'resource'
    )),
    node_name VARCHAR(500) NOT NULL,
    display_name VARCHAR(500),
    description TEXT,
    properties JSONB DEFAULT '{}',
    embedding vector(1536),
    importance FLOAT DEFAULT 0.5 CHECK (importance >= 0 AND importance <= 1),
    access_frequency FLOAT DEFAULT 0.0,
    last_accessed TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    UNIQUE(user_id, node_type, node_name)
);

-- Knowledge edges represent relationships between nodes
CREATE TABLE IF NOT EXISTS knowledge_edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_node_id UUID NOT NULL REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
    target_node_id UUID NOT NULL REFERENCES knowledge_nodes(id) ON DELETE CASCADE,
    edge_type VARCHAR(100) NOT NULL, -- 'uses', 'implements', 'depends_on', etc.
    strength FLOAT DEFAULT 0.5 CHECK (strength >= 0 AND strength <= 1),
    properties JSONB DEFAULT '{}',
    evidence_count INTEGER DEFAULT 1,
    last_observed TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    UNIQUE(source_node_id, target_node_id, edge_type)
);

-- Knowledge evolution tracks how understanding changes
CREATE TABLE IF NOT EXISTS knowledge_evolution (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL CHECK (entity_type IN ('node', 'edge', 'cluster')),
    entity_id UUID NOT NULL,
    change_type VARCHAR(50) NOT NULL CHECK (change_type IN (
        'created', 'strengthened', 'weakened', 'modified', 'split', 'merged'
    )),
    previous_state JSONB,
    new_state JSONB NOT NULL,
    change_reason TEXT,
    confidence FLOAT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- AGENT COLLABORATION TABLES
-- =====================================================

-- Agent definitions and capabilities
CREATE TABLE IF NOT EXISTS agent_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_name VARCHAR(100) NOT NULL UNIQUE,
    agent_type VARCHAR(50) NOT NULL,
    capabilities JSONB NOT NULL, -- List of capabilities with confidence scores
    expertise_domains TEXT[],
    collaboration_preferences JSONB DEFAULT '{}',
    performance_metrics JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Agent collaboration sessions
CREATE TABLE IF NOT EXISTS agent_collaborations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NOT NULL,
    lead_agent_id UUID NOT NULL REFERENCES agent_definitions(id),
    participating_agents UUID[] NOT NULL,
    collaboration_type VARCHAR(50) NOT NULL,
    task_description TEXT NOT NULL,
    task_complexity FLOAT CHECK (task_complexity >= 0 AND task_complexity <= 1),
    collaboration_plan JSONB NOT NULL,
    execution_trace JSONB DEFAULT '[]', -- Detailed execution log
    outcome VARCHAR(50),
    total_duration_ms INTEGER,
    resource_usage JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Agent knowledge sharing
CREATE TABLE IF NOT EXISTS agent_knowledge_shares (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source_agent_id UUID NOT NULL REFERENCES agent_definitions(id),
    target_agent_id UUID NOT NULL REFERENCES agent_definitions(id),
    knowledge_type VARCHAR(50) NOT NULL,
    knowledge_content JSONB NOT NULL,
    relevance_score FLOAT CHECK (relevance_score >= 0 AND relevance_score <= 1),
    was_accepted BOOLEAN,
    integration_result JSONB,
    collaboration_id UUID REFERENCES agent_collaborations(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- EVENT SOURCING TABLES
-- =====================================================

-- System events for complete audit trail
CREATE TABLE IF NOT EXISTS system_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    aggregate_id UUID NOT NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_data JSONB NOT NULL,
    event_metadata JSONB DEFAULT '{}',
    event_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    processing_error TEXT
);

-- Event projections for read models
CREATE TABLE IF NOT EXISTS event_projections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    projection_name VARCHAR(200) NOT NULL,
    last_processed_event_id UUID REFERENCES system_events(id),
    last_processed_at TIMESTAMP WITH TIME ZONE,
    projection_state JSONB NOT NULL DEFAULT '{}',
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(projection_name)
);

-- =====================================================
-- DEVELOPMENT WORKFLOW TABLES
-- =====================================================

-- Development sessions track work periods
CREATE TABLE IF NOT EXISTS development_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_type VARCHAR(50) NOT NULL,
    project_context JSONB NOT NULL,
    goals TEXT[],
    actual_outcomes TEXT[],
    interruption_count INTEGER DEFAULT 0,
    focus_score FLOAT, -- Calculated from interruptions and context switches
    productivity_metrics JSONB DEFAULT '{}',
    mood_indicators JSONB DEFAULT '{}',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    total_duration_minutes INTEGER
);

-- Code patterns track recurring code structures
CREATE TABLE IF NOT EXISTS code_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    pattern_category VARCHAR(100) NOT NULL,
    pattern_name VARCHAR(200) NOT NULL,
    pattern_ast JSONB NOT NULL, -- Abstract syntax tree representation
    usage_contexts TEXT[],
    frequency INTEGER DEFAULT 1,
    last_used TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    evolution_history JSONB DEFAULT '[]', -- How pattern changed over time
    quality_score FLOAT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Learning system indexes
CREATE INDEX idx_learning_events_user_type ON learning_events(user_id, event_type);
CREATE INDEX idx_learning_events_created ON learning_events(created_at DESC);
CREATE INDEX idx_learning_events_session ON learning_events(session_id);
CREATE INDEX idx_learning_events_correlation ON learning_events(correlation_id);

CREATE INDEX idx_learned_patterns_user_type ON learned_patterns(user_id, pattern_type);
CREATE INDEX idx_learned_patterns_active ON learned_patterns(is_active, confidence DESC);
CREATE INDEX idx_learned_patterns_signature ON learned_patterns USING GIN (pattern_signature);

CREATE INDEX idx_user_skills_user_category ON user_skills(user_id, skill_category);
CREATE INDEX idx_user_skills_proficiency ON user_skills(proficiency_level DESC);

-- Memory system indexes
CREATE INDEX idx_episodic_memories_user ON episodic_memories(user_id);
CREATE INDEX idx_episodic_memories_importance ON episodic_memories(importance DESC);
CREATE INDEX idx_episodic_memories_embedding ON episodic_memories USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_episodic_memories_temporal ON episodic_memories USING GIN (temporal_context);

CREATE INDEX idx_semantic_memories_user_type ON semantic_memories(user_id, knowledge_type);
CREATE INDEX idx_semantic_memories_subject ON semantic_memories USING GIN (to_tsvector('english', subject));
CREATE INDEX idx_semantic_memories_embedding ON semantic_memories USING ivfflat (embedding vector_cosine_ops);

CREATE INDEX idx_procedural_memories_user ON procedural_memories(user_id);
CREATE INDEX idx_procedural_memories_triggers ON procedural_memories USING GIN (trigger_conditions);
CREATE INDEX idx_procedural_memories_automated ON procedural_memories(is_automated, automation_confidence DESC);

CREATE INDEX idx_working_memory_session ON working_memory(user_id, session_id);
CREATE INDEX idx_working_memory_expires ON working_memory(expires_at);

-- Knowledge graph indexes
CREATE INDEX idx_knowledge_nodes_user_type ON knowledge_nodes(user_id, node_type);
CREATE INDEX idx_knowledge_nodes_name ON knowledge_nodes USING GIN (to_tsvector('english', node_name));
CREATE INDEX idx_knowledge_nodes_embedding ON knowledge_nodes USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_knowledge_nodes_importance ON knowledge_nodes(importance DESC);

CREATE INDEX idx_knowledge_edges_source ON knowledge_edges(source_node_id);
CREATE INDEX idx_knowledge_edges_target ON knowledge_edges(target_node_id);
CREATE INDEX idx_knowledge_edges_type ON knowledge_edges(edge_type);
CREATE INDEX idx_knowledge_edges_strength ON knowledge_edges(strength DESC);

-- Agent collaboration indexes
CREATE INDEX idx_agent_collaborations_user ON agent_collaborations(user_id);
CREATE INDEX idx_agent_collaborations_session ON agent_collaborations(session_id);
CREATE INDEX idx_agent_collaborations_lead ON agent_collaborations(lead_agent_id);

-- Event sourcing indexes
CREATE INDEX idx_system_events_aggregate ON system_events(aggregate_type, aggregate_id, event_version);
CREATE INDEX idx_system_events_type ON system_events(event_type);
CREATE INDEX idx_system_events_created ON system_events(created_at DESC);
CREATE INDEX idx_system_events_unprocessed ON system_events(processed_at) WHERE processed_at IS NULL;

-- Development workflow indexes
CREATE INDEX idx_dev_sessions_user ON development_sessions(user_id);
CREATE INDEX idx_dev_sessions_started ON development_sessions(started_at DESC);
CREATE INDEX idx_code_patterns_user_category ON code_patterns(user_id, pattern_category);

-- =====================================================
-- TRIGGERS FOR INTELLIGENT FEATURES
-- =====================================================

-- Trigger to update learned patterns from events
CREATE OR REPLACE FUNCTION update_learned_patterns_from_event()
RETURNS TRIGGER AS $$
BEGIN
    -- This is a placeholder for pattern detection logic
    -- In practice, this would be handled by the application layer
    -- or a background job that analyzes events periodically
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to decay episodic memory vividness
CREATE OR REPLACE FUNCTION decay_episodic_memory_vividness()
RETURNS TRIGGER AS $$
BEGIN
    NEW.vividness = OLD.vividness * (1 - OLD.decay_rate);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER decay_vividness_on_access
    BEFORE UPDATE ON episodic_memories
    FOR EACH ROW
    WHEN (NEW.last_accessed IS DISTINCT FROM OLD.last_accessed)
    EXECUTE FUNCTION decay_episodic_memory_vividness();

-- Trigger to update knowledge graph importance
CREATE OR REPLACE FUNCTION update_knowledge_node_importance()
RETURNS TRIGGER AS $$
BEGIN
    -- Calculate importance based on connections and access frequency
    NEW.importance = LEAST(1.0, 
        (NEW.access_frequency * 0.3) + 
        (SELECT COUNT(*)::float / 100 FROM knowledge_edges 
         WHERE source_node_id = NEW.id OR target_node_id = NEW.id) * 0.7
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_node_importance
    BEFORE UPDATE ON knowledge_nodes
    FOR EACH ROW
    WHEN (NEW.access_frequency IS DISTINCT FROM OLD.access_frequency)
    EXECUTE FUNCTION update_knowledge_node_importance();

-- Apply updated_at triggers to new tables
CREATE TRIGGER update_learned_patterns_updated_at BEFORE UPDATE ON learned_patterns FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_skills_updated_at BEFORE UPDATE ON user_skills FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_semantic_memories_updated_at BEFORE UPDATE ON semantic_memories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_procedural_memories_updated_at BEFORE UPDATE ON procedural_memories FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_knowledge_nodes_updated_at BEFORE UPDATE ON knowledge_nodes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_knowledge_edges_updated_at BEFORE UPDATE ON knowledge_edges FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_agent_definitions_updated_at BEFORE UPDATE ON agent_definitions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_event_projections_updated_at BEFORE UPDATE ON event_projections FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_code_patterns_updated_at BEFORE UPDATE ON code_patterns FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();