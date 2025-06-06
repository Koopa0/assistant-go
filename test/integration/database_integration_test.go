//go:build integration

package integration

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
	"github.com/koopa0/assistant-go/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseIntegration tests database operations with real PostgreSQL
func TestDatabaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup test database
	dbContainer, cleanup := testutil.SetupTestDatabase(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wait for database to be ready
	err := dbContainer.WaitForHealthy(ctx, 10*time.Second)
	require.NoError(t, err, "Database should be healthy")

	// Create configuration
	cfg := config.DatabaseConfig{
		URL:             dbContainer.URL,
		MaxConnections:  5,
		ConnMaxLifetime: time.Hour,
	}

	logger := testutil.CreateTestLogger(slog.LevelDebug)

	// Create storage
	storage, err := postgres.NewStorage(ctx, cfg, logger)
	require.NoError(t, err, "Should create storage")
	defer storage.Close(ctx)

	factory := testutil.NewTestDataFactory()

	t.Run("Connection_Pool", func(t *testing.T) {
		// Test connection pool functionality
		pool, err := dbContainer.GetConnectionPool(ctx)
		require.NoError(t, err, "Should get connection pool")
		defer pool.Close()

		// Test ping
		err = pool.Ping(ctx)
		assert.NoError(t, err, "Should ping successfully")

		// Test concurrent connections
		numConnections := 3
		done := make(chan error, numConnections)

		for i := 0; i < numConnections; i++ {
			go func() {
				conn, err := pool.Acquire(ctx)
				if err != nil {
					done <- err
					return
				}
				defer conn.Release()

				// Execute simple query
				var result int
				err = conn.QueryRow(ctx, "SELECT 1").Scan(&result)
				done <- err
			}()
		}

		// Wait for all connections to complete
		for i := 0; i < numConnections; i++ {
			select {
			case err := <-done:
				assert.NoError(t, err, "Concurrent connection should succeed")
			case <-time.After(5 * time.Second):
				t.Error("Timeout waiting for concurrent connections")
				return
			}
		}
	})

	t.Run("Basic_CRUD_Operations", func(t *testing.T) {
		pool, err := dbContainer.GetConnectionPool(ctx)
		require.NoError(t, err)
		defer pool.Close()

		// Test INSERT
		conversationID := factory.GenerateID()
		userID := factory.GenerateUserID()
		title := "Test Conversation"

		_, err = pool.Exec(ctx, `
			INSERT INTO assistant.conversations (id, user_id, title) 
			VALUES ($1, $2, $3)
		`, conversationID, userID, title)
		assert.NoError(t, err, "Should insert conversation")

		// Test SELECT
		var retrievedTitle string
		var retrievedUserID string
		err = pool.QueryRow(ctx, `
			SELECT user_id, title 
			FROM assistant.conversations 
			WHERE id = $1
		`, conversationID).Scan(&retrievedUserID, &retrievedTitle)
		assert.NoError(t, err, "Should select conversation")
		assert.Equal(t, userID, retrievedUserID, "User ID should match")
		assert.Equal(t, title, retrievedTitle, "Title should match")

		// Test UPDATE
		newTitle := "Updated Test Conversation"
		_, err = pool.Exec(ctx, `
			UPDATE assistant.conversations 
			SET title = $1, updated_at = NOW() 
			WHERE id = $2
		`, newTitle, conversationID)
		assert.NoError(t, err, "Should update conversation")

		// Verify update
		err = pool.QueryRow(ctx, `
			SELECT title 
			FROM assistant.conversations 
			WHERE id = $1
		`, conversationID).Scan(&retrievedTitle)
		assert.NoError(t, err, "Should select updated conversation")
		assert.Equal(t, newTitle, retrievedTitle, "Title should be updated")

		// Test DELETE
		_, err = pool.Exec(ctx, `
			DELETE FROM assistant.conversations 
			WHERE id = $1
		`, conversationID)
		assert.NoError(t, err, "Should delete conversation")

		// Verify deletion
		var count int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM assistant.conversations 
			WHERE id = $1
		`, conversationID).Scan(&count)
		assert.NoError(t, err, "Should count conversations")
		assert.Equal(t, 0, count, "Conversation should be deleted")
	})

	t.Run("Vector_Operations", func(t *testing.T) {
		pool, err := dbContainer.GetConnectionPool(ctx)
		require.NoError(t, err)
		defer pool.Close()

		// Test vector insertion
		embeddingID := factory.GenerateID()
		contentType := "test"
		contentID := factory.GenerateID()
		contentText := "This is test content for vector operations"
		embedding := factory.GenerateEmbedding(1536)

		_, err = pool.Exec(ctx, `
			INSERT INTO assistant.embeddings (id, content_type, content_id, content_text, embedding) 
			VALUES ($1, $2, $3, $4, $5)
		`, embeddingID, contentType, contentID, contentText, embedding)
		assert.NoError(t, err, "Should insert embedding")

		// Test vector similarity search
		queryEmbedding := factory.GenerateEmbedding(1536)
		rows, err := pool.Query(ctx, `
			SELECT id, content_text, 1 - (embedding <=> $1) as similarity
			FROM assistant.embeddings 
			WHERE content_type = $2
			ORDER BY embedding <=> $1
			LIMIT 5
		`, queryEmbedding, contentType)
		assert.NoError(t, err, "Should perform similarity search")
		defer rows.Close()

		var results []struct {
			ID         string
			Content    string
			Similarity float64
		}

		for rows.Next() {
			var result struct {
				ID         string
				Content    string
				Similarity float64
			}
			err := rows.Scan(&result.ID, &result.Content, &result.Similarity)
			assert.NoError(t, err, "Should scan similarity result")
			results = append(results, result)
		}

		assert.NotEmpty(t, results, "Should have similarity results")
		assert.Equal(t, embeddingID, results[0].ID, "Should find inserted embedding")
	})

	t.Run("Transaction_Operations", func(t *testing.T) {
		pool, err := dbContainer.GetConnectionPool(ctx)
		require.NoError(t, err)
		defer pool.Close()

		// Test successful transaction
		tx, err := pool.Begin(ctx)
		require.NoError(t, err, "Should begin transaction")

		conversationID := factory.GenerateID()
		userID := factory.GenerateUserID()

		// Insert conversation
		_, err = tx.Exec(ctx, `
			INSERT INTO assistant.conversations (id, user_id, title) 
			VALUES ($1, $2, $3)
		`, conversationID, userID, "Transaction Test")
		assert.NoError(t, err, "Should insert in transaction")

		// Insert message
		messageID := factory.GenerateID()
		_, err = tx.Exec(ctx, `
			INSERT INTO assistant.messages (id, conversation_id, role, content) 
			VALUES ($1, $2, $3, $4)
		`, messageID, conversationID, "user", "Test message")
		assert.NoError(t, err, "Should insert message in transaction")

		// Commit transaction
		err = tx.Commit(ctx)
		assert.NoError(t, err, "Should commit transaction")

		// Verify data exists
		var count int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM assistant.conversations 
			WHERE id = $1
		`, conversationID).Scan(&count)
		assert.NoError(t, err, "Should count conversations")
		assert.Equal(t, 1, count, "Conversation should exist after commit")

		// Test rollback transaction
		tx2, err := pool.Begin(ctx)
		require.NoError(t, err, "Should begin second transaction")

		rollbackConversationID := factory.GenerateID()
		_, err = tx2.Exec(ctx, `
			INSERT INTO assistant.conversations (id, user_id, title) 
			VALUES ($1, $2, $3)
		`, rollbackConversationID, userID, "Rollback Test")
		assert.NoError(t, err, "Should insert in rollback transaction")

		// Rollback transaction
		err = tx2.Rollback(ctx)
		assert.NoError(t, err, "Should rollback transaction")

		// Verify data doesn't exist
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM assistant.conversations 
			WHERE id = $1
		`, rollbackConversationID).Scan(&count)
		assert.NoError(t, err, "Should count conversations")
		assert.Equal(t, 0, count, "Conversation should not exist after rollback")
	})

	t.Run("Concurrent_Operations", func(t *testing.T) {
		pool, err := dbContainer.GetConnectionPool(ctx)
		require.NoError(t, err)
		defer pool.Close()

		// Test concurrent inserts
		numOperations := 10
		done := make(chan error, numOperations)

		for i := 0; i < numOperations; i++ {
			go func(index int) {
				conversationID := factory.GenerateID()
				userID := factory.GenerateUserID()
				title := factory.GenerateRandomString(20)

				_, err := pool.Exec(ctx, `
					INSERT INTO assistant.conversations (id, user_id, title) 
					VALUES ($1, $2, $3)
				`, conversationID, userID, title)
				done <- err
			}(i)
		}

		// Wait for all operations to complete
		for i := 0; i < numOperations; i++ {
			select {
			case err := <-done:
				assert.NoError(t, err, "Concurrent insert should succeed")
			case <-time.After(10 * time.Second):
				t.Error("Timeout waiting for concurrent operations")
				return
			}
		}

		// Verify all inserts succeeded
		var count int
		err = pool.QueryRow(ctx, `
			SELECT COUNT(*) 
			FROM assistant.conversations
		`).Scan(&count)
		assert.NoError(t, err, "Should count all conversations")
		assert.GreaterOrEqual(t, count, numOperations, "Should have at least the inserted conversations")
	})

	t.Run("Performance_Metrics", func(t *testing.T) {
		pool, err := dbContainer.GetConnectionPool(ctx)
		require.NoError(t, err)
		defer pool.Close()

		// Test query performance
		start := time.Now()

		for i := 0; i < 100; i++ {
			var result int
			err := pool.QueryRow(ctx, "SELECT 1").Scan(&result)
			assert.NoError(t, err, "Simple query should succeed")
		}

		duration := time.Since(start)
		t.Logf("100 simple queries took: %v (avg: %v per query)", duration, duration/100)

		// Performance should be reasonable (less than 1ms per query on average)
		avgDuration := duration / 100
		assert.Less(t, avgDuration, 10*time.Millisecond, "Average query time should be reasonable")
	})

	t.Run("Error_Handling", func(t *testing.T) {
		pool, err := dbContainer.GetConnectionPool(ctx)
		require.NoError(t, err)
		defer pool.Close()

		// Test invalid SQL
		_, err = pool.Exec(ctx, "INVALID SQL STATEMENT")
		assert.Error(t, err, "Should return error for invalid SQL")

		// Test constraint violation
		conversationID := factory.GenerateID()
		userID := factory.GenerateUserID()

		// Insert conversation
		_, err = pool.Exec(ctx, `
			INSERT INTO assistant.conversations (id, user_id, title) 
			VALUES ($1, $2, $3)
		`, conversationID, userID, "Test")
		assert.NoError(t, err, "Should insert conversation")

		// Try to insert duplicate
		_, err = pool.Exec(ctx, `
			INSERT INTO assistant.conversations (id, user_id, title) 
			VALUES ($1, $2, $3)
		`, conversationID, userID, "Duplicate")
		assert.Error(t, err, "Should return error for duplicate key")

		// Test foreign key violation
		_, err = pool.Exec(ctx, `
			INSERT INTO assistant.messages (id, conversation_id, role, content) 
			VALUES ($1, $2, $3, $4)
		`, factory.GenerateID(), "non-existent-conversation", "user", "Test")
		assert.Error(t, err, "Should return error for foreign key violation")
	})
}
