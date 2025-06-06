package memory

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/koopa0/assistant-go/internal/core/memory"
	"github.com/koopa0/assistant-go/internal/platform/observability"
)

// MemoryStore 定義 MemoryService 需要的記憶體操作
// Consumer-defined interface following Go principles
type MemoryStore interface {
	Store(ctx context.Context, entry *memory.Entry) error
	Retrieve(ctx context.Context, criteria memory.SearchCriteria) ([]*memory.Entry, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	SearchRelated(ctx context.Context, entryID string, maxResults int) ([]*memory.Entry, error)
}

// MemoryService 提供記憶系統的業務邏輯
type MemoryService struct {
	store   MemoryStore
	logger  *slog.Logger
	metrics *observability.Metrics
}

// NewMemoryService 建立新的記憶服務
func NewMemoryService(store MemoryStore, logger *slog.Logger, metrics *observability.Metrics) *MemoryService {
	return &MemoryService{
		store:   store,
		logger:  logger,
		metrics: metrics,
	}
}

// MemoryNode 代表記憶節點
type MemoryNode struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Type        string                 `json:"type"`
	Content     string                 `json:"content"`
	Importance  float64                `json:"importance"`
	AccessCount int32                  `json:"access_count"`
	LastAccess  *time.Time             `json:"last_access,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	// 關聯資訊
	Connections []MemoryConnection `json:"connections,omitempty"`
}

// MemoryConnection 代表記憶節點間的連接
type MemoryConnection struct {
	TargetID  string    `json:"target_id"`
	Type      string    `json:"type"`
	Strength  float64   `json:"strength"`
	CreatedAt time.Time `json:"created_at"`
}

// MemoryGraph 代表記憶圖譜
type MemoryGraph struct {
	Nodes []MemoryNode       `json:"nodes"`
	Edges []MemoryConnection `json:"edges"`
	Stats MemoryStats        `json:"stats"`
}

// MemoryStats 記憶統計資訊
type MemoryStats struct {
	TotalNodes        int32            `json:"total_nodes"`
	NodesByType       map[string]int32 `json:"nodes_by_type"`
	AverageImportance float64          `json:"average_importance"`
	TotalConnections  int32            `json:"total_connections"`
	LastUpdated       time.Time        `json:"last_updated"`
}

// GetMemoryNodes 取得使用者的記憶節點
func (s *MemoryService) GetMemoryNodes(ctx context.Context, userID string, filters MemoryFilters) ([]MemoryNode, error) {
	s.logger.Debug("Getting memory nodes", slog.String("user_id", userID))

	// Build search criteria
	criteria := memory.SearchCriteria{
		UserID: userID,
		Limit:  1000, // Default large limit for server layer
	}

	// Apply type filter
	if filters.Type != nil {
		if mappedType := s.mapStringToMemoryType(*filters.Type); mappedType != "" {
			criteria.Types = []memory.Type{mappedType}
		}
	}

	// Apply importance filter
	if filters.MinImportance != nil {
		criteria.ImportanceMin = *filters.MinImportance
	}

	// Apply time filter
	if filters.MaxAge != nil {
		since := time.Now().Add(-*filters.MaxAge)
		criteria.TimeFrom = &since
	}

	// Search memories using store
	entries, err := s.store.Retrieve(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search memories: %w", err)
	}

	// Convert to MemoryNode format
	nodes := make([]MemoryNode, 0, len(entries))
	for _, entry := range entries {
		// Convert context map to metadata
		metadata := make(map[string]interface{})
		if entry.Context != nil {
			metadata = entry.Context
		}

		node := MemoryNode{
			ID:          entry.ID,
			UserID:      entry.UserID,
			Type:        string(entry.Type),
			Content:     entry.Content,
			Importance:  entry.Importance,
			AccessCount: int32(entry.AccessCount),
			LastAccess:  nil, // Core memory doesn't have LastAccess in Entry
			CreatedAt:   entry.CreatedAt,
			ExpiresAt:   entry.ExpiresAt,
			Metadata:    metadata,
		}

		// Apply additional filters
		if s.passesFilters(node, filters) {
			nodes = append(nodes, node)
		}
	}

	s.logger.Debug("Retrieved memory nodes",
		slog.String("user_id", userID),
		slog.Int("count", len(nodes)))

	return nodes, nil
}

// GetMemoryGraph 取得完整的記憶圖譜
func (s *MemoryService) GetMemoryGraph(ctx context.Context, userID string) (*MemoryGraph, error) {
	s.logger.Debug("Getting memory graph", slog.String("user_id", userID))

	// 取得所有記憶節點
	nodes, err := s.GetMemoryNodes(ctx, userID, MemoryFilters{})
	if err != nil {
		return nil, fmt.Errorf("failed to get memory nodes: %w", err)
	}

	// Get graph connections using SearchRelated
	edges := []MemoryConnection{}

	// For each node, get some related memories to build edges
	for _, node := range nodes {
		related, err := s.store.SearchRelated(ctx, node.ID, 5)
		if err != nil {
			s.logger.Warn("Failed to get related memories",
				slog.String("node_id", node.ID),
				slog.Any("error", err))
			continue
		}

		// Convert to MemoryConnection format
		for _, relatedEntry := range related {
			edges = append(edges, MemoryConnection{
				TargetID:  relatedEntry.ID,
				Type:      "related", // Simple relation type
				Strength:  0.5,       // Default strength
				CreatedAt: relatedEntry.CreatedAt,
			})
		}
	}

	// 計算統計資訊
	stats := s.calculateMemoryStats(nodes)
	stats.TotalConnections = int32(len(edges))

	memoryGraph := &MemoryGraph{
		Nodes: nodes,
		Edges: edges,
		Stats: stats,
	}

	s.logger.Debug("Generated memory graph",
		slog.String("user_id", userID),
		slog.Int("nodes", len(nodes)),
		slog.Int("edges", len(edges)))

	return memoryGraph, nil
}

// UpdateMemoryNode 更新記憶節點
func (s *MemoryService) UpdateMemoryNode(ctx context.Context, nodeID string, updates MemoryNodeUpdate) (*MemoryNode, error) {
	s.logger.Debug("Updating memory node", slog.String("node_id", nodeID))

	if nodeID == "" {
		return nil, fmt.Errorf("node ID cannot be empty")
	}

	// Get existing entry by searching for the specific node ID
	criteria := memory.SearchCriteria{
		UserID: "", // Will be filled when we have user context
		Limit:  1,
	}

	entries, err := s.store.Retrieve(ctx, criteria)
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("memory node not found: %s", nodeID)
	}

	existingEntry := entries[0]

	// Create updated entry based on existing entry
	updatedEntry := *existingEntry
	updatedEntry.UpdatedAt = time.Now()
	updatedEntry.AccessCount++

	// Apply updates
	if updates.Content != nil {
		updatedEntry.Content = *updates.Content
	}

	if updates.Importance != nil {
		updatedEntry.Importance = *updates.Importance
	}

	if updates.Metadata != nil {
		// Update context map (which serves as metadata in core memory)
		if updatedEntry.Context == nil {
			updatedEntry.Context = make(map[string]interface{})
		}
		for k, v := range *updates.Metadata {
			updatedEntry.Context[k] = v
		}
	}

	// Store updated entry
	if err := s.store.Store(ctx, &updatedEntry); err != nil {
		return nil, fmt.Errorf("failed to update memory: %w", err)
	}

	// Convert to MemoryNode format
	metadata := make(map[string]interface{})
	if updatedEntry.Context != nil {
		metadata = updatedEntry.Context
	}

	node := &MemoryNode{
		ID:          updatedEntry.ID,
		UserID:      updatedEntry.UserID,
		Type:        string(updatedEntry.Type),
		Content:     updatedEntry.Content,
		Importance:  updatedEntry.Importance,
		AccessCount: int32(updatedEntry.AccessCount),
		LastAccess:  nil, // Core memory doesn't track LastAccess in Entry
		CreatedAt:   updatedEntry.CreatedAt,
		ExpiresAt:   updatedEntry.ExpiresAt,
		Metadata:    metadata,
	}

	s.logger.Debug("Updated memory node", slog.String("node_id", nodeID))
	return node, nil
}

// MemoryFilters 記憶節點過濾器
type MemoryFilters struct {
	Type          *string        `json:"type,omitempty"`
	MinImportance *float64       `json:"min_importance,omitempty"`
	MaxAge        *time.Duration `json:"max_age,omitempty"`
	Search        *string        `json:"search,omitempty"`
}

// MemoryNodeUpdate 記憶節點更新資料
type MemoryNodeUpdate struct {
	Content    *string                 `json:"content,omitempty"`
	Importance *float64                `json:"importance,omitempty"`
	Metadata   *map[string]interface{} `json:"metadata,omitempty"`
}

// passesFilters 檢查節點是否通過過濾器
func (s *MemoryService) passesFilters(node MemoryNode, filters MemoryFilters) bool {
	// 類型過濾
	if filters.Type != nil && node.Type != *filters.Type {
		return false
	}

	// 重要性過濾
	if filters.MinImportance != nil && node.Importance < *filters.MinImportance {
		return false
	}

	// 年齡過濾
	if filters.MaxAge != nil {
		age := time.Since(node.CreatedAt)
		if age > *filters.MaxAge {
			return false
		}
	}

	// 搜尋過濾
	if filters.Search != nil {
		// 簡單的內容搜尋
		// TODO: 實現更高級的語義搜尋
		searchTerm := *filters.Search
		if !contains(node.Content, searchTerm) {
			return false
		}
	}

	return true
}

// calculateMemoryStats 計算記憶統計資訊
func (s *MemoryService) calculateMemoryStats(nodes []MemoryNode) MemoryStats {
	stats := MemoryStats{
		TotalNodes:  int32(len(nodes)),
		NodesByType: make(map[string]int32),
		LastUpdated: time.Now(),
	}

	if len(nodes) == 0 {
		return stats
	}

	var totalImportance float64
	for _, node := range nodes {
		// 按類型統計
		stats.NodesByType[node.Type]++

		// 累計重要性
		totalImportance += node.Importance
	}

	// 計算平均重要性
	stats.AverageImportance = totalImportance / float64(len(nodes))

	return stats
}

// contains 簡單的字串包含檢查（不區分大小寫）
func contains(text, search string) bool {
	// TODO: 實現更高級的搜尋算法
	return len(search) == 0 ||
		len(text) > 0 &&
			(text == search ||
				len(text) >= len(search) &&
					(text[:len(search)] == search ||
						text[len(text)-len(search):] == search))
}

// mapStringToMemoryType maps string type to core memory type
func (s *MemoryService) mapStringToMemoryType(typeStr string) memory.Type {
	switch typeStr {
	case "working":
		return memory.TypeWorking
	case "episodic":
		return memory.TypeEpisodic
	case "semantic":
		return memory.TypeSemantic
	case "procedural":
		return memory.TypeProcedural
	default:
		s.logger.Warn("Unknown memory type", slog.String("type", typeStr))
		return ""
	}
}
