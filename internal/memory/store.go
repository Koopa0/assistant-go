package memory

import (
	"context"
	"fmt"
	"log/slog"
)

// DatabaseMemoryStore implements MemoryStore using the memory.Service
// This is a direct integration pattern, no unnecessary abstractions
type DatabaseMemoryStore struct {
	memService MemoryService
	logger     *slog.Logger
}

// NewDatabaseMemoryStore creates a new database-backed memory store
func NewDatabaseMemoryStore(memService MemoryService, logger *slog.Logger) *DatabaseMemoryStore {
	return &DatabaseMemoryStore{
		memService: memService,
		logger:     logger,
	}
}

// Store implements MemoryStore.Store
func (s *DatabaseMemoryStore) Store(ctx context.Context, entry Entry) error {
	return s.memService.Store(ctx, entry)
}

// Retrieve implements MemoryStore.Retrieve
func (s *DatabaseMemoryStore) Retrieve(ctx context.Context, criteria SearchCriteria) ([]Entry, error) {
	return s.memService.Search(ctx, criteria)
}

// Update implements MemoryStore.Update
// Note: The memory.Service expects a full Entry for updates, not a map
func (s *DatabaseMemoryStore) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	// First retrieve the existing entry
	entry, err := s.memService.Retrieve(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to retrieve entry for update: %w", err)
	}

	// Apply updates to the entry
	for key, value := range updates {
		switch key {
		case "content":
			if content, ok := value.(string); ok {
				entry.Content = content
			}
		case "importance":
			if importance, ok := value.(float64); ok {
				entry.Importance = importance
			}
		case "metadata":
			if metadata, ok := value.(map[string]interface{}); ok {
				// Update metadata fields
				for k, v := range metadata {
					switch k {
					case "category":
						if cat, ok := v.(string); ok {
							entry.Metadata.Category = cat
						}
					case "tags":
						if tags, ok := v.([]string); ok {
							entry.Metadata.Tags = tags
						}
					case "source":
						if src, ok := v.(string); ok {
							entry.Metadata.Source = src
						}
					case "confidence":
						if conf, ok := v.(float64); ok {
							entry.Metadata.Confidence = conf
						}
					default:
						// Store in Extra
						if entry.Metadata.Extra == nil {
							entry.Metadata.Extra = make(map[string]interface{})
						}
						entry.Metadata.Extra[k] = v
					}
				}
			}
		}
	}

	// Update the entry
	return s.memService.Update(ctx, *entry)
}

// Delete implements MemoryStore.Delete
func (s *DatabaseMemoryStore) Delete(ctx context.Context, id string) error {
	return s.memService.Delete(ctx, id)
}

// SearchRelated implements MemoryStore.SearchRelated
// Since the memory.Service doesn't have a specific SearchRelated method,
// we implement it using semantic search with the query being the entry's content
func (s *DatabaseMemoryStore) SearchRelated(ctx context.Context, entryID string, maxResults int) ([]Entry, error) {
	// First get the entry
	entry, err := s.memService.Retrieve(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve entry for related search: %w", err)
	}

	// Search for related entries using the content as query
	criteria := SearchCriteria{
		UserID: entry.UserID,
		Query:  entry.Content,
		Limit:  maxResults,
	}

	// Get results
	results, err := s.memService.Search(ctx, criteria)
	if err != nil {
		return nil, err
	}

	// Filter out the original entry
	filtered := make([]Entry, 0, len(results))
	for _, r := range results {
		if r.ID != entryID {
			filtered = append(filtered, r)
		}
	}

	return filtered, nil
}
