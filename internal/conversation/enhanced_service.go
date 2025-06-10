package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/ai/token"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server/middleware"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// EnhancedConversationService 提供增強的對話管理功能
type EnhancedConversationService struct {
	queries      sqlc.Querier
	logger       *slog.Logger
	metrics      *observability.Metrics
	tokenCounter token.Counter
}

// NewEnhancedConversationService 建立新的增強對話服務
func NewEnhancedConversationService(queries sqlc.Querier, logger *slog.Logger, metrics *observability.Metrics) *EnhancedConversationService {
	return &EnhancedConversationService{
		queries:      queries,
		logger:       logger,
		metrics:      metrics,
		tokenCounter: token.NewTokenCounter("claude"), // 預設使用 Claude 模型的 token 計算
	}
}

// ConversationSummary 對話摘要
type ConversationSummary struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"user_id"`
	Title        string                 `json:"title"`
	Summary      *string                `json:"summary,omitempty"`
	MessageCount int32                  `json:"message_count"`
	LastMessage  *MessageSummary        `json:"last_message,omitempty"`
	IsArchived   bool                   `json:"is_archived"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ConversationDetail 對話詳情
type ConversationDetail struct {
	ID         string                 `json:"id"`
	UserID     string                 `json:"user_id"`
	Title      string                 `json:"title"`
	Summary    *string                `json:"summary,omitempty"`
	Messages   []MessageDetail        `json:"messages"`
	IsArchived bool                   `json:"is_archived"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// MessageSummary 訊息摘要
type MessageSummary struct {
	ID         string    `json:"id"`
	Role       string    `json:"role"`
	Preview    string    `json:"preview"` // 前 100 個字符
	TokenCount *int32    `json:"token_count,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// MessageDetail 訊息詳情
type MessageDetail struct {
	ID         string                 `json:"id"`
	Role       string                 `json:"role"`
	Content    string                 `json:"content"`
	TokenCount *int32                 `json:"token_count,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	Executions []ToolExecutionSummary `json:"tool_executions,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ToolExecutionSummary 工具執行摘要
type ToolExecutionSummary struct {
	ID              string     `json:"id"`
	ToolName        string     `json:"tool_name"`
	Status          string     `json:"status"`
	ExecutionTimeMs *int32     `json:"execution_time_ms,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

// CreateConversationRequest 建立對話請求
type CreateConversationRequest struct {
	Title    string                 `json:"title"`
	Summary  string                 `json:"summary,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SendMessageRequest 發送訊息請求
type SendMessageRequest struct {
	Role     string                 `json:"role"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SendMessageResponse 發送訊息回應
type SendMessageResponse struct {
	Message      MessageDetail       `json:"message"`
	AIResponse   *MessageDetail      `json:"ai_response,omitempty"`
	Conversation ConversationSummary `json:"conversation"`
}

// GetConversations 取得對話列表
func (s *EnhancedConversationService) GetConversations(ctx context.Context, userID string, archived *bool, limit, offset int32) ([]ConversationSummary, int32, error) {
	s.logger.Debug("Getting conversations",
		slog.String("user_id", userID),
		slog.Any("archived", archived),
		slog.Int("limit", int(limit)),
		slog.Int("offset", int(offset)))

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid user ID: %w", err)
	}

	// 取得對話列表
	conversations, err := s.queries.GetConversationsByUser(ctx, middleware.UUIDToPgtypeUUID(userUUID))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get conversations: %w", err)
	}

	// 轉換為摘要格式
	summaries := make([]ConversationSummary, 0, len(conversations))
	for _, conv := range conversations {
		// 過濾歸檔狀態
		if archived != nil && conv.IsArchived.Bool != *archived {
			continue
		}

		// 解析 metadata
		var metadata map[string]interface{}
		if conv.Metadata != nil {
			if err := json.Unmarshal(conv.Metadata, &metadata); err != nil {
				s.logger.Warn("Failed to parse conversation metadata",
					slog.String("conv_id", conv.ID.String()))
				metadata = make(map[string]interface{})
			}
		} else {
			metadata = make(map[string]interface{})
		}

		// 取得訊息數量和最後訊息
		messageCount, lastMessage, err := s.getConversationMessageInfo(ctx, conv.ID)
		if err != nil {
			s.logger.Warn("Failed to get message info",
				slog.String("conv_id", middleware.PgtypeUUIDToUUID(conv.ID).String()),
				slog.Any("error", err))
		}

		summary := ConversationSummary{
			ID:           middleware.PgtypeUUIDToUUID(conv.ID).String(),
			UserID:       middleware.PgtypeUUIDToUUID(conv.UserID).String(),
			Title:        conv.Title,
			Summary:      middleware.PgtypeTextToStringPtr(conv.Summary),
			MessageCount: messageCount,
			LastMessage:  lastMessage,
			IsArchived:   conv.IsArchived.Bool,
			CreatedAt:    conv.CreatedAt,
			UpdatedAt:    conv.UpdatedAt,
			Metadata:     metadata,
		}

		summaries = append(summaries, summary)
	}

	// 應用分頁
	total := int32(len(summaries))
	if offset >= total {
		return []ConversationSummary{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return summaries[offset:end], total, nil
}

// CreateConversation 建立新對話
func (s *EnhancedConversationService) CreateConversation(ctx context.Context, userID string, req CreateConversationRequest) (*ConversationDetail, error) {
	s.logger.Debug("Creating conversation",
		slog.String("user_id", userID),
		slog.String("title", req.Title))

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// 準備 metadata
	var metadataJSON []byte
	if req.Metadata != nil {
		metadataJSON, err = json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// 建立對話
	conv, err := s.queries.CreateConversation(ctx, sqlc.CreateConversationParams{
		UserID:   middleware.UUIDToPgtypeUUID(userUUID),
		Title:    req.Title,
		Summary:  middleware.StringPtrToPgtypeText(&req.Summary),
		Metadata: metadataJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// 轉換為詳情格式
	detail := &ConversationDetail{
		ID:         middleware.PgtypeUUIDToUUID(conv.ID).String(),
		UserID:     middleware.PgtypeUUIDToUUID(conv.UserID).String(),
		Title:      conv.Title,
		Summary:    middleware.PgtypeTextToStringPtr(conv.Summary),
		Messages:   []MessageDetail{}, // 新對話沒有訊息
		IsArchived: conv.IsArchived.Bool,
		CreatedAt:  conv.CreatedAt,
		UpdatedAt:  conv.UpdatedAt,
		Metadata:   req.Metadata,
	}

	s.logger.Debug("Created conversation",
		slog.String("conversation_id", detail.ID))

	return detail, nil
}

// GetConversation 取得對話詳情
func (s *EnhancedConversationService) GetConversation(ctx context.Context, conversationID string) (*ConversationDetail, error) {
	s.logger.Debug("Getting conversation", slog.String("conversation_id", conversationID))

	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	// 取得對話
	conv, err := s.queries.GetConversation(ctx, middleware.UUIDToPgtypeUUID(convUUID))
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// 取得訊息
	messages, err := s.queries.GetMessagesByConversation(ctx, sqlc.GetMessagesByConversationParams{
		ConversationID: middleware.UUIDToPgtypeUUID(convUUID),
		Limit:          1000, // 預設限制
		Offset:         0,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	// 轉換訊息格式
	messageDetails := make([]MessageDetail, 0, len(messages))
	for _, msg := range messages {
		// 解析 metadata
		var metadata map[string]interface{}
		if msg.Metadata != nil {
			if err := json.Unmarshal(msg.Metadata, &metadata); err != nil {
				s.logger.Warn("Failed to parse message metadata",
					slog.String("msg_id", msg.ID.String()))
				metadata = make(map[string]interface{})
			}
		} else {
			metadata = make(map[string]interface{})
		}

		// 取得工具執行記錄
		executions, err := s.getToolExecutions(ctx, msg.ID)
		if err != nil {
			s.logger.Warn("Failed to get tool executions",
				slog.String("msg_id", middleware.PgtypeUUIDToUUID(msg.ID).String()),
				slog.Any("error", err))
		}

		messageDetail := MessageDetail{
			ID:         middleware.PgtypeUUIDToUUID(msg.ID).String(),
			Role:       msg.Role,
			Content:    msg.Content,
			TokenCount: middleware.PgtypeInt4ToInt32Ptr(msg.TokenCount),
			Metadata:   metadata,
			Executions: executions,
			CreatedAt:  msg.CreatedAt,
		}

		messageDetails = append(messageDetails, messageDetail)
	}

	// 解析對話 metadata
	var convMetadata map[string]interface{}
	if conv.Metadata != nil {
		if err := json.Unmarshal(conv.Metadata, &convMetadata); err != nil {
			s.logger.Warn("Failed to parse conversation metadata",
				slog.String("conv_id", conv.ID.String()))
			convMetadata = make(map[string]interface{})
		}
	} else {
		convMetadata = make(map[string]interface{})
	}

	detail := &ConversationDetail{
		ID:         middleware.PgtypeUUIDToUUID(conv.ID).String(),
		UserID:     middleware.PgtypeUUIDToUUID(conv.UserID).String(),
		Title:      conv.Title,
		Summary:    middleware.PgtypeTextToStringPtr(conv.Summary),
		Messages:   messageDetails,
		IsArchived: conv.IsArchived.Bool,
		CreatedAt:  conv.CreatedAt,
		UpdatedAt:  conv.UpdatedAt,
		Metadata:   convMetadata,
	}

	s.logger.Debug("Retrieved conversation",
		slog.String("conversation_id", conversationID),
		slog.Int("message_count", len(messageDetails)))

	return detail, nil
}

// SendMessage 發送訊息到對話
func (s *EnhancedConversationService) SendMessage(ctx context.Context, conversationID string, req SendMessageRequest) (*SendMessageResponse, error) {
	s.logger.Debug("Sending message",
		slog.String("conversation_id", conversationID),
		slog.String("role", req.Role))

	convUUID, err := uuid.Parse(conversationID)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	// 驗證對話存在
	conv, err := s.queries.GetConversation(ctx, middleware.UUIDToPgtypeUUID(convUUID))
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	// 準備 metadata
	var metadataJSON []byte
	if req.Metadata != nil {
		metadataJSON, err = json.Marshal(req.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// 計算 token 數量
	tokenCount := s.tokenCounter.EstimateMessageTokens(req.Content, req.Metadata)

	// 建立預覽文本（最多 50 個字符）
	preview := req.Content
	if len(preview) > 50 {
		preview = preview[:50] + "..."
	}

	s.logger.Debug("Calculated token count",
		slog.String("content_preview", preview),
		slog.Int("token_count", int(tokenCount)))

	// 建立訊息
	msg, err := s.queries.CreateMessage(ctx, sqlc.CreateMessageParams{
		ConversationID: middleware.UUIDToPgtypeUUID(convUUID),
		Role:           req.Role,
		Content:        req.Content,
		Metadata:       metadataJSON,
		TokenCount:     middleware.Int32ToPgtypeInt4(tokenCount),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// 轉換訊息格式
	messageDetail := MessageDetail{
		ID:         middleware.PgtypeUUIDToUUID(msg.ID).String(),
		Role:       msg.Role,
		Content:    msg.Content,
		TokenCount: middleware.PgtypeInt4ToInt32Ptr(msg.TokenCount),
		Metadata:   req.Metadata,
		Executions: []ToolExecutionSummary{},
		CreatedAt:  msg.CreatedAt,
	}

	// 建立對話摘要
	messageCount, lastMessage, _ := s.getConversationMessageInfo(ctx, middleware.UUIDToPgtypeUUID(convUUID))

	var convMetadata map[string]interface{}
	if conv.Metadata != nil {
		json.Unmarshal(conv.Metadata, &convMetadata)
	} else {
		convMetadata = make(map[string]interface{})
	}

	conversationSummary := ConversationSummary{
		ID:           middleware.PgtypeUUIDToUUID(conv.ID).String(),
		UserID:       middleware.PgtypeUUIDToUUID(conv.UserID).String(),
		Title:        conv.Title,
		Summary:      middleware.PgtypeTextToStringPtr(conv.Summary),
		MessageCount: messageCount,
		LastMessage:  lastMessage,
		IsArchived:   conv.IsArchived.Bool,
		CreatedAt:    conv.CreatedAt,
		UpdatedAt:    conv.UpdatedAt,
		Metadata:     convMetadata,
	}

	response := &SendMessageResponse{
		Message:      messageDetail,
		AIResponse:   nil, // TODO: 如果需要 AI 回應，這裡處理
		Conversation: conversationSummary,
	}

	s.logger.Debug("Message sent",
		slog.String("message_id", messageDetail.ID))

	return response, nil
}

// 輔助方法

// getConversationMessageInfo 取得對話的訊息資訊
func (s *EnhancedConversationService) getConversationMessageInfo(ctx context.Context, conversationID pgtype.UUID) (int32, *MessageSummary, error) {
	messages, err := s.queries.GetMessagesByConversation(ctx, sqlc.GetMessagesByConversationParams{
		ConversationID: conversationID,
		Limit:          1000, // 預設限制
		Offset:         0,
	})
	if err != nil {
		return 0, nil, err
	}

	count := int32(len(messages))
	if count == 0 {
		return 0, nil, nil
	}

	// 最後一則訊息
	lastMsg := messages[len(messages)-1]
	preview := lastMsg.Content
	if len(preview) > 100 {
		preview = preview[:100] + "..."
	}

	lastMessage := &MessageSummary{
		ID:         middleware.PgtypeUUIDToUUID(lastMsg.ID).String(),
		Role:       lastMsg.Role,
		Preview:    preview,
		TokenCount: middleware.PgtypeInt4ToInt32Ptr(lastMsg.TokenCount),
		CreatedAt:  lastMsg.CreatedAt,
	}

	return count, lastMessage, nil
}

// getToolExecutions 取得訊息的工具執行記錄
func (s *EnhancedConversationService) getToolExecutions(ctx context.Context, messageID pgtype.UUID) ([]ToolExecutionSummary, error) {
	executions, err := s.queries.GetToolExecutionsByMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	summaries := make([]ToolExecutionSummary, 0, len(executions))
	for _, exec := range executions {
		summary := ToolExecutionSummary{
			ID:              middleware.PgtypeUUIDToUUID(exec.ID).String(),
			ToolName:        exec.ToolName,
			Status:          exec.Status,
			ExecutionTimeMs: middleware.PgtypeInt4ToInt32Ptr(exec.ExecutionTimeMs),
			StartedAt:       middleware.PgtypeTimestamptzToTime(exec.StartedAt),
			CompletedAt:     middleware.PgtypeTimestamptzToTimePtr(exec.CompletedAt),
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// SetTokenCountingModel 設定 token 計算模型
func (s *EnhancedConversationService) SetTokenCountingModel(model string) {
	s.tokenCounter.SetTokenCountingModel(model)
	s.logger.Debug("Updated token counting model", slog.String("model", model))
}

// GetTokenCountingModel 取得當前 token 計算模型
func (s *EnhancedConversationService) GetTokenCountingModel() string {
	return s.tokenCounter.GetTokenCountingModel()
}

// GetTokenCounter 取得 token 計算器（用於外部整合）
func (s *EnhancedConversationService) GetTokenCounter() token.Counter {
	return s.tokenCounter
}
