package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/server/pghelpers"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
)

// MemoryService 提供記憶系統的業務邏輯
type MemoryService struct {
	queries *sqlc.Queries
	logger  *slog.Logger
	metrics *observability.Metrics
}

// NewMemoryService 建立新的記憶服務
func NewMemoryService(queries *sqlc.Queries, logger *slog.Logger, metrics *observability.Metrics) *MemoryService {
	return &MemoryService{
		queries: queries,
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

	// 取得記憶條目
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	entries, err := s.queries.GetMemoryEntriesByUser(ctx, pghelpers.UUIDToPgtypeUUID(userUUID))
	if err != nil {
		return nil, fmt.Errorf("failed to get memory entries: %w", err)
	}

	nodes := make([]MemoryNode, 0, len(entries))
	for _, entry := range entries {
		// 解析 metadata JSON
		var metadata map[string]interface{}
		if entry.Metadata != nil {
			if err := json.Unmarshal(entry.Metadata, &metadata); err != nil {
				s.logger.Warn("Failed to parse metadata", slog.String("entry_id", entry.ID.String()))
				metadata = make(map[string]interface{})
			}
		} else {
			metadata = make(map[string]interface{})
		}

		node := MemoryNode{
			ID:          pghelpers.PgtypeUUIDToUUID(entry.ID).String(),
			UserID:      pghelpers.PgtypeUUIDToUUID(entry.UserID).String(),
			Type:        entry.MemoryType,
			Content:     entry.Content,
			Importance:  pghelpers.PgtypeNumericToFloat64(entry.Importance),
			AccessCount: pghelpers.PgtypeInt4ToInt32(entry.AccessCount),
			LastAccess:  pghelpers.PgtypeTimestamptzToTimePtr(entry.LastAccess),
			CreatedAt:   entry.CreatedAt,
			ExpiresAt:   pghelpers.PgtypeTimestamptzToTimePtr(entry.ExpiresAt),
			Metadata:    metadata,
		}

		// 應用過濾器
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

	// TODO: 實現記憶關聯查詢（需要擴展資料庫 schema）
	// 目前返回基本圖譜結構
	edges := []MemoryConnection{}

	// 計算統計資訊
	stats := s.calculateMemoryStats(nodes)

	graph := &MemoryGraph{
		Nodes: nodes,
		Edges: edges,
		Stats: stats,
	}

	s.logger.Debug("Generated memory graph",
		slog.String("user_id", userID),
		slog.Int("nodes", len(nodes)),
		slog.Int("edges", len(edges)))

	return graph, nil
}

// UpdateMemoryNode 更新記憶節點
func (s *MemoryService) UpdateMemoryNode(ctx context.Context, nodeID string, updates MemoryNodeUpdate) (*MemoryNode, error) {
	s.logger.Debug("Updating memory node", slog.String("node_id", nodeID))

	// 驗證節點ID
	nodeUUID, err := uuid.Parse(nodeID)
	if err != nil {
		return nil, fmt.Errorf("invalid node ID: %w", err)
	}

	// 取得現有節點
	entry, err := s.queries.GetMemoryEntry(ctx, pghelpers.UUIDToPgtypeUUID(nodeUUID))
	if err != nil {
		return nil, fmt.Errorf("failed to get memory entry: %w", err)
	}

	// 準備更新的字段
	var needsUpdate bool
	content := entry.Content
	importance := entry.Importance
	metadataBytes := entry.Metadata

	// 更新內容
	if updates.Content != nil {
		content = *updates.Content
		needsUpdate = true
	}

	// 更新重要性
	if updates.Importance != nil {
		importance = pghelpers.Float64ToPgtypeNumeric(*updates.Importance)
		needsUpdate = true
	}

	// 更新元數據
	if updates.Metadata != nil {
		metadataJSON, err := json.Marshal(*updates.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataBytes = metadataJSON
		needsUpdate = true
	}

	// 如果有更新，執行更新操作
	if needsUpdate {
		_, err := s.queries.UpdateMemoryEntry(ctx, sqlc.UpdateMemoryEntryParams{
			ID:          pghelpers.UUIDToPgtypeUUID(nodeUUID),
			Content:     content,
			Importance:  importance,
			AccessCount: entry.AccessCount,
			LastAccess:  entry.LastAccess,
			ExpiresAt:   entry.ExpiresAt,
			Metadata:    metadataBytes,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update memory entry: %w", err)
		}

		// 重新取得更新後的資料
		entry, err = s.queries.GetMemoryEntry(ctx, pghelpers.UUIDToPgtypeUUID(nodeUUID))
		if err != nil {
			return nil, fmt.Errorf("failed to get updated memory entry: %w", err)
		}
	}

	// 增加存取計數
	newAccessCount := pghelpers.PgtypeInt4ToInt32(entry.AccessCount) + 1
	if err := s.queries.IncrementMemoryAccess(ctx, sqlc.IncrementMemoryAccessParams{
		ID:          pghelpers.UUIDToPgtypeUUID(nodeUUID),
		AccessCount: pghelpers.Int32ToPgtypeInt4(newAccessCount),
		LastAccess:  pghelpers.TimeToPgtypeTimestamptz(time.Now()),
	}); err != nil {
		s.logger.Warn("Failed to increment access count", slog.Any("error", err))
	}

	// 解析 metadata
	var parsedMetadata map[string]interface{}
	if entry.Metadata != nil {
		if err := json.Unmarshal(entry.Metadata, &parsedMetadata); err != nil {
			parsedMetadata = make(map[string]interface{})
		}
	} else {
		parsedMetadata = make(map[string]interface{})
	}

	// 返回更新後的節點
	node := &MemoryNode{
		ID:          pghelpers.PgtypeUUIDToUUID(entry.ID).String(),
		UserID:      pghelpers.PgtypeUUIDToUUID(entry.UserID).String(),
		Type:        entry.MemoryType,
		Content:     entry.Content,
		Importance:  pghelpers.PgtypeNumericToFloat64(entry.Importance),
		AccessCount: newAccessCount,
		LastAccess:  pghelpers.PgtypeTimestamptzToTimePtr(entry.LastAccess),
		CreatedAt:   entry.CreatedAt,
		ExpiresAt:   pghelpers.PgtypeTimestamptzToTimePtr(entry.ExpiresAt),
		Metadata:    parsedMetadata,
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
