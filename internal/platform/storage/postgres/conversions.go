// Package postgres provides PostgreSQL database connectivity and operations.
package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
	"github.com/pgvector/pgvector-go"
)

// ConversionHelpers provides utility functions for converting between sqlc and domain types
type ConversionHelpers struct{}

// NewConversionHelpers creates a new instance of conversion helpers
func NewConversionHelpers() *ConversionHelpers {
	return &ConversionHelpers{}
}

// ParseUUID parses a string UUID into pgtype.UUID
func ParseUUID(id string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	err := uuid.Scan(id)
	return uuid, err
}

// UUIDToString converts pgtype.UUID to string
func UUIDToString(uuid pgtype.UUID) string {
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

// VectorToPgVector converts []float64 to pgvector.Vector
func VectorToPgVector(vector []float64) pgvector.Vector {
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

// PgVectorToVector converts pgvector.Vector to []float64
func PgVectorToVector(pgvec pgvector.Vector) []float64 {
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

// ConvertSQLCConversation converts sqlc.Conversation to domain Conversation
func ConvertSQLCConversation(sqlcConv *sqlc.Conversation) *Conversation {
	var metadata map[string]interface{}
	if len(sqlcConv.Metadata) > 0 {
		json.Unmarshal(sqlcConv.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &Conversation{
		ID:        UUIDToString(sqlcConv.ID),
		UserID:    UUIDToString(sqlcConv.UserID),
		Title:     sqlcConv.Title,
		Metadata:  metadata,
		CreatedAt: sqlcConv.CreatedAt,
		UpdatedAt: sqlcConv.UpdatedAt,
		Messages:  []*Message{}, // Will be loaded separately if needed
	}
}

// ConvertSQLCMessage converts various sqlc message types to domain Message
func ConvertSQLCMessage(sqlcMsg interface{}) *Message {
	switch msg := sqlcMsg.(type) {
	case *sqlc.Message:
		return convertBaseSQLCMessage(msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.TokenCount, msg.Metadata, msg.CreatedAt)
	case *sqlc.CreateMessageRow:
		return convertBaseSQLCMessage(msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.TokenCount, msg.Metadata, msg.CreatedAt)
	case *sqlc.GetAllMessagesByConversationRow:
		return convertBaseSQLCMessage(msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.TokenCount, msg.Metadata, msg.CreatedAt)
	case *sqlc.GetMessagesByConversationRow:
		return convertBaseSQLCMessage(msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.TokenCount, msg.Metadata, msg.CreatedAt)
	default:
		return nil
	}
}

func convertBaseSQLCMessage(id, conversationID pgtype.UUID, role, content string, tokenCount pgtype.Int4, metadata []byte, createdAt time.Time) *Message {
	var metadataMap map[string]interface{}
	if len(metadata) > 0 {
		json.Unmarshal(metadata, &metadataMap)
	}
	if metadataMap == nil {
		metadataMap = make(map[string]interface{})
	}

	tokenCountInt := 0
	if tokenCount.Valid {
		tokenCountInt = int(tokenCount.Int32)
	}

	return &Message{
		ID:             UUIDToString(id),
		ConversationID: UUIDToString(conversationID),
		Role:           role,
		Content:        content,
		TokenCount:     tokenCountInt,
		Metadata:       metadataMap,
		CreatedAt:      createdAt,
	}
}

// ConvertSQLCEmbedding converts sqlc.Embedding to domain EmbeddingRecord
func ConvertSQLCEmbedding(sqlcEmb *sqlc.Embedding) *EmbeddingRecord {
	var metadata map[string]interface{}
	if len(sqlcEmb.Metadata) > 0 {
		json.Unmarshal(sqlcEmb.Metadata, &metadata)
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &EmbeddingRecord{
		ID:          UUIDToString(sqlcEmb.ID),
		ContentType: sqlcEmb.ContentType,
		ContentID:   UUIDToString(sqlcEmb.ContentID),
		ContentText: sqlcEmb.ContentText,
		Embedding:   PgVectorToVector(sqlcEmb.Embedding),
		Metadata:    metadata,
		CreatedAt:   sqlcEmb.CreatedAt,
	}
}

// ConvertSQLCMemoryEntry converts sqlc.MemoryEntry to domain MemoryEntryDomain
func ConvertSQLCMemoryEntry(sqlcEntry *sqlc.MemoryEntry) *MemoryEntryDomain {
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
		ID:          UUIDToString(sqlcEntry.ID),
		Type:        sqlcEntry.MemoryType,
		UserID:      UUIDToString(sqlcEntry.UserID),
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

// ConvertSQLCAgentExecution converts sqlc.AgentExecution to domain AgentExecutionDomain
func ConvertSQLCAgentExecution(sqlcExec *sqlc.AgentExecution) *AgentExecutionDomain {
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
		conversationID = UUIDToString(sqlcExec.ConversationID)
	}

	return &AgentExecutionDomain{
		ID:              UUIDToString(sqlcExec.ID),
		AgentType:       sqlcExec.AgentType,
		UserID:          UUIDToString(sqlcExec.UserID),
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

// ConvertSQLCChainExecution converts sqlc.ChainExecution to domain ChainExecutionDomain
func ConvertSQLCChainExecution(sqlcExec *sqlc.ChainExecution) *ChainExecutionDomain {
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
		conversationID = UUIDToString(sqlcExec.ConversationID)
	}

	return &ChainExecutionDomain{
		ID:              UUIDToString(sqlcExec.ID),
		ChainType:       sqlcExec.ChainType,
		UserID:          UUIDToString(sqlcExec.UserID),
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

// ConvertSQLCToolCache converts sqlc.ToolCache to domain ToolCacheDomain
func ConvertSQLCToolCache(sqlcCache *sqlc.ToolCache) *ToolCacheDomain {
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
		ID:              UUIDToString(sqlcCache.ID),
		UserID:          UUIDToString(sqlcCache.UserID),
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

// ConvertSQLCUserPreference converts sqlc.UserPreference to domain UserPreferenceDomain
func ConvertSQLCUserPreference(sqlcPref *sqlc.UserPreference) *UserPreferenceDomain {
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
		ID:              UUIDToString(sqlcPref.ID),
		UserID:          UUIDToString(sqlcPref.UserID),
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

// ConvertSQLCUserContext converts sqlc.UserContext to domain UserContextDomain
func ConvertSQLCUserContext(sqlcCtx *sqlc.UserContext) *UserContextDomain {
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
		ID:           UUIDToString(sqlcCtx.ID),
		UserID:       UUIDToString(sqlcCtx.UserID),
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
