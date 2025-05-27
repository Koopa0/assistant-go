package memory

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
	"github.com/koopa0/assistant-go/internal/testutil"
)

func TestMemoryManager(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	// Create nil database client for testing (memory components will handle gracefully)
	var dbClient *postgres.SQLCClient = nil

	manager := NewMemoryManager(dbClient, config, logger)

	// Test manager creation
	if manager == nil {
		t.Fatal("Failed to create memory manager")
	}

	ctx := context.Background()

	// Test storing different types of memory entries
	testEntries := []*MemoryEntry{
		{
			ID:         "test_short_1",
			Type:       MemoryTypeShortTerm,
			UserID:     "user123",
			SessionID:  "session456",
			Content:    "This is a short-term memory entry",
			Importance: 0.8,
			CreatedAt:  time.Now(),
			Metadata:   map[string]interface{}{"test": true},
		},
		{
			ID:         "test_long_1",
			Type:       MemoryTypeLongTerm,
			UserID:     "user123",
			Content:    "This is a long-term memory entry",
			Importance: 0.9,
			CreatedAt:  time.Now(),
			Metadata:   map[string]interface{}{"category": "important"},
		},
		{
			ID:      "test_tool_1",
			Type:    MemoryTypeTool,
			UserID:  "user123",
			Content: "Tool execution result",
			Context: map[string]interface{}{
				"tool_name":      "test_tool",
				"input_hash":     "abc123",
				"input":          map[string]interface{}{"param": "value"},
				"output":         "tool output",
				"execution_time": time.Millisecond * 100,
				"success":        true,
			},
			Importance: 0.5,
			CreatedAt:  time.Now(),
		},
		{
			ID:      "test_personal_1",
			Type:    MemoryTypePersonalization,
			UserID:  "user123",
			Content: "User preference setting",
			Context: map[string]interface{}{
				"category": "ui",
				"key":      "theme",
				"value":    "dark",
				"type":     "string",
			},
			Importance: 0.7,
			CreatedAt:  time.Now(),
		},
	}

	// Store all test entries
	for _, entry := range testEntries {
		err := manager.Store(ctx, entry)
		if err != nil {
			t.Errorf("Failed to store memory entry %s: %v", entry.ID, err)
		}
	}

	// Test retrieving memories
	query := &MemoryQuery{
		UserID: "user123",
		Limit:  10,
	}

	results, err := manager.Retrieve(ctx, query)
	if err != nil {
		t.Fatalf("Failed to retrieve memories: %v", err)
	}

	if len(results) == 0 {
		t.Error("Should have retrieved some memory entries")
	}

	// Test retrieving specific memory types
	shortTermQuery := &MemoryQuery{
		UserID: "user123",
		Types:  []MemoryType{MemoryTypeShortTerm},
		Limit:  5,
	}

	shortTermResults, err := manager.Retrieve(ctx, shortTermQuery)
	if err != nil {
		t.Fatalf("Failed to retrieve short-term memories: %v", err)
	}

	for _, result := range shortTermResults {
		if result.Entry.Type != MemoryTypeShortTerm {
			t.Errorf("Expected short-term memory, got %s", result.Entry.Type)
		}
	}

	// Test getting statistics
	stats, err := manager.GetStats(ctx, "user123")
	if err != nil {
		t.Fatalf("Failed to get memory stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if stats.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", stats.UserID)
	}

	// Test cleanup
	err = manager.Cleanup(ctx)
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}

	// Test clearing memories
	err = manager.Clear(ctx, "user123", []MemoryType{MemoryTypeShortTerm}, nil)
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}
}

func TestShortTermMemory(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    3, // Small size for testing
	}

	stm := NewShortTermMemory(config, logger)

	// Test memory creation
	if stm == nil {
		t.Fatal("Failed to create short-term memory")
	}

	ctx := context.Background()

	// Test storing entries
	entries := []*MemoryEntry{
		{
			Type:       MemoryTypeShortTerm,
			UserID:     "user123",
			SessionID:  "session456",
			Content:    "First memory entry",
			Importance: 0.8,
			CreatedAt:  time.Now(),
		},
		{
			Type:       MemoryTypeShortTerm,
			UserID:     "user123",
			SessionID:  "session456",
			Content:    "Second memory entry",
			Importance: 0.7,
			CreatedAt:  time.Now().Add(time.Minute),
		},
		{
			Type:       MemoryTypeShortTerm,
			UserID:     "user123",
			SessionID:  "session456",
			Content:    "Third memory entry",
			Importance: 0.9,
			CreatedAt:  time.Now().Add(time.Minute * 2),
		},
		{
			Type:       MemoryTypeShortTerm,
			UserID:     "user123",
			SessionID:  "session456",
			Content:    "Fourth memory entry (should trigger size limit)",
			Importance: 0.6,
			CreatedAt:  time.Now().Add(time.Minute * 3),
		},
	}

	for _, entry := range entries {
		err := stm.Store(ctx, entry)
		if err != nil {
			t.Errorf("Failed to store entry: %v", err)
		}
	}

	// Test search
	query := &MemoryQuery{
		UserID:    "user123",
		SessionID: "session456",
		Content:   "memory",
		Limit:     10,
	}

	results, err := stm.Search(ctx, query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should have at most 3 entries due to size limit
	if len(results) > 3 {
		t.Errorf("Expected at most 3 results due to size limit, got %d", len(results))
	}

	// Test stats
	stats, err := stm.GetStats(ctx, "user123")
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.EntryCount > 3 {
		t.Errorf("Expected at most 3 entries in stats, got %d", stats.EntryCount)
	}

	// Test cleanup (expired entries)
	err = stm.Cleanup(ctx)
	if err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
}

func TestToolMemory(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	tm := NewToolMemory(config, logger)

	// Test memory creation
	if tm == nil {
		t.Fatal("Failed to create tool memory")
	}

	ctx := context.Background()

	// Test caching tool result
	err := tm.CacheToolResult(
		ctx,
		"user123",
		"test_tool",
		map[string]interface{}{"param1": "value1", "param2": 42},
		"tool execution result",
		time.Millisecond*150,
		true,
		"",
	)
	if err != nil {
		t.Fatalf("Failed to cache tool result: %v", err)
	}

	// Test retrieving cached result
	inputHash := tm.hashInput(map[string]interface{}{"param1": "value1", "param2": 42})
	cachedResult, found := tm.GetCachedResult(ctx, "user123", "test_tool", inputHash)
	if !found {
		t.Error("Should have found cached result")
	}

	if cachedResult == nil {
		t.Fatal("Cached result should not be nil")
	}

	if cachedResult.ToolName != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", cachedResult.ToolName)
	}

	if cachedResult.Success != true {
		t.Error("Expected successful tool execution")
	}

	// Test hit count increment
	initialHitCount := cachedResult.HitCount
	cachedResult2, found2 := tm.GetCachedResult(ctx, "user123", "test_tool", inputHash)
	if !found2 {
		t.Error("Should have found cached result on second access")
	}

	if cachedResult2.HitCount != initialHitCount+1 {
		t.Errorf("Expected hit count to increment from %d to %d, got %d",
			initialHitCount, initialHitCount+1, cachedResult2.HitCount)
	}

	// Test search
	query := &MemoryQuery{
		UserID:  "user123",
		Content: "test_tool",
		Limit:   5,
	}

	results, err := tm.Search(ctx, query)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Should have found search results")
	}

	// Test stats
	stats, err := tm.GetStats(ctx, "user123")
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.EntryCount == 0 {
		t.Error("Stats should show at least one entry")
	}
}

func TestMemoryQuery(t *testing.T) {
	// Test memory query validation and filtering
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	stm := NewShortTermMemory(config, logger)
	ctx := context.Background()

	// Store test entries with different properties
	now := time.Now()
	entries := []*MemoryEntry{
		{
			Type:       MemoryTypeShortTerm,
			UserID:     "user123",
			SessionID:  "session1",
			Content:    "High importance entry",
			Importance: 0.9,
			CreatedAt:  now,
		},
		{
			Type:       MemoryTypeShortTerm,
			UserID:     "user123",
			SessionID:  "session2",
			Content:    "Low importance entry",
			Importance: 0.3,
			CreatedAt:  now.Add(-time.Hour),
		},
		{
			Type:       MemoryTypeShortTerm,
			UserID:     "user456",
			SessionID:  "session1",
			Content:    "Different user entry",
			Importance: 0.8,
			CreatedAt:  now,
		},
	}

	for _, entry := range entries {
		err := stm.Store(ctx, entry)
		if err != nil {
			t.Errorf("Failed to store entry: %v", err)
		}
	}

	// Test filtering by session ID
	sessionQuery := &MemoryQuery{
		UserID:    "user123",
		SessionID: "session1",
		Limit:     10,
	}

	sessionResults, err := stm.Search(ctx, sessionQuery)
	if err != nil {
		t.Fatalf("Session search failed: %v", err)
	}

	for _, result := range sessionResults {
		if result.Entry.SessionID != "session1" {
			t.Errorf("Expected session ID 'session1', got '%s'", result.Entry.SessionID)
		}
	}

	// Test filtering by importance
	importanceQuery := &MemoryQuery{
		UserID:        "user123",
		MinImportance: 0.5,
		Limit:         10,
	}

	importanceResults, err := stm.Search(ctx, importanceQuery)
	if err != nil {
		t.Fatalf("Importance search failed: %v", err)
	}

	for _, result := range importanceResults {
		if result.Entry.Importance < 0.5 {
			t.Errorf("Expected importance >= 0.5, got %f", result.Entry.Importance)
		}
	}

	// Test filtering by time range
	timeQuery := &MemoryQuery{
		UserID: "user123",
		TimeRange: &TimeRange{
			Start: now.Add(-time.Minute),
			End:   now.Add(time.Minute),
		},
		Limit: 10,
	}

	timeResults, err := stm.Search(ctx, timeQuery)
	if err != nil {
		t.Fatalf("Time range search failed: %v", err)
	}

	for _, result := range timeResults {
		if result.Entry.CreatedAt.Before(timeQuery.TimeRange.Start) ||
			result.Entry.CreatedAt.After(timeQuery.TimeRange.End) {
			t.Errorf("Entry created at %v is outside time range %v - %v",
				result.Entry.CreatedAt, timeQuery.TimeRange.Start, timeQuery.TimeRange.End)
		}
	}

	// Test content search
	contentQuery := &MemoryQuery{
		UserID:  "user123",
		Content: "importance",
		Limit:   10,
	}

	contentResults, err := stm.Search(ctx, contentQuery)
	if err != nil {
		t.Fatalf("Content search failed: %v", err)
	}

	for _, result := range contentResults {
		if !containsIgnoreCase(result.Entry.Content, "importance") {
			t.Errorf("Entry content '%s' does not contain 'importance'", result.Entry.Content)
		}
	}
}

func TestMemoryEntryValidation(t *testing.T) {
	logger := testutil.NewTestLogger()
	config := config.LangChain{
		MaxIterations: 5,
		EnableMemory:  true,
		MemorySize:    10,
	}

	dbClient := &postgres.SQLCClient{}
	manager := NewMemoryManager(dbClient, config, logger)

	ctx := context.Background()

	// Test storing entry with unknown memory type
	invalidEntry := &MemoryEntry{
		ID:         "invalid_1",
		Type:       MemoryType("unknown"),
		UserID:     "user123",
		Content:    "Invalid memory type",
		Importance: 0.5,
		CreatedAt:  time.Now(),
	}

	err := manager.Store(ctx, invalidEntry)
	if err == nil {
		t.Error("Expected error for unknown memory type")
	}

	// Test valid entry
	validEntry := &MemoryEntry{
		ID:         "valid_1",
		Type:       MemoryTypeShortTerm,
		UserID:     "user123",
		Content:    "Valid memory entry",
		Importance: 0.5,
		CreatedAt:  time.Now(),
	}

	err = manager.Store(ctx, validEntry)
	if err != nil {
		t.Errorf("Should not error for valid entry: %v", err)
	}
}

// Helper function for case-insensitive string contains check
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			(len(s) > 0 && len(substr) > 0 &&
				strings.ToLower(s) == strings.ToLower(s) &&
				strings.Contains(strings.ToLower(s), strings.ToLower(substr))))
}
