package assistant

import (
	"context"

	"github.com/koopa0/assistant-go/internal/core/memory"
)

// AssistantMemory defines what the assistant actually needs from memory
// Small interface discovered from actual usage, not predetermined abstraction
type AssistantMemory interface {
	// StoreUserPreference saves a user preference for future context
	StoreUserPreference(ctx context.Context, userID, key, value string) error

	// GetUserPreferences retrieves user preferences for context building
	GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error)

	// StoreWorkingContext saves current working context
	StoreWorkingContext(ctx context.Context, userID, context string) error

	// GetWorkingContext gets current working context
	GetWorkingContext(ctx context.Context, userID string) (string, error)

	// StoreTopicDiscussion records a topic that was discussed
	StoreTopicDiscussion(ctx context.Context, userID, topic string) error

	// GetRecentTopics gets recently discussed topics for context
	GetRecentTopics(ctx context.Context, userID string, limit int) ([]string, error)
}

// MemoryAdapter adapts the concrete memory.Memory to what assistant needs
// This implements the "Accept interfaces, return concrete types" principle
type MemoryAdapter struct {
	mem *memory.Memory
}

// NewMemoryAdapter creates an adapter from concrete memory
func NewMemoryAdapter(mem *memory.Memory) *MemoryAdapter {
	return &MemoryAdapter{mem: mem}
}

// StoreUserPreference implements AssistantMemory
func (m *MemoryAdapter) StoreUserPreference(ctx context.Context, userID, key, value string) error {
	entry := memory.Entry{
		Type:    memory.TypeSemantic,
		UserID:  userID,
		Content: value,
		Context: map[string]interface{}{
			"type":           "user_preference",
			"preference_key": key,
		},
		Importance: 0.8, // User preferences are important
	}

	return m.mem.Store(ctx, entry)
}

// GetUserPreferences implements AssistantMemory
func (m *MemoryAdapter) GetUserPreferences(ctx context.Context, userID string) (*UserPreferences, error) {
	criteria := memory.SearchCriteria{
		UserID: userID,
		Types:  []memory.Type{memory.TypeSemantic},
		Limit:  20, // Get recent preferences
	}

	entries, err := m.mem.Search(ctx, criteria)
	if err != nil {
		return nil, err
	}

	// Build preferences from memory entries
	prefs := &UserPreferences{
		PreferredLanguage: "en",
		CodeStyle:         "standard",
		Documentation:     "standard",
		ResponseLanguage:  "en",
	}

	for _, entry := range entries {
		if entryType, ok := entry.Context["type"].(string); ok && entryType == "user_preference" {
			if key, ok := entry.Context["preference_key"].(string); ok {
				switch key {
				case "preferred_language":
					prefs.PreferredLanguage = entry.Content
				case "code_style":
					prefs.CodeStyle = entry.Content
				case "documentation":
					prefs.Documentation = entry.Content
				case "response_language":
					prefs.ResponseLanguage = entry.Content
				case "language_restriction":
					prefs.LanguageRestriction = entry.Content
				}
			}
		}
	}

	return prefs, nil
}

// StoreWorkingContext implements AssistantMemory
func (m *MemoryAdapter) StoreWorkingContext(ctx context.Context, userID, context string) error {
	entry := memory.Entry{
		Type:    memory.TypeWorking,
		UserID:  userID,
		Content: context,
		Context: map[string]interface{}{
			"type": "working_context",
		},
		Importance: 0.9, // Working context is very important
	}

	return m.mem.Store(ctx, entry)
}

// GetWorkingContext implements AssistantMemory
func (m *MemoryAdapter) GetWorkingContext(ctx context.Context, userID string) (string, error) {
	criteria := memory.SearchCriteria{
		UserID: userID,
		Types:  []memory.Type{memory.TypeWorking},
		Limit:  1, // Get most recent
	}

	entries, err := m.mem.Search(ctx, criteria)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entryType, ok := entry.Context["type"].(string); ok && entryType == "working_context" {
			return entry.Content, nil
		}
	}

	return "", nil // No working context found
}

// StoreTopicDiscussion implements AssistantMemory
func (m *MemoryAdapter) StoreTopicDiscussion(ctx context.Context, userID, topic string) error {
	entry := memory.Entry{
		Type:    memory.TypeEpisodic,
		UserID:  userID,
		Content: topic,
		Context: map[string]interface{}{
			"type": "topic_discussion",
		},
		Importance: 0.6, // Topics are moderately important
	}

	return m.mem.Store(ctx, entry)
}

// GetRecentTopics implements AssistantMemory
func (m *MemoryAdapter) GetRecentTopics(ctx context.Context, userID string, limit int) ([]string, error) {
	criteria := memory.SearchCriteria{
		UserID: userID,
		Types:  []memory.Type{memory.TypeEpisodic},
		Limit:  limit,
	}

	entries, err := m.mem.Search(ctx, criteria)
	if err != nil {
		return nil, err
	}

	var topics []string
	for _, entry := range entries {
		if entryType, ok := entry.Context["type"].(string); ok && entryType == "topic_discussion" {
			topics = append(topics, entry.Content)
		}
	}

	return topics, nil
}
