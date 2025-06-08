// Package collaboration provides collaboration services for the Assistant API server.
package collaboration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// CollaborationService handles collaboration logic
type CollaborationService struct {
	db      *sqlc.Queries
	logger  *slog.Logger
	metrics *observability.Metrics
}

// NewCollaborationService creates a new collaboration service
func NewCollaborationService(db *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *CollaborationService {
	return &CollaborationService{
		db:      db,
		logger:  observability.ServerLogger(logger, "collaboration_service"),
		metrics: metrics,
	}
}

// Agent represents an AI agent
type Agent struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Type             string                 `json:"type"`
	Capabilities     []string               `json:"capabilities"`
	ExpertiseDomains []string               `json:"expertise_domains"`
	CurrentWorkload  float64                `json:"current_workload"` // 0-1
	Status           string                 `json:"status"`           // available, busy, offline
	PerformanceScore float64                `json:"performance_score"`
	LastActive       time.Time              `json:"last_active"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// AgentSession represents an agent collaboration session
type AgentSession struct {
	ID                  string                 `json:"id"`
	SessionType         string                 `json:"session_type"`
	LeadAgent           *Agent                 `json:"lead_agent"`
	ParticipatingAgents []Agent                `json:"participating_agents"`
	TaskDescription     string                 `json:"task_description"`
	TaskComplexity      float64                `json:"task_complexity"`
	Status              string                 `json:"status"` // planning, executing, completed, failed
	CollaborationPlan   CollaborationPlan      `json:"collaboration_plan"`
	Progress            float64                `json:"progress"` // 0-1
	StartedAt           time.Time              `json:"started_at"`
	CompletedAt         *time.Time             `json:"completed_at,omitempty"`
	Duration            *time.Duration         `json:"duration,omitempty"`
	Results             map[string]interface{} `json:"results,omitempty"`
}

// CollaborationPlan represents a collaboration plan
type CollaborationPlan struct {
	Phases          []CollaborationPhase   `json:"phases"`
	Dependencies    []TaskDependency       `json:"dependencies"`
	EstimatedTime   time.Duration          `json:"estimated_time"`
	ResourceNeeds   map[string]interface{} `json:"resource_needs"`
	SuccessCriteria []string               `json:"success_criteria"`
}

// CollaborationPhase represents a phase in collaboration
type CollaborationPhase struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Agent     string    `json:"agent"`
	Tasks     []string  `json:"tasks"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
}

// TaskDependency represents a task dependency
type TaskDependency struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"` // blocks, requires, optional
}

// SharedKnowledge represents shared knowledge
type SharedKnowledge struct {
	ID             string                 `json:"id"`
	Title          string                 `json:"title"`
	Content        string                 `json:"content"`
	Type           string                 `json:"type"`
	SharedBy       string                 `json:"shared_by"`
	SharedWith     []string               `json:"shared_with"`
	Visibility     string                 `json:"visibility"` // private, team, public
	Tags           []string               `json:"tags"`
	UseCount       int                    `json:"use_count"`
	EffectiveScore float64                `json:"effective_score"`
	SharedAt       time.Time              `json:"shared_at"`
	LastUsed       *time.Time             `json:"last_used,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ListAgents returns available agents
func (s *CollaborationService) ListAgents(ctx context.Context, agentType string, status string) ([]Agent, error) {
	s.logger.Debug("Listing agents",
		slog.String("agent_type", agentType),
		slog.String("status", status))

	// Query agent definitions from database
	var agentTypes []string
	if agentType != "" {
		agentTypes = []string{agentType}
	}

	agentDefs, err := s.db.GetAgentDefinitions(ctx, agentTypes)
	if err != nil {
		s.logger.Error("Failed to get agent definitions", slog.Any("error", err))
		return nil, fmt.Errorf("get agent definitions: %w", err)
	}

	// Convert to API model
	agents := make([]Agent, 0, len(agentDefs))
	for _, def := range agentDefs {
		agent, err := s.convertAgentDefinitionToAgent(def)
		if err != nil {
			s.logger.Error("Failed to convert agent definition",
				slog.String("agent_id", def.ID.String()),
				slog.Any("error", err))
			continue
		}

		// Apply status filter if specified
		if status != "" && agent.Status != status {
			continue
		}

		agents = append(agents, agent)
	}

	// If no agents found, provide default demo agents
	if len(agents) == 0 {
		s.logger.Info("No agents found in database, providing demo agents")
		agents = s.getDemoAgents()

		// Apply filters to demo agents
		if agentType != "" {
			filtered := []Agent{}
			for _, agent := range agents {
				if agent.Type == agentType {
					filtered = append(filtered, agent)
				}
			}
			agents = filtered
		}

		if status != "" {
			filtered := []Agent{}
			for _, agent := range agents {
				if agent.Status == status {
					filtered = append(filtered, agent)
				}
			}
			agents = filtered
		}
	}

	return agents, nil
}

// CreateAgentSession creates a new collaboration session
func (s *CollaborationService) CreateAgentSession(ctx context.Context, sessionType, taskDesc string, requiredAgents []string) (*AgentSession, error) {
	// TODO: Implement actual session creation logic
	agents, _ := s.ListAgents(ctx, "", "available")

	// Select lead agent
	leadAgent := &agents[0]

	// Select participating agents
	participating := []Agent{}
	if len(agents) > 1 {
		participating = append(participating, agents[1])
		if len(agents) > 2 {
			participating = append(participating, agents[2])
		}
	}

	// Create collaboration plan
	plan := CollaborationPlan{
		Phases: []CollaborationPhase{
			{
				ID:        "phase-1",
				Name:      "分析階段",
				Agent:     leadAgent.ID,
				Tasks:     []string{"需求分析", "可行性評估", "技術選型"},
				StartTime: time.Now(),
				EndTime:   time.Now().Add(30 * time.Minute),
				Status:    "pending",
			},
			{
				ID:        "phase-2",
				Name:      "實施階段",
				Agent:     participating[0].ID,
				Tasks:     []string{"開發實作", "測試編寫", "文檔更新"},
				StartTime: time.Now().Add(30 * time.Minute),
				EndTime:   time.Now().Add(90 * time.Minute),
				Status:    "pending",
			},
		},
		Dependencies: []TaskDependency{
			{From: "phase-1", To: "phase-2", Type: "blocks"},
		},
		EstimatedTime: 90 * time.Minute,
		ResourceNeeds: map[string]interface{}{
			"compute": "medium",
			"memory":  "4GB",
			"tools":   []string{"compiler", "debugger", "profiler"},
		},
		SuccessCriteria: []string{
			"所有測試通過",
			"代碼覆蓋率 > 80%",
			"性能指標達標",
		},
	}

	session := &AgentSession{
		ID:                  fmt.Sprintf("session-%d", time.Now().Unix()),
		SessionType:         sessionType,
		LeadAgent:           leadAgent,
		ParticipatingAgents: participating,
		TaskDescription:     taskDesc,
		TaskComplexity:      0.7,
		Status:              "planning",
		CollaborationPlan:   plan,
		Progress:            0.0,
		StartedAt:           time.Now(),
	}

	return session, nil
}

// GetAgentSession retrieves a collaboration session
func (s *CollaborationService) GetAgentSession(ctx context.Context, sessionID string) (*AgentSession, error) {
	s.logger.Debug("Getting collaboration session", slog.String("session_id", sessionID))

	// Parse session ID to UUID
	parsedUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}
	sessionUUID := pgtype.UUID{Bytes: parsedUUID, Valid: true}

	// Get collaboration from database
	collabRow, err := s.db.GetAgentCollaboration(ctx, sessionUUID)
	if err != nil {
		s.logger.Error("Failed to get collaboration", slog.Any("error", err))
		return nil, fmt.Errorf("get collaboration: %w", err)
	}

	// Get lead agent details
	leadAgent, err := s.db.GetAgentDefinition(ctx, collabRow.LeadAgentID)
	if err != nil {
		s.logger.Error("Failed to get lead agent", slog.Any("error", err))
		return nil, fmt.Errorf("get lead agent: %w", err)
	}

	// Convert lead agent
	leadAgentModel, err := s.convertAgentDefinitionToAgent(leadAgent)
	if err != nil {
		return nil, fmt.Errorf("convert lead agent: %w", err)
	}

	// Get participating agents
	participatingAgents := make([]Agent, 0, len(collabRow.ParticipatingAgents))
	for _, agentID := range collabRow.ParticipatingAgents {
		if agentID.Valid {
			agentDef, err := s.db.GetAgentDefinition(ctx, agentID)
			if err != nil {
				s.logger.Error("Failed to get participating agent",
					slog.String("agent_id", agentID.String()),
					slog.Any("error", err))
				continue
			}

			agentModel, err := s.convertAgentDefinitionToAgent(agentDef)
			if err != nil {
				s.logger.Error("Failed to convert participating agent",
					slog.String("agent_id", agentID.String()),
					slog.Any("error", err))
				continue
			}

			participatingAgents = append(participatingAgents, agentModel)
		}
	}

	// Parse collaboration plan
	var plan CollaborationPlan
	if len(collabRow.CollaborationPlan) > 0 {
		if err := json.Unmarshal(collabRow.CollaborationPlan, &plan); err != nil {
			s.logger.Error("Failed to parse collaboration plan", slog.Any("error", err))
		}
	}

	// Calculate progress
	progress := 0.0
	status := "planning"
	if collabRow.CompletedAt.Valid {
		progress = 1.0
		status = "completed"
		if collabRow.Outcome.Valid && collabRow.Outcome.String == "failure" {
			status = "failed"
		}
	} else if len(collabRow.ExecutionTrace) > 0 {
		progress = 0.5 // Simple progress estimation
		status = "executing"
	}

	// Calculate duration
	var duration *time.Duration
	if collabRow.CompletedAt.Valid {
		d := collabRow.CompletedAt.Time.Sub(collabRow.CreatedAt)
		duration = &d
	}

	// Parse results from execution trace or outcome
	results := make(map[string]interface{})
	if len(collabRow.ExecutionTrace) > 0 {
		var trace map[string]interface{}
		if err := json.Unmarshal(collabRow.ExecutionTrace, &trace); err == nil {
			if r, ok := trace["results"].(map[string]interface{}); ok {
				results = r
			}
		}
	}

	return &AgentSession{
		ID:                  sessionID,
		SessionType:         collabRow.CollaborationType,
		LeadAgent:           &leadAgentModel,
		ParticipatingAgents: participatingAgents,
		TaskDescription:     collabRow.TaskDescription,
		TaskComplexity:      collabRow.TaskComplexity.Float64,
		Status:              status,
		CollaborationPlan:   plan,
		Progress:            progress,
		StartedAt:           collabRow.CreatedAt,
		CompletedAt:         s.pgTimestamptzToTimePtr(collabRow.CompletedAt),
		Duration:            duration,
		Results:             results,
	}, nil
}

// AssignAgentTask assigns a task to agents
func (s *CollaborationService) AssignAgentTask(ctx context.Context, task map[string]interface{}) (map[string]interface{}, error) {
	// TODO: Implement task assignment logic
	taskID := fmt.Sprintf("task-%d", time.Now().Unix())

	assignment := map[string]interface{}{
		"task_id":         taskID,
		"assigned_to":     []string{"agent-1", "agent-2"},
		"assignment_type": "collaborative",
		"priority":        "high",
		"estimated_time":  "45 minutes",
		"strategy": map[string]interface{}{
			"approach":      "divide_and_conquer",
			"coordination":  "async_with_sync_points",
			"communication": "event_driven",
		},
		"created_at": time.Now().Format(time.RFC3339),
	}

	return assignment, nil
}

// ShareKnowledge shares knowledge between agents or users
func (s *CollaborationService) ShareKnowledge(ctx context.Context, knowledge SharedKnowledge) (*SharedKnowledge, error) {
	// TODO: Store shared knowledge in database
	knowledge.ID = fmt.Sprintf("knowledge-%d", time.Now().Unix())
	knowledge.SharedAt = time.Now()
	knowledge.UseCount = 0
	knowledge.EffectiveScore = 0.8

	s.logger.Info("Knowledge shared",
		slog.String("id", knowledge.ID),
		slog.String("title", knowledge.Title))

	return &knowledge, nil
}

// GetSharedKnowledge retrieves shared knowledge
func (s *CollaborationService) GetSharedKnowledge(ctx context.Context, knowledgeType, visibility string, tags []string) ([]SharedKnowledge, error) {
	s.logger.Debug("Getting shared knowledge",
		slog.String("knowledge_type", knowledgeType),
		slog.String("visibility", visibility),
		slog.Any("tags", tags))

	// Query knowledge shares from database
	// Note: The database doesn't have visibility field, so we'll simulate it
	params := sqlc.GetKnowledgeSharesParams{
		KnowledgeType: knowledgeType,
		Limit:         100,
		Offset:        0,
	}

	knowledgeShares, err := s.db.GetKnowledgeShares(ctx, params)
	if err != nil {
		s.logger.Error("Failed to get knowledge shares", slog.Any("error", err))
		return nil, fmt.Errorf("get knowledge shares: %w", err)
	}

	// If no knowledge shares in database, provide demo data
	if len(knowledgeShares) == 0 {
		s.logger.Info("No knowledge shares found in database, providing demo data")
		return s.getDemoSharedKnowledge(knowledgeType, visibility, tags), nil
	}

	// Convert to API model
	sharedKnowledge := make([]SharedKnowledge, 0, len(knowledgeShares))
	for _, share := range knowledgeShares {
		// Parse knowledge content
		var content map[string]interface{}
		if err := json.Unmarshal(share.KnowledgeContent, &content); err != nil {
			s.logger.Error("Failed to parse knowledge content",
				slog.String("knowledge_id", share.ID.String()),
				slog.Any("error", err))
			continue
		}

		// Extract data from content
		title := "Untitled Knowledge"
		if t, ok := content["title"].(string); ok {
			title = t
		}

		contentStr := ""
		if c, ok := content["content"].(string); ok {
			contentStr = c
		} else if c, ok := content["description"].(string); ok {
			contentStr = c
		}

		// Extract tags
		knowledgeTags := []string{}
		if t, ok := content["tags"].([]interface{}); ok {
			for _, tag := range t {
				if tagStr, ok := tag.(string); ok {
					knowledgeTags = append(knowledgeTags, tagStr)
				}
			}
		}

		// Filter by tags if specified
		if len(tags) > 0 && !hasAnyTag(knowledgeTags, tags) {
			continue
		}

		// Determine visibility based on metadata
		knowledgeVisibility := "team" // Default
		if v, ok := content["visibility"].(string); ok {
			knowledgeVisibility = v
		}

		// Filter by visibility if specified
		if visibility != "" && knowledgeVisibility != visibility {
			continue
		}

		// Calculate use count and effectiveness from integration results
		useCount := 0
		if share.IntegrationResult != nil && len(share.IntegrationResult) > 0 {
			var integration map[string]interface{}
			if err := json.Unmarshal(share.IntegrationResult, &integration); err == nil {
				if count, ok := integration["use_count"].(float64); ok {
					useCount = int(count)
				}
			}
		}

		// Build shared with list
		sharedWith := []string{}
		if share.TargetAgentID.Valid {
			sharedWith = append(sharedWith, share.TargetAgentName)
		}

		knowledge := SharedKnowledge{
			ID:             share.ID.String(),
			Title:          title,
			Content:        contentStr,
			Type:           share.KnowledgeType,
			SharedBy:       share.SourceAgentName,
			SharedWith:     sharedWith,
			Visibility:     knowledgeVisibility,
			Tags:           knowledgeTags,
			UseCount:       useCount,
			EffectiveScore: share.RelevanceScore.Float64,
			SharedAt:       share.CreatedAt,
			Metadata:       content,
		}

		// Set LastUsed based on integration result
		if share.IntegrationResult != nil && len(share.IntegrationResult) > 0 {
			var integration map[string]interface{}
			if err := json.Unmarshal(share.IntegrationResult, &integration); err == nil {
				if lastUsedStr, ok := integration["last_used"].(string); ok {
					if lastUsed, err := time.Parse(time.RFC3339, lastUsedStr); err == nil {
						knowledge.LastUsed = &lastUsed
					}
				}
			}
		}

		sharedKnowledge = append(sharedKnowledge, knowledge)
	}

	return sharedKnowledge, nil
}

// GetCollaborationEffectiveness analyzes collaboration effectiveness
func (s *CollaborationService) GetCollaborationEffectiveness(ctx context.Context, period string) (map[string]interface{}, error) {
	// TODO: Calculate actual effectiveness from data
	effectiveness := map[string]interface{}{
		"period": period,
		"metrics": map[string]interface{}{
			"overall_effectiveness":    0.87,
			"task_completion_rate":     0.92,
			"average_completion_time":  "42 minutes",
			"collaboration_frequency":  15,
			"knowledge_sharing_impact": 0.85,
			"agent_utilization":        0.78,
		},
		"agent_performance": []map[string]interface{}{
			{
				"agent_id":            "agent-1",
				"agent_name":          "開發專家",
				"tasks_completed":     28,
				"success_rate":        0.93,
				"avg_response_time":   "1.2 seconds",
				"collaboration_score": 0.88,
			},
			{
				"agent_id":            "agent-2",
				"agent_name":          "資料庫專家",
				"tasks_completed":     22,
				"success_rate":        0.91,
				"avg_response_time":   "1.5 seconds",
				"collaboration_score": 0.85,
			},
		},
		"collaboration_patterns": []map[string]interface{}{
			{
				"pattern":      "pair_programming",
				"frequency":    12,
				"success_rate": 0.92,
				"avg_duration": "55 minutes",
			},
			{
				"pattern":      "knowledge_transfer",
				"frequency":    8,
				"success_rate": 0.88,
				"avg_duration": "30 minutes",
			},
		},
		"improvements": []string{
			"增加架構師參與早期設計討論",
			"建立更多知識分享機制",
			"優化任務分配算法",
		},
	}

	return effectiveness, nil
}

// GetCollaborationRecommendations generates collaboration recommendations
func (s *CollaborationService) GetCollaborationRecommendations(ctx context.Context, taskType string) ([]map[string]interface{}, error) {
	// TODO: Generate actual recommendations based on data
	recommendations := []map[string]interface{}{
		{
			"id":          "rec-1",
			"type":        "agent_selection",
			"title":       "建議加入測試專家",
			"description": "根據任務複雜度，建議加入測試專家以確保代碼品質",
			"agents":      []string{"agent-4"},
			"confidence":  0.85,
			"reasoning": []string{
				"任務涉及複雜邏輯",
				"需要高測試覆蓋率",
				"測試專家當前工作量較低",
			},
		},
		{
			"id":          "rec-2",
			"type":        "workflow",
			"title":       "採用迭代開發模式",
			"description": "將任務分解為多個小迭代，每個迭代包含開發和測試",
			"confidence":  0.92,
			"benefits": []string{
				"降低風險",
				"快速獲得反饋",
				"提高靈活性",
			},
		},
		{
			"id":              "rec-3",
			"type":            "knowledge_reuse",
			"title":           "重用相關知識庫",
			"description":     "發現 3 個相關的知識條目可以加速開發",
			"knowledge_items": []string{"knowledge-1", "knowledge-2", "knowledge-3"},
			"confidence":      0.78,
			"time_saved":      "約 2 小時",
		},
	}

	return recommendations, nil
}

// Helper functions

// Helper functions

func hasAnyTag(itemTags, searchTags []string) bool {
	for _, searchTag := range searchTags {
		for _, itemTag := range itemTags {
			if itemTag == searchTag {
				return true
			}
		}
	}
	return false
}

// convertAgentDefinitionToAgent converts database model to API model
func (s *CollaborationService) convertAgentDefinitionToAgent(def *sqlc.AgentDefinition) (Agent, error) {
	// Parse capabilities
	var capabilities map[string]interface{}
	if err := json.Unmarshal(def.Capabilities, &capabilities); err != nil {
		return Agent{}, fmt.Errorf("parse capabilities: %w", err)
	}

	// Extract capability list
	capList := []string{}
	if caps, ok := capabilities["capabilities"].([]interface{}); ok {
		for _, cap := range caps {
			if capStr, ok := cap.(string); ok {
				capList = append(capList, capStr)
			}
		}
	}

	// Parse performance metrics
	var perfMetrics map[string]interface{}
	if err := json.Unmarshal(def.PerformanceMetrics, &perfMetrics); err != nil {
		perfMetrics = make(map[string]interface{})
	}

	// Extract performance score
	performanceScore := 0.85 // Default
	if score, ok := perfMetrics["score"].(float64); ok {
		performanceScore = score
	}

	// Extract workload
	currentWorkload := 0.3 // Default
	if workload, ok := perfMetrics["workload"].(float64); ok {
		currentWorkload = workload
	}

	// Determine status based on workload
	status := "available"
	if currentWorkload > 0.7 {
		status = "busy"
	} else if !def.IsActive.Bool {
		status = "offline"
	}

	// Build metadata
	metadata := make(map[string]interface{})
	if model, ok := capabilities["model"].(string); ok {
		metadata["model"] = model
	}
	if version, ok := capabilities["version"].(string); ok {
		metadata["version"] = version
	}

	return Agent{
		ID:               def.ID.String(),
		Name:             def.AgentName,
		Type:             def.AgentType,
		Capabilities:     capList,
		ExpertiseDomains: def.ExpertiseDomains,
		CurrentWorkload:  currentWorkload,
		Status:           status,
		PerformanceScore: performanceScore,
		LastActive:       def.UpdatedAt,
		Metadata:         metadata,
	}, nil
}

// pgTimestamptzToTimePtr converts pgtype.Timestamptz to *time.Time
func (s *CollaborationService) pgTimestamptzToTimePtr(ts pgtype.Timestamptz) *time.Time {
	if ts.Valid {
		return &ts.Time
	}
	return nil
}

// getDemoAgents returns demo agents for testing when database is empty
func (s *CollaborationService) getDemoAgents() []Agent {
	return []Agent{
		{
			ID:               "demo-agent-1",
			Name:             "開發專家",
			Type:             "development",
			Capabilities:     []string{"code_generation", "debugging", "refactoring", "testing"},
			ExpertiseDomains: []string{"golang", "api_design", "performance"},
			CurrentWorkload:  0.3,
			Status:           "available",
			PerformanceScore: 0.92,
			LastActive:       time.Now().Add(-5 * time.Minute),
			Metadata: map[string]interface{}{
				"model":    "claude-3-opus",
				"version":  "1.0",
				"language": "zh-TW",
			},
		},
		{
			ID:               "demo-agent-2",
			Name:             "資料庫專家",
			Type:             "database",
			Capabilities:     []string{"query_optimization", "schema_design", "migration", "indexing"},
			ExpertiseDomains: []string{"postgresql", "mongodb", "redis"},
			CurrentWorkload:  0.5,
			Status:           "available",
			PerformanceScore: 0.88,
			LastActive:       time.Now().Add(-10 * time.Minute),
		},
		{
			ID:               "demo-agent-3",
			Name:             "架構師",
			Type:             "architecture",
			Capabilities:     []string{"system_design", "pattern_recognition", "scalability", "integration"},
			ExpertiseDomains: []string{"microservices", "distributed_systems", "cloud"},
			CurrentWorkload:  0.2,
			Status:           "available",
			PerformanceScore: 0.95,
			LastActive:       time.Now().Add(-2 * time.Minute),
		},
		{
			ID:               "demo-agent-4",
			Name:             "測試專家",
			Type:             "testing",
			Capabilities:     []string{"unit_testing", "integration_testing", "e2e_testing", "test_generation"},
			ExpertiseDomains: []string{"tdd", "bdd", "performance_testing"},
			CurrentWorkload:  0.4,
			Status:           "busy",
			PerformanceScore: 0.90,
			LastActive:       time.Now().Add(-1 * time.Minute),
		},
	}
}

// getDemoSharedKnowledge returns demo shared knowledge for testing
func (s *CollaborationService) getDemoSharedKnowledge(knowledgeType, visibility string, tags []string) []SharedKnowledge {
	demoKnowledge := []SharedKnowledge{
		{
			ID:             "demo-knowledge-1",
			Title:          "Go 並發模式最佳實踐",
			Content:        "使用 context 進行取消控制，避免 goroutine 洩漏...",
			Type:           "best_practice",
			SharedBy:       "開發專家",
			SharedWith:     []string{"team"},
			Visibility:     "team",
			Tags:           []string{"golang", "concurrency", "patterns"},
			UseCount:       25,
			EffectiveScore: 0.92,
			SharedAt:       time.Now().Add(-7 * 24 * time.Hour),
		},
		{
			ID:             "demo-knowledge-2",
			Title:          "PostgreSQL 查詢優化技巧",
			Content:        "使用 EXPLAIN ANALYZE 分析查詢計劃...",
			Type:           "optimization",
			SharedBy:       "資料庫專家",
			SharedWith:     []string{"public"},
			Visibility:     "public",
			Tags:           []string{"postgresql", "performance", "database"},
			UseCount:       18,
			EffectiveScore: 0.88,
			SharedAt:       time.Now().Add(-3 * 24 * time.Hour),
		},
		{
			ID:             "demo-knowledge-3",
			Title:          "微服務架構設計原則",
			Content:        "服務邊界的定義應基於業務能力...",
			Type:           "architecture",
			SharedBy:       "架構師",
			SharedWith:     []string{"team"},
			Visibility:     "team",
			Tags:           []string{"microservices", "architecture", "design"},
			UseCount:       32,
			EffectiveScore: 0.95,
			SharedAt:       time.Now().Add(-14 * 24 * time.Hour),
		},
	}

	// Apply filters
	filtered := demoKnowledge

	if knowledgeType != "" {
		var typeFiltered []SharedKnowledge
		for _, k := range filtered {
			if k.Type == knowledgeType {
				typeFiltered = append(typeFiltered, k)
			}
		}
		filtered = typeFiltered
	}

	if visibility != "" {
		var visFiltered []SharedKnowledge
		for _, k := range filtered {
			if k.Visibility == visibility {
				visFiltered = append(visFiltered, k)
			}
		}
		filtered = visFiltered
	}

	if len(tags) > 0 {
		var tagFiltered []SharedKnowledge
		for _, k := range filtered {
			if hasAnyTag(k.Tags, tags) {
				tagFiltered = append(tagFiltered, k)
			}
		}
		filtered = tagFiltered
	}

	return filtered
}
