package context

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextEngine_Creation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("valid_logger", func(t *testing.T) {
		engine, err := NewContextEngine(logger)
		require.NoError(t, err)
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.workspace)
		assert.NotNil(t, engine.temporal)
		assert.NotNil(t, engine.semantic)
		assert.NotNil(t, engine.personal)
	})

	t.Run("nil_logger", func(t *testing.T) {
		engine, err := NewContextEngine(nil)
		assert.Error(t, err)
		assert.Nil(t, engine)
		assert.Contains(t, err.Error(), "logger is required")
	})
}

func TestContextUpdate(t *testing.T) {
	update := ContextUpdate{
		Type:      WorkspaceChange,
		Timestamp: time.Now(),
		Source:    "test",
		Data: map[string]interface{}{
			"file": "test.go",
		},
	}

	assert.Equal(t, WorkspaceChange, update.Type)
	assert.Equal(t, "test", update.Source)
	assert.Contains(t, update.Data, "file")
	assert.Equal(t, "test.go", update.Data["file"])
}

func TestRequest(t *testing.T) {
	req := Request{
		ID:        "test-id",
		Query:     "test query",
		Type:      "test",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"user_id": "user123",
		},
	}

	assert.Equal(t, "test-id", req.ID)
	assert.Equal(t, "test query", req.Query)
	assert.Equal(t, "test", req.Type)
	assert.Contains(t, req.Metadata, "user_id")
}

func TestContextualRequest(t *testing.T) {
	original := Request{
		ID:        "test-id",
		Query:     "test query",
		Type:      "test",
		Timestamp: time.Now(),
	}

	contextual := ContextualRequest{
		Original:   original,
		Confidence: 0.8,
	}

	assert.Equal(t, original.ID, contextual.Original.ID)
	assert.Equal(t, 0.8, contextual.Confidence)
}
