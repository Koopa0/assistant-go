package postgres_test

import (
	"context"
	"fmt"
	"log"

	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
)

// ExamplePoolStats demonstrates how to use the typed PoolStats interface
func ExamplePoolStats() {
	// Create a database client
	cfg := config.DatabaseConfig{
		URL:            "postgres://user:pass@localhost/testdb",
		MaxConnections: 10,
		MinConnections: 2,
	}

	ctx := context.Background()
	client, err := postgres.NewClient(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Get typed pool statistics
	stats := client.GetPoolStats()
	if stats != nil {
		fmt.Printf("Total connections: %d\n", stats.TotalConns)
		fmt.Printf("Idle connections: %d\n", stats.IdleConns)
		fmt.Printf("Active connections: %d\n", stats.AcquiredConns)
		fmt.Printf("Max connections: %d\n", stats.MaxConns)
		fmt.Printf("Total connections created: %d\n", stats.NewConnsCount)
		fmt.Printf("Connections acquired: %d times\n", stats.AcquireCount)
		fmt.Printf("Average acquire time: %dms\n", stats.AcquireDuration.Milliseconds()/stats.AcquireCount)
	}
}

// ExampleMockClient_GetPoolStats demonstrates mock pool stats for testing
func ExampleMockClient_GetPoolStats() {
	// Create a mock client for testing
	mockClient := postgres.NewMockClient(nil)

	// Get mock pool statistics
	stats := mockClient.GetPoolStats()

	// Mock stats are always available
	fmt.Printf("Mock total connections: %d\n", stats.TotalConns)
	fmt.Printf("Mock idle connections: %d\n", stats.IdleConns)
	fmt.Printf("Mock active connections: %d\n", stats.AcquiredConns)

	// Output:
	// Mock total connections: 5
	// Mock idle connections: 3
	// Mock active connections: 2
}
