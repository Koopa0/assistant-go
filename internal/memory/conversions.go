package memory

import (
	"encoding/json"
	"log/slog"

	"github.com/koopa0/assistant-go/internal/platform/storage/postgres/sqlc"
)

// Type conversion helpers for sqlc types to domain types
// Following Go principle: No pointers for small structs unless needed for optionality

func convertCreateMemoryEntryRow(row *sqlc.CreateMemoryEntryRow, logger *slog.Logger) *Entry {
	var importance float64
	if row.Importance.Valid {
		f8, _ := row.Importance.Float64Value()
		importance = f8.Float64
	}

	entry := &Entry{
		ID:          row.ID.String(),
		Type:        Type(row.MemoryType),
		UserID:      row.UserID.String(),
		Content:     row.Content,
		Importance:  importance,
		AccessCount: int(row.AccessCount.Int32),
		LastAccess:  row.LastAccess.Time,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		Version:     0, // Version field doesn't exist in CreateMemoryEntryRow
	}

	// Handle optional fields
	if row.SessionID.Valid && row.SessionID.String != "" {
		entry.SessionID = &row.SessionID.String
	}
	if row.ExpiresAt.Valid {
		entry.ExpiresAt = &row.ExpiresAt.Time
	}

	// Parse metadata
	parseMetadata(row.Metadata, entry, logger)

	return entry
}

func convertGetMemoryEntryRow(row *sqlc.GetMemoryEntryRow, logger *slog.Logger) *Entry {
	var importance float64
	if row.Importance.Valid {
		f8, _ := row.Importance.Float64Value()
		importance = f8.Float64
	}

	entry := &Entry{
		ID:          row.ID.String(),
		Type:        Type(row.MemoryType),
		UserID:      row.UserID.String(),
		Content:     row.Content,
		Importance:  importance,
		AccessCount: int(row.AccessCount.Int32),
		LastAccess:  row.LastAccess.Time,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		Version:     0, // Version field doesn't exist in GetMemoryEntryRow
	}

	// Handle optional fields
	if row.SessionID.Valid && row.SessionID.String != "" {
		entry.SessionID = &row.SessionID.String
	}
	if row.ExpiresAt.Valid {
		entry.ExpiresAt = &row.ExpiresAt.Time
	}

	// Parse metadata
	parseMetadata(row.Metadata, entry, logger)

	return entry
}

func convertUpdateMemoryEntryRow(row *sqlc.UpdateMemoryEntryRow, logger *slog.Logger) *Entry {
	var importance float64
	if row.Importance.Valid {
		f8, _ := row.Importance.Float64Value()
		importance = f8.Float64
	}

	entry := &Entry{
		ID:          row.ID.String(),
		Type:        Type(row.MemoryType),
		UserID:      row.UserID.String(),
		Content:     row.Content,
		Importance:  importance,
		AccessCount: int(row.AccessCount.Int32),
		LastAccess:  row.LastAccess.Time,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		Version:     0, // Version field doesn't exist in UpdateMemoryEntryRow
	}

	// Handle optional fields
	if row.SessionID.Valid && row.SessionID.String != "" {
		entry.SessionID = &row.SessionID.String
	}
	if row.ExpiresAt.Valid {
		entry.ExpiresAt = &row.ExpiresAt.Time
	}

	// Parse metadata
	parseMetadata(row.Metadata, entry, logger)

	return entry
}

func convertSearchMemoryEntriesRow(row *sqlc.SearchMemoryEntriesRow, logger *slog.Logger) *Entry {
	var importance float64
	if row.Importance.Valid {
		f8, _ := row.Importance.Float64Value()
		importance = f8.Float64
	}

	entry := &Entry{
		ID:          row.ID.String(),
		Type:        Type(row.MemoryType),
		UserID:      row.UserID.String(),
		Content:     row.Content,
		Importance:  importance,
		AccessCount: int(row.AccessCount.Int32),
		LastAccess:  row.LastAccess.Time,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		Version:     0, // Version field doesn't exist in SearchMemoryEntriesRow
	}

	// Handle optional fields
	if row.SessionID.Valid && row.SessionID.String != "" {
		entry.SessionID = &row.SessionID.String
	}
	if row.ExpiresAt.Valid {
		entry.ExpiresAt = &row.ExpiresAt.Time
	}

	// Parse metadata
	parseMetadata(row.Metadata, entry, logger)

	return entry
}

// parseMetadata extracts metadata from JSON bytes
func parseMetadata(data []byte, entry *Entry, logger *slog.Logger) {
	if len(data) == 0 || string(data) == "{}" {
		return
	}

	// Try to parse as EntryMetadata first
	if err := json.Unmarshal(data, &entry.Metadata); err == nil {
		return
	}

	// Fall back to Context for backward compatibility
	context := make(map[string]interface{})
	if err := json.Unmarshal(data, &context); err != nil {
		logger.Error("Failed to parse memory metadata",
			slog.String("memory_id", entry.ID),
			slog.Any("error", err))
		return
	}

	// Convert context to metadata where possible
	if category, ok := context["category"].(string); ok {
		entry.Metadata.Category = category
	}
	if tags, ok := context["tags"].([]interface{}); ok {
		entry.Metadata.Tags = make([]string, 0, len(tags))
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				entry.Metadata.Tags = append(entry.Metadata.Tags, tagStr)
			}
		}
	}
	if source, ok := context["source"].(string); ok {
		entry.Metadata.Source = source
	}
	if confidence, ok := context["confidence"].(float64); ok {
		entry.Metadata.Confidence = confidence
	}

	// Store remaining fields in Context for backward compatibility
	entry.Context = context
}
