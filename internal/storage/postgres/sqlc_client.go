package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/koopa0/assistant-go/internal/storage/postgres/sqlc"
	"github.com/pgvector/pgvector-go"
)

// SQLCClient wraps the sqlc-generated queries with our domain types
type SQLCClient struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *slog.Logger
}

// NewSQLCClient creates a new SQLC-based database client
func NewSQLCClient(pool *pgxpool.Pool, logger *slog.Logger) *SQLCClient {
	return &SQLCClient{
		pool:    pool,
		queries: sqlc.New(pool),
		logger:  logger,
	}
}

// Health checks the database connection
func (c *SQLCClient) Health(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// Close closes the database connection pool
func (c *SQLCClient) Close() error {
	c.pool.Close()
	return nil
}

// GetQueries returns the underlying sqlc.Queries for direct access
func (c *SQLCClient) GetQueries() *sqlc.Queries {
	return c.queries
}

// Conversation domain methods

// CreateConversation creates a new conversation
func (c *SQLCClient) CreateConversation(ctx context.Context, userID, title string, metadata map[string]interface{}) (*Conversation, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	userUUID, err := c.parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	params := sqlc.CreateConversationParams{
		UserID:   userUUID,
		Title:    title,
		Metadata: metadataJSON,
	}

	sqlcConv, err := c.queries.CreateConversation(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return c.convertSQLCConversationRow(sqlcConv), nil
}

// GetConversation retrieves a conversation by ID
func (c *SQLCClient) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	uuid, err := c.parseUUID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	sqlcConv, err := c.queries.GetConversation(ctx, uuid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("conversation not found")
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	return c.convertSQLCConversationFromGetRow(sqlcConv), nil
}

// GetConversationsByUser retrieves conversations for a user with pagination
func (c *SQLCClient) GetConversationsByUser(ctx context.Context, userID string, limit, offset int) ([]*Conversation, error) {
	userUUID, err := c.parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// GetConversationsByUser 現在只需要 userID 參數
	sqlcConvs, err := c.queries.GetConversationsByUser(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversations: %w", err)
	}

	conversations := make([]*Conversation, len(sqlcConvs))
	for i, sqlcConv := range sqlcConvs {
		conversations[i] = c.convertSQLCConversationFromGetByUserRow(sqlcConv)
	}

	return conversations, nil
}

// Message domain methods

// CreateMessage creates a new message
func (c *SQLCClient) CreateMessage(ctx context.Context, conversationID, role, content string, tokenCount int, metadata map[string]interface{}) (*Message, error) {
	convUUID, err := c.parseUUID(conversationID)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	params := sqlc.CreateMessageParams{
		ConversationID: convUUID,
		Role:           role,
		Content:        content,
		TokenCount:     pgtype.Int4{Int32: int32(tokenCount), Valid: true},
		Metadata:       metadataJSON,
	}

	sqlcMsg, err := c.queries.CreateMessage(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return c.convertSQLCMessageFromCreateRow(sqlcMsg), nil
}

// GetMessagesByConversation retrieves messages for a conversation
func (c *SQLCClient) GetMessagesByConversation(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	convUUID, err := c.parseUUID(conversationID)
	if err != nil {
		return nil, fmt.Errorf("invalid conversation ID: %w", err)
	}

	if limit <= 0 {
		// Get all messages
		sqlcMsgs, err := c.queries.GetAllMessagesByConversation(ctx, convUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to get messages: %w", err)
		}

		messages := make([]*Message, len(sqlcMsgs))
		for i, sqlcMsg := range sqlcMsgs {
			messages[i] = c.convertSQLCMessageFromGetAllRow(sqlcMsg)
		}
		return messages, nil
	} else {
		// Get paginated messages
		params := sqlc.GetMessagesByConversationParams{
			ConversationID: convUUID,
			Limit:          int32(limit),
			Offset:         int32(offset),
		}
		sqlcMsgs, err := c.queries.GetMessagesByConversation(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to get messages: %w", err)
		}

		messages := make([]*Message, len(sqlcMsgs))
		for i, sqlcMsg := range sqlcMsgs {
			messages[i] = c.convertSQLCMessageFromGetRow(sqlcMsg)
		}
		return messages, nil
	}
}

// Embedding domain methods

// CreateEmbedding creates or updates an embedding
func (c *SQLCClient) CreateEmbedding(ctx context.Context, contentType, contentID, contentText string, embedding []float64, metadata map[string]interface{}) (*EmbeddingRecord, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	embeddingVector := c.vectorToPgVector(embedding)

	contentUUID, err := c.parseUUID(contentID)
	if err != nil {
		return nil, fmt.Errorf("invalid content ID: %w", err)
	}

	params := sqlc.CreateEmbeddingParams{
		ContentType: contentType,
		ContentID:   contentUUID,
		ContentText: contentText,
		Embedding:   embeddingVector,
		Metadata:    metadataJSON,
	}

	sqlcEmb, err := c.queries.CreateEmbedding(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	return c.convertSQLCEmbedding(sqlcEmb), nil
}

// SearchSimilarEmbeddings searches for similar embeddings
// Note: This is a simplified implementation. For production, use proper vector similarity search.
func (c *SQLCClient) SearchSimilarEmbeddings(ctx context.Context, queryEmbedding []float64, contentType string, limit int, threshold float64) ([]*EmbeddingSearchResult, error) {
	// For now, use a simple query to get embeddings by content type
	// In production, this should use proper vector similarity search
	params := sqlc.GetEmbeddingsByTypeParams{
		ContentType: contentType,
		Limit:       int32(limit),
		Offset:      0,
	}

	sqlcEmbeddings, err := c.queries.GetEmbeddingsByType(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to search embeddings: %w", err)
	}

	results := make([]*EmbeddingSearchResult, len(sqlcEmbeddings))
	for i, sqlcEmb := range sqlcEmbeddings {
		// Mock similarity calculation for now
		similarity := 0.8 // This should be calculated using actual vector similarity
		results[i] = &EmbeddingSearchResult{
			Record:     c.convertSQLCEmbedding(sqlcEmb),
			Similarity: similarity,
			Distance:   1.0 - similarity,
		}
	}

	return results, nil
}

// Helper methods for type conversion

func (c *SQLCClient) parseUUID(id string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	err := uuid.Scan(id)
	return uuid, err
}

func (c *SQLCClient) uuidToString(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}
	// Convert UUID bytes to string format
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid.Bytes[0:4],
		uuid.Bytes[4:6],
		uuid.Bytes[6:8],
		uuid.Bytes[8:10],
		uuid.Bytes[10:16])
}

func (c *SQLCClient) vectorToPgVector(vector []float64) pgvector.Vector {
	if len(vector) == 0 {
		return pgvector.NewVector([]float32{})
	}

	// Convert float64 to float32 for pgvector
	float32Vector := make([]float32, len(vector))
	for i, v := range vector {
		float32Vector[i] = float32(v)
	}

	return pgvector.NewVector(float32Vector)
}

func (c *SQLCClient) pgVectorToVector(pgvec pgvector.Vector) []float64 {
	slice := pgvec.Slice()
	if len(slice) == 0 {
		return make([]float64, 0)
	}

	// Convert float32 to float64
	float64Vector := make([]float64, len(slice))
	for i, v := range slice {
		float64Vector[i] = float64(v)
	}

	return float64Vector
}

func (c *SQLCClient) convertSQLCConversation(sqlcConv *sqlc.Conversation) *Conversation {
	var metadata map[string]interface{}
	if len(sqlcConv.Metadata) > 0 {
		json.Unmarshal(sqlcConv.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &Conversation{
		ID:        c.uuidToString(sqlcConv.ID),
		UserID:    c.uuidToString(sqlcConv.UserID),
		Title:     sqlcConv.Title,
		Metadata:  metadata,
		CreatedAt: sqlcConv.CreatedAt,
		UpdatedAt: sqlcConv.UpdatedAt,
		Messages:  []*Message{}, // Will be loaded separately if needed
	}
}

func (c *SQLCClient) convertSQLCConversationRow(sqlcConv *sqlc.Conversation) *Conversation {
	var metadata map[string]interface{}
	if len(sqlcConv.Metadata) > 0 {
		json.Unmarshal(sqlcConv.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &Conversation{
		ID:        c.uuidToString(sqlcConv.ID),
		UserID:    c.uuidToString(sqlcConv.UserID),
		Title:     sqlcConv.Title,
		Metadata:  metadata,
		CreatedAt: sqlcConv.CreatedAt,
		UpdatedAt: sqlcConv.UpdatedAt,
		Messages:  []*Message{}, // Will be loaded separately if needed
	}
}

func (c *SQLCClient) convertSQLCMessage(sqlcMsg *sqlc.Message) *Message {
	var metadata map[string]interface{}
	if len(sqlcMsg.Metadata) > 0 {
		json.Unmarshal(sqlcMsg.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	tokenCount := 0
	if sqlcMsg.TokenCount.Valid {
		tokenCount = int(sqlcMsg.TokenCount.Int32)
	}

	return &Message{
		ID:             c.uuidToString(sqlcMsg.ID),
		ConversationID: c.uuidToString(sqlcMsg.ConversationID),
		Role:           sqlcMsg.Role,
		Content:        sqlcMsg.Content,
		TokenCount:     tokenCount,
		Metadata:       metadata,
		CreatedAt:      sqlcMsg.CreatedAt,
	}
}

func (c *SQLCClient) convertSQLCEmbedding(sqlcEmb *sqlc.Embedding) *EmbeddingRecord {
	var metadata map[string]interface{}
	if len(sqlcEmb.Metadata) > 0 {
		json.Unmarshal(sqlcEmb.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &EmbeddingRecord{
		ID:          c.uuidToString(sqlcEmb.ID),
		ContentType: sqlcEmb.ContentType,
		ContentID:   c.uuidToString(sqlcEmb.ContentID),
		ContentText: sqlcEmb.ContentText,
		Embedding:   c.pgVectorToVector(sqlcEmb.Embedding),
		Metadata:    metadata,
		CreatedAt:   sqlcEmb.CreatedAt,
	}
}

// Memory Entry domain methods

// CreateMemoryEntry creates a new memory entry
func (c *SQLCClient) CreateMemoryEntry(ctx context.Context, entry *MemoryEntryDomain) (*MemoryEntryDomain, error) {
	userUUID, err := c.parseUUID(entry.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var sessionID pgtype.Text
	if entry.SessionID != "" {
		sessionID = pgtype.Text{String: entry.SessionID, Valid: true}
	}

	var expiresAt pgtype.Timestamptz
	if entry.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *entry.ExpiresAt, Valid: true}
	}

	params := sqlc.CreateMemoryEntryParams{
		MemoryType:  entry.Type,
		Column2:     userUUID, // user_id parameter
		SessionID:   sessionID,
		Content:     entry.Content,
		Importance:  pgtype.Numeric{Valid: true},
		AccessCount: pgtype.Int4{Int32: int32(entry.AccessCount), Valid: true},
		LastAccess:  pgtype.Timestamptz{Time: entry.LastAccess, Valid: true},
		ExpiresAt:   expiresAt,
		Metadata:    metadataJSON,
	}

	sqlcEntry, err := c.queries.CreateMemoryEntry(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory entry: %w", err)
	}

	return c.convertSQLCMemoryEntry(sqlcEntry), nil
}

// GetMemoryEntriesByUser retrieves memory entries for a user
func (c *SQLCClient) GetMemoryEntriesByUser(ctx context.Context, userID string, memoryTypes []string, limit, offset int) ([]*MemoryEntryDomain, error) {
	userUUID, err := c.parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// GetMemoryEntriesByUser 現在只需要 userID 參數
	sqlcEntries, err := c.queries.GetMemoryEntriesByUser(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get memory entries: %w", err)
	}

	entries := make([]*MemoryEntryDomain, len(sqlcEntries))
	for i, sqlcEntry := range sqlcEntries {
		entries[i] = c.convertSQLCMemoryEntry(sqlcEntry)
	}

	return entries, nil
}

// Agent Execution domain methods

// CreateAgentExecution creates a new agent execution record
func (c *SQLCClient) CreateAgentExecution(ctx context.Context, execution *AgentExecutionDomain) (*AgentExecutionDomain, error) {
	userUUID, err := c.parseUUID(execution.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	stepsJSON, err := json.Marshal(execution.Steps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal steps: %w", err)
	}

	metadataJSON, err := json.Marshal(execution.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	params := sqlc.CreateAgentExecutionParams{
		AgentType:       execution.AgentType,
		Column2:         userUUID, // user_id parameter
		Query:           execution.Query,
		Response:        pgtype.Text{String: execution.Response, Valid: execution.Response != ""},
		Steps:           stepsJSON,
		ExecutionTimeMs: pgtype.Int4{Int32: int32(execution.ExecutionTimeMs), Valid: true},
		Success:         pgtype.Bool{Bool: execution.Success, Valid: true},
		ErrorMessage:    pgtype.Text{String: execution.ErrorMessage, Valid: execution.ErrorMessage != ""},
		Metadata:        metadataJSON,
	}

	sqlcExecution, err := c.queries.CreateAgentExecution(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent execution: %w", err)
	}

	return c.convertSQLCAgentExecution(sqlcExecution), nil
}

// Chain Execution domain methods

// CreateChainExecution creates a new chain execution record
func (c *SQLCClient) CreateChainExecution(ctx context.Context, execution *ChainExecutionDomain) (*ChainExecutionDomain, error) {
	userUUID, err := c.parseUUID(execution.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	stepsJSON, err := json.Marshal(execution.Steps)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal steps: %w", err)
	}

	metadataJSON, err := json.Marshal(execution.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	params := sqlc.CreateChainExecutionParams{
		ChainType:       execution.ChainType,
		Column2:         userUUID, // user_id parameter
		Input:           execution.Input,
		Output:          pgtype.Text{String: execution.Output, Valid: execution.Output != ""},
		Steps:           stepsJSON,
		ExecutionTimeMs: pgtype.Int4{Int32: int32(execution.ExecutionTimeMs), Valid: true},
		TokensUsed:      pgtype.Int4{Int32: int32(execution.TokensUsed), Valid: true},
		Success:         pgtype.Bool{Bool: execution.Success, Valid: true},
		ErrorMessage:    pgtype.Text{String: execution.ErrorMessage, Valid: execution.ErrorMessage != ""},
		Metadata:        metadataJSON,
	}

	sqlcExecution, err := c.queries.CreateChainExecution(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create chain execution: %w", err)
	}

	return c.convertSQLCChainExecution(sqlcExecution), nil
}

// Tool Cache domain methods

// CreateToolCacheEntry creates or updates a tool cache entry
func (c *SQLCClient) CreateToolCacheEntry(ctx context.Context, entry *ToolCacheDomain) (*ToolCacheDomain, error) {
	userUUID, err := c.parseUUID(entry.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	inputDataJSON, err := json.Marshal(entry.InputData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input data: %w", err)
	}

	outputDataJSON, err := json.Marshal(entry.OutputData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output data: %w", err)
	}

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	params := sqlc.CreateToolCacheEntryParams{
		Column1:         userUUID, // user_id parameter
		ToolName:        entry.ToolName,
		InputHash:       entry.InputHash,
		InputData:       inputDataJSON,
		OutputData:      outputDataJSON,
		ExecutionTimeMs: pgtype.Int4{Int32: int32(entry.ExecutionTimeMs), Valid: true},
		Success:         pgtype.Bool{Bool: entry.Success, Valid: true},
		ErrorMessage:    pgtype.Text{String: entry.ErrorMessage, Valid: entry.ErrorMessage != ""},
		ExpiresAt:       pgtype.Timestamptz{Time: entry.ExpiresAt, Valid: true},
		Metadata:        metadataJSON,
	}

	sqlcEntry, err := c.queries.CreateToolCacheEntry(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool cache entry: %w", err)
	}

	return c.convertSQLCToolCache(sqlcEntry), nil
}

// GetToolCacheEntry retrieves a specific tool cache entry
func (c *SQLCClient) GetToolCacheEntry(ctx context.Context, userID, toolName, inputHash string) (*ToolCacheDomain, error) {
	userUUID, err := c.parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	params := sqlc.GetToolCacheEntryParams{
		Column1:   userUUID, // user_id parameter
		ToolName:  toolName,
		InputHash: inputHash,
	}

	sqlcEntry, err := c.queries.GetToolCacheEntry(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get tool cache entry: %w", err)
	}

	return c.convertSQLCToolCache(sqlcEntry), nil
}

// User Preferences domain methods

// CreateUserPreference creates or updates a user preference
func (c *SQLCClient) CreateUserPreference(ctx context.Context, pref *UserPreferenceDomain) (*UserPreferenceDomain, error) {
	userUUID, err := c.parseUUID(pref.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	valueJSON, err := json.Marshal(pref.PreferenceValue)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal preference value: %w", err)
	}

	metadataJSON, err := json.Marshal(pref.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	params := sqlc.CreateUserPreferenceParams{
		Column1:         userUUID, // user_id parameter
		Category:        pref.Category,
		PreferenceKey:   pref.PreferenceKey,
		PreferenceValue: valueJSON,
		ValueType:       pref.ValueType,
		Description:     pgtype.Text{String: pref.Description, Valid: pref.Description != ""},
		Metadata:        metadataJSON,
	}

	sqlcPref, err := c.queries.CreateUserPreference(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user preference: %w", err)
	}

	return c.convertSQLCUserPreference(sqlcPref), nil
}

// User Context domain methods

// CreateUserContext creates or updates user context
func (c *SQLCClient) CreateUserContext(ctx context.Context, userCtx *UserContextDomain) (*UserContextDomain, error) {
	userUUID, err := c.parseUUID(userCtx.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	valueJSON, err := json.Marshal(userCtx.ContextValue)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal context value: %w", err)
	}

	metadataJSON, err := json.Marshal(userCtx.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var expiresAt pgtype.Timestamptz
	if userCtx.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *userCtx.ExpiresAt, Valid: true}
	}

	params := sqlc.CreateUserContextParams{
		Column1:      userUUID, // user_id parameter
		ContextType:  userCtx.ContextType,
		ContextKey:   userCtx.ContextKey,
		ContextValue: valueJSON,
		Importance:   pgtype.Numeric{Valid: true}, // TODO: Fix numeric value handling
		ExpiresAt:    expiresAt,
		Metadata:     metadataJSON,
	}

	sqlcCtx, err := c.queries.CreateUserContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create user context: %w", err)
	}

	return c.convertSQLCUserContext(sqlcCtx), nil
}

// Conversion methods for domain types

func (c *SQLCClient) convertSQLCMemoryEntry(sqlcEntry *sqlc.MemoryEntry) *MemoryEntryDomain {
	var metadata map[string]interface{}
	if len(sqlcEntry.Metadata) > 0 {
		json.Unmarshal(sqlcEntry.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	importance := 0.5
	if sqlcEntry.Importance.Valid {
		// Convert pgtype.Numeric to float64
		if sqlcEntry.Importance.Int != nil {
			importance = float64(sqlcEntry.Importance.Int.Int64()) / 100.0
		}
	}

	sessionID := ""
	if sqlcEntry.SessionID.Valid {
		sessionID = sqlcEntry.SessionID.String
	}

	var expiresAt *time.Time
	if sqlcEntry.ExpiresAt.Valid {
		expiresAt = &sqlcEntry.ExpiresAt.Time
	}

	accessCount := 0
	if sqlcEntry.AccessCount.Valid {
		accessCount = int(sqlcEntry.AccessCount.Int32)
	}

	lastAccess := time.Now()
	if sqlcEntry.LastAccess.Valid {
		lastAccess = sqlcEntry.LastAccess.Time
	}

	return &MemoryEntryDomain{
		ID:          c.uuidToString(sqlcEntry.ID),
		Type:        sqlcEntry.MemoryType,
		UserID:      c.uuidToString(sqlcEntry.UserID),
		SessionID:   sessionID,
		Content:     sqlcEntry.Content,
		Importance:  importance,
		AccessCount: accessCount,
		LastAccess:  lastAccess,
		CreatedAt:   sqlcEntry.CreatedAt,
		ExpiresAt:   expiresAt,
		Metadata:    metadata,
	}
}

func (c *SQLCClient) convertSQLCAgentExecution(sqlcExec *sqlc.AgentExecution) *AgentExecutionDomain {
	var metadata map[string]interface{}
	if len(sqlcExec.Metadata) > 0 {
		json.Unmarshal(sqlcExec.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	var steps []interface{}
	if len(sqlcExec.Steps) > 0 {
		json.Unmarshal(sqlcExec.Steps, &steps)
	}
	if steps == nil {
		steps = make([]interface{}, 0)
	}

	response := ""
	if sqlcExec.Response.Valid {
		response = sqlcExec.Response.String
	}

	executionTimeMs := 0
	if sqlcExec.ExecutionTimeMs.Valid {
		executionTimeMs = int(sqlcExec.ExecutionTimeMs.Int32)
	}

	success := false
	if sqlcExec.Success.Valid {
		success = sqlcExec.Success.Bool
	}

	errorMessage := ""
	if sqlcExec.ErrorMessage.Valid {
		errorMessage = sqlcExec.ErrorMessage.String
	}

	conversationID := ""
	if sqlcExec.ConversationID.Valid {
		conversationID = c.uuidToString(sqlcExec.ConversationID)
	}

	return &AgentExecutionDomain{
		ID:              c.uuidToString(sqlcExec.ID),
		AgentType:       sqlcExec.AgentType,
		UserID:          c.uuidToString(sqlcExec.UserID),
		ConversationID:  conversationID,
		Query:           sqlcExec.Query,
		Response:        response,
		Steps:           steps,
		ExecutionTimeMs: executionTimeMs,
		Success:         success,
		ErrorMessage:    errorMessage,
		CreatedAt:       sqlcExec.CreatedAt,
		Metadata:        metadata,
	}
}

func (c *SQLCClient) convertSQLCChainExecution(sqlcExec *sqlc.ChainExecution) *ChainExecutionDomain {
	var metadata map[string]interface{}
	if len(sqlcExec.Metadata) > 0 {
		json.Unmarshal(sqlcExec.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	var steps []interface{}
	if len(sqlcExec.Steps) > 0 {
		json.Unmarshal(sqlcExec.Steps, &steps)
	}
	if steps == nil {
		steps = make([]interface{}, 0)
	}

	output := ""
	if sqlcExec.Output.Valid {
		output = sqlcExec.Output.String
	}

	executionTimeMs := 0
	if sqlcExec.ExecutionTimeMs.Valid {
		executionTimeMs = int(sqlcExec.ExecutionTimeMs.Int32)
	}

	tokensUsed := 0
	if sqlcExec.TokensUsed.Valid {
		tokensUsed = int(sqlcExec.TokensUsed.Int32)
	}

	success := false
	if sqlcExec.Success.Valid {
		success = sqlcExec.Success.Bool
	}

	errorMessage := ""
	if sqlcExec.ErrorMessage.Valid {
		errorMessage = sqlcExec.ErrorMessage.String
	}

	conversationID := ""
	if sqlcExec.ConversationID.Valid {
		conversationID = c.uuidToString(sqlcExec.ConversationID)
	}

	return &ChainExecutionDomain{
		ID:              c.uuidToString(sqlcExec.ID),
		ChainType:       sqlcExec.ChainType,
		UserID:          c.uuidToString(sqlcExec.UserID),
		ConversationID:  conversationID,
		Input:           sqlcExec.Input,
		Output:          output,
		Steps:           steps,
		ExecutionTimeMs: executionTimeMs,
		TokensUsed:      tokensUsed,
		Success:         success,
		ErrorMessage:    errorMessage,
		CreatedAt:       sqlcExec.CreatedAt,
		Metadata:        metadata,
	}
}

func (c *SQLCClient) convertSQLCToolCache(sqlcCache *sqlc.ToolCache) *ToolCacheDomain {
	var metadata map[string]interface{}
	if len(sqlcCache.Metadata) > 0 {
		json.Unmarshal(sqlcCache.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	var inputData map[string]interface{}
	if len(sqlcCache.InputData) > 0 {
		json.Unmarshal(sqlcCache.InputData, &inputData)
	}
	if inputData == nil {
		inputData = make(map[string]interface{})
	}

	var outputData map[string]interface{}
	if len(sqlcCache.OutputData) > 0 {
		json.Unmarshal(sqlcCache.OutputData, &outputData)
	}
	if outputData == nil {
		outputData = make(map[string]interface{})
	}

	executionTimeMs := 0
	if sqlcCache.ExecutionTimeMs.Valid {
		executionTimeMs = int(sqlcCache.ExecutionTimeMs.Int32)
	}

	success := false
	if sqlcCache.Success.Valid {
		success = sqlcCache.Success.Bool
	}

	errorMessage := ""
	if sqlcCache.ErrorMessage.Valid {
		errorMessage = sqlcCache.ErrorMessage.String
	}

	hitCount := 0
	if sqlcCache.HitCount.Valid {
		hitCount = int(sqlcCache.HitCount.Int32)
	}

	lastHit := time.Now()
	if sqlcCache.LastHit.Valid {
		lastHit = sqlcCache.LastHit.Time
	}

	expiresAt := time.Now().Add(time.Hour * 6) // Default 6 hours
	if sqlcCache.ExpiresAt.Valid {
		expiresAt = sqlcCache.ExpiresAt.Time
	}

	return &ToolCacheDomain{
		ID:              c.uuidToString(sqlcCache.ID),
		UserID:          c.uuidToString(sqlcCache.UserID),
		ToolName:        sqlcCache.ToolName,
		InputHash:       sqlcCache.InputHash,
		InputData:       inputData,
		OutputData:      outputData,
		ExecutionTimeMs: executionTimeMs,
		Success:         success,
		ErrorMessage:    errorMessage,
		HitCount:        hitCount,
		LastHit:         lastHit,
		CreatedAt:       sqlcCache.CreatedAt,
		ExpiresAt:       expiresAt,
		Metadata:        metadata,
	}
}

func (c *SQLCClient) convertSQLCUserPreference(sqlcPref *sqlc.UserPreference) *UserPreferenceDomain {
	var metadata map[string]interface{}
	if len(sqlcPref.Metadata) > 0 {
		json.Unmarshal(sqlcPref.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	var preferenceValue map[string]interface{}
	if len(sqlcPref.PreferenceValue) > 0 {
		json.Unmarshal(sqlcPref.PreferenceValue, &preferenceValue)
	}
	if preferenceValue == nil {
		preferenceValue = make(map[string]interface{})
	}

	description := ""
	if sqlcPref.Description.Valid {
		description = sqlcPref.Description.String
	}

	return &UserPreferenceDomain{
		ID:              c.uuidToString(sqlcPref.ID),
		UserID:          c.uuidToString(sqlcPref.UserID),
		Category:        sqlcPref.Category,
		PreferenceKey:   sqlcPref.PreferenceKey,
		PreferenceValue: preferenceValue,
		ValueType:       sqlcPref.ValueType,
		Description:     description,
		CreatedAt:       sqlcPref.CreatedAt,
		UpdatedAt:       sqlcPref.UpdatedAt,
		Metadata:        metadata,
	}
}

func (c *SQLCClient) convertSQLCUserContext(sqlcCtx *sqlc.UserContext) *UserContextDomain {
	var metadata map[string]interface{}
	if len(sqlcCtx.Metadata) > 0 {
		json.Unmarshal(sqlcCtx.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	var contextValue map[string]interface{}
	if len(sqlcCtx.ContextValue) > 0 {
		json.Unmarshal(sqlcCtx.ContextValue, &contextValue)
	}
	if contextValue == nil {
		contextValue = make(map[string]interface{})
	}

	importance := 0.5
	if sqlcCtx.Importance.Valid {
		// Convert pgtype.Numeric to float64
		if sqlcCtx.Importance.Int != nil {
			importance = float64(sqlcCtx.Importance.Int.Int64()) / 100.0
		}
	}

	var expiresAt *time.Time
	if sqlcCtx.ExpiresAt.Valid {
		expiresAt = &sqlcCtx.ExpiresAt.Time
	}

	return &UserContextDomain{
		ID:           c.uuidToString(sqlcCtx.ID),
		UserID:       c.uuidToString(sqlcCtx.UserID),
		ContextType:  sqlcCtx.ContextType,
		ContextKey:   sqlcCtx.ContextKey,
		ContextValue: contextValue,
		Importance:   importance,
		CreatedAt:    sqlcCtx.CreatedAt,
		UpdatedAt:    sqlcCtx.UpdatedAt,
		ExpiresAt:    expiresAt,
		Metadata:     metadata,
	}
}

// Additional conversion functions for different SQLC row types

func (c *SQLCClient) convertSQLCConversationFromGetRow(sqlcConv *sqlc.Conversation) *Conversation {
	var metadata map[string]interface{}
	if len(sqlcConv.Metadata) > 0 {
		json.Unmarshal(sqlcConv.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &Conversation{
		ID:        c.uuidToString(sqlcConv.ID),
		UserID:    c.uuidToString(sqlcConv.UserID),
		Title:     sqlcConv.Title,
		Metadata:  metadata,
		CreatedAt: sqlcConv.CreatedAt,
		UpdatedAt: sqlcConv.UpdatedAt,
		Messages:  []*Message{}, // Will be loaded separately if needed
	}
}

func (c *SQLCClient) convertSQLCConversationFromGetByUserRow(sqlcConv *sqlc.Conversation) *Conversation {
	var metadata map[string]interface{}
	if len(sqlcConv.Metadata) > 0 {
		json.Unmarshal(sqlcConv.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &Conversation{
		ID:        c.uuidToString(sqlcConv.ID),
		UserID:    c.uuidToString(sqlcConv.UserID),
		Title:     sqlcConv.Title,
		Metadata:  metadata,
		CreatedAt: sqlcConv.CreatedAt,
		UpdatedAt: sqlcConv.UpdatedAt,
		Messages:  []*Message{}, // Will be loaded separately if needed
	}
}

func (c *SQLCClient) convertSQLCMessageFromCreateRow(sqlcMsg *sqlc.CreateMessageRow) *Message {
	var metadata map[string]interface{}
	if len(sqlcMsg.Metadata) > 0 {
		json.Unmarshal(sqlcMsg.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	tokenCount := 0
	if sqlcMsg.TokenCount.Valid {
		tokenCount = int(sqlcMsg.TokenCount.Int32)
	}

	return &Message{
		ID:             c.uuidToString(sqlcMsg.ID),
		ConversationID: c.uuidToString(sqlcMsg.ConversationID),
		Role:           sqlcMsg.Role,
		Content:        sqlcMsg.Content,
		TokenCount:     tokenCount,
		Metadata:       metadata,
		CreatedAt:      sqlcMsg.CreatedAt,
	}
}

func (c *SQLCClient) convertSQLCMessageFromGetAllRow(sqlcMsg *sqlc.GetAllMessagesByConversationRow) *Message {
	var metadata map[string]interface{}
	if len(sqlcMsg.Metadata) > 0 {
		json.Unmarshal(sqlcMsg.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	tokenCount := 0
	if sqlcMsg.TokenCount.Valid {
		tokenCount = int(sqlcMsg.TokenCount.Int32)
	}

	return &Message{
		ID:             c.uuidToString(sqlcMsg.ID),
		ConversationID: c.uuidToString(sqlcMsg.ConversationID),
		Role:           sqlcMsg.Role,
		Content:        sqlcMsg.Content,
		TokenCount:     tokenCount,
		Metadata:       metadata,
		CreatedAt:      sqlcMsg.CreatedAt,
	}
}

func (c *SQLCClient) convertSQLCMessageFromGetRow(sqlcMsg *sqlc.GetMessagesByConversationRow) *Message {
	var metadata map[string]interface{}
	if len(sqlcMsg.Metadata) > 0 {
		json.Unmarshal(sqlcMsg.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	tokenCount := 0
	if sqlcMsg.TokenCount.Valid {
		tokenCount = int(sqlcMsg.TokenCount.Int32)
	}

	return &Message{
		ID:             c.uuidToString(sqlcMsg.ID),
		ConversationID: c.uuidToString(sqlcMsg.ConversationID),
		Role:           sqlcMsg.Role,
		Content:        sqlcMsg.Content,
		TokenCount:     tokenCount,
		Metadata:       metadata,
		CreatedAt:      sqlcMsg.CreatedAt,
	}
}
