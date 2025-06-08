package postgres_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/koopa0/assistant-go/internal/tool"
	"github.com/koopa0/assistant-go/internal/tool/postgres"
)

func ExamplePostgresTool_analyzeQuery() {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create the PostgreSQL tool
	pgTool := postgres.NewPostgresTool(logger)

	// Prepare input for query analysis
	input := &tool.ToolInput{
		Parameters: map[string]interface{}{
			"action": "analyze_query",
			"query":  "SELECT * FROM users WHERE email = 'user@example.com'",
		},
	}

	// Execute the tool
	ctx := context.Background()
	result, err := pgTool.Execute(ctx, input)
	if err != nil {
		panic(err)
	}

	// Parse and display the result
	if result.Success && result.Data != nil && result.Data.Output != nil {
		analysis := result.Data.Output
		fmt.Printf("Query Type: %v\n", analysis["query_type"])
		fmt.Printf("Complexity: %v\n", analysis["complexity"])
	}

	// Output:
	// Query Type: SELECT
	// Complexity: simple
}

func ExamplePostgresTool_validateMigration() {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create the PostgreSQL tool
	pgTool := postgres.NewPostgresTool(logger)

	// Prepare input for migration validation
	migration := `
		-- Create users table
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		
		-- Create index on email (concurrently to avoid locking)
		CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
	`

	input := &tool.ToolInput{
		Parameters: map[string]interface{}{
			"action":    "validate_migration",
			"migration": migration,
		},
	}

	// Execute the tool
	ctx := context.Background()
	result, err := pgTool.Execute(ctx, input)
	if err != nil {
		panic(err)
	}

	// Parse and display the result
	if result.Success && result.Data != nil && result.Data.Output != nil {
		validation := result.Data.Output
		fmt.Printf("Is Valid: %v\n", validation["is_valid"])

		// Check best practices
		if bp, ok := validation["best_practices"].(map[string]interface{}); ok {
			fmt.Printf("Has Comments: %v\n", bp["has_comments"])
		}
	}

	// Output:
	// Is Valid: true
	// Has Comments: true
}

func ExamplePostgresTool_withConnection() {
	// This example shows how to use the tool with a database connection
	// Note: This is just an example structure, actual execution requires a database

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	pgTool := postgres.NewPostgresTool(logger)

	// When using with a connection, provide the connection string
	input := &tool.ToolInput{
		Parameters: map[string]interface{}{
			"action":            "check_performance",
			"connection_string": "postgres://user:password@localhost:5432/mydb?sslmode=disable",
		},
	}

	// In a real scenario, this would connect to the database and return performance metrics
	_ = input
	_ = pgTool

	fmt.Println("With a database connection, the tool can:")
	fmt.Println("- Run EXPLAIN ANALYZE on queries")
	fmt.Println("- Check real-time performance metrics")
	fmt.Println("- Analyze actual table and index statistics")
	fmt.Println("- Suggest indexes based on query patterns")

	// Output:
	// With a database connection, the tool can:
	// - Run EXPLAIN ANALYZE on queries
	// - Check real-time performance metrics
	// - Analyze actual table and index statistics
	// - Suggest indexes based on query patterns
}
