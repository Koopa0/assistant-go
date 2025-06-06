package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // Enable pprof endpoints
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/cli"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/platform/observability"
	"github.com/koopa0/assistant-go/internal/platform/server"
	"github.com/koopa0/assistant-go/internal/platform/storage/postgres"
)

const (
	appName = "Assistant"
)

func main() {
	// Initialize context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle help and version commands early (before config loading)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			cli.PrintVersion(appName)
			return
		case "help", "-h", "--help":
			printUsage()
			return
		}
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load environment variables from .env file if it exists
	// This allows flexible configuration: .env for simple setups, YAML for complex ones
	if err := godotenv.Load(); err != nil {
		// .env file is optional, so we only warn if it exists but can't be loaded
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: Error loading .env file: %v\n", err)
		}
	}

	// Allow environment variable to override config file preference
	if configMode := os.Getenv("CONFIG_MODE"); configMode == "env-only" {
		// When CONFIG_MODE=env-only, skip YAML config and rely on environment variables only
		os.Setenv("CONFIG_FILE", "")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Determine if we're in quiet mode for CLI/ask commands
	isQuietMode := len(os.Args) > 1 && (os.Args[1] == "cli" || os.Args[1] == "interactive" || os.Args[1] == "ask")

	// Setup structured logging with mode-specific configuration
	logLevel := cfg.LogLevel
	var logger *slog.Logger
	if isQuietMode {
		// For CLI/ask modes, redirect logs to /dev/null to keep output clean
		nullFile, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open /dev/null: %v\n", err)
			os.Exit(1)
		}
		defer nullFile.Close()
		logger = observability.SetupLoggingWithWriter(nullFile, "error", cfg.LogFormat)
	} else {
		logger = observability.SetupLogging(logLevel, cfg.LogFormat)
	}
	slog.SetDefault(logger)

	// Only log startup info for server mode
	if !isQuietMode {
		logger.Info("Starting Assistant",
			slog.String("version", cli.GetVersion()),
			slog.String("mode", cfg.Mode))
	}

	// Handle migration commands early (before full database init)
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s migrate <up|down|status>\n", os.Args[0])
			os.Exit(1)
		}
		runMigrate(ctx, cfg, logger, os.Args[2])
		return
	}

	// Initialize database connection
	var db postgres.DB

	// Check if we're in test/demo mode or quiet mode
	if os.Getenv("ASSISTANT_DEMO_MODE") == "true" || cfg.Database.URL == "" || isQuietMode {
		if !isQuietMode {
			logger.Info("Running in demo mode without database")
		}
		db = postgres.NewMockClient(logger)
	} else {
		client, err := postgres.NewClient(ctx, cfg.Database)
		if err != nil {
			logger.Error("Failed to initialize database", slog.Any("error", err))
			os.Exit(1)
		}
		db = client
		defer client.Close()

		// Run database migrations
		if err := client.Migrate(ctx); err != nil {
			logger.Error("Failed to run database migrations", slog.Any("error", err))
			os.Exit(1)
		}
	}

	// Initialize performance profiling manager (golang_guide.md recommendation)
	profileManager := observability.NewProfileManager(logger)
	if cfg.Mode == "production" || os.Getenv("ENABLE_PROFILING") == "true" {
		profileManager.EnableProfiling(time.Minute * 10) // Profile every 10 minutes in production
		profileManager.StartPeriodicProfiling(ctx)
	}

	// Initialize assistant core
	assistantCore, err := assistant.New(ctx, cfg, db, logger)
	if err != nil {
		logger.Error("Failed to initialize assistant", slog.Any("error", err))
		os.Exit(1)
	}

	// Determine run mode based on command line arguments
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve", "server", "web":
			runWebServer(ctx, cfg, assistantCore, logger, sigChan)
		case "cli", "interactive":
			runCLI(ctx, cfg, assistantCore, logger)
		case "ask":
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: %s ask <question>\n", os.Args[0])
				os.Exit(1)
			}
			runDirectQuery(ctx, assistantCore, os.Args[2], logger)

		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
	} else {
		// Default to web server mode
		runWebServer(ctx, cfg, assistantCore, logger, sigChan)
	}
}

func runWebServer(ctx context.Context, cfg *config.Config, assistant *assistant.Assistant, logger *slog.Logger, sigChan chan os.Signal) {
	// Start pprof server for performance profiling (golang_guide.md recommendation)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("pprof server panicked", slog.Any("panic", r))
			}
		}()

		logger.Info("Starting pprof server", slog.String("address", "localhost:6060"))
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			logger.Warn("pprof server failed", slog.Any("error", err))
		}
	}()

	// Initialize metrics
	metrics, err := observability.NewMetrics("assistant")
	if err != nil {
		logger.Error("Failed to initialize metrics", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize web server
	srv, err := server.New(cfg.Server, assistant, logger, metrics)
	if err != nil {
		logger.Error("Failed to initialize server", slog.Any("error", err))
		os.Exit(1)
	}

	// Start server in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("HTTP server goroutine panicked",
					slog.Any("panic", r),
					slog.String("address", cfg.Server.Address))
				os.Exit(1)
			}
		}()

		logger.Info("Starting web server", slog.String("address", cfg.Server.Address))
		if err := srv.Start(ctx); err != nil {
			logger.Error("Server failed to start", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Received shutdown signal, gracefully shutting down...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("Server shutdown complete")
}

func runCLI(ctx context.Context, cfg *config.Config, assistant *assistant.Assistant, logger *slog.Logger) {
	cliApp, err := cli.New(cfg.CLI, assistant, logger)
	if err != nil {
		logger.Error("Failed to initialize CLI", slog.Any("error", err))
		os.Exit(1)
	}

	if err := cliApp.Run(ctx); err != nil {
		logger.Error("CLI execution failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func runDirectQuery(ctx context.Context, assistant *assistant.Assistant, query string, logger *slog.Logger) {
	// Use streaming for direct queries
	streamResp, err := assistant.ProcessQueryStream(ctx, query)
	if err != nil {
		logger.Error("Query processing failed", slog.Any("error", err))
		os.Exit(1)
	}

	// Process the stream
	for {
		select {
		case text, ok := <-streamResp.TextChan:
			if !ok {
				fmt.Println() // Final newline
				return
			}
			fmt.Print(text)

		case <-streamResp.EventChan:
			// Ignore events for simple output

		case err := <-streamResp.ErrorChan:
			logger.Error("Streaming error", slog.Any("error", err))
			os.Exit(1)

		case <-streamResp.Done:
			fmt.Println() // Final newline
			return
		}
	}
}

func runMigrate(ctx context.Context, cfg *config.Config, logger *slog.Logger, command string) {
	// Initialize database connection for migration
	client, err := postgres.NewClient(ctx, cfg.Database)
	if err != nil {
		logger.Error("Failed to initialize database for migration", slog.Any("error", err))
		os.Exit(1)
	}
	defer client.Close()

	migrator := postgres.NewMigrator(client, cfg.Database.MigrationsPath)

	switch command {
	case "up":
		logger.Info("Running database migrations...")
		if err := migrator.Up(ctx); err != nil {
			logger.Error("Migration failed", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Migrations completed successfully")
	case "down":
		logger.Info("Rolling back last migration...")
		if err := migrator.Down(ctx); err != nil {
			logger.Error("Migration rollback failed", slog.Any("error", err))
			os.Exit(1)
		}
		logger.Info("Migration rolled back successfully")
	case "status":
		logger.Info("Checking migration status...")
		status, err := migrator.Status(ctx)
		if err != nil {
			logger.Error("Failed to get migration status", slog.Any("error", err))
			os.Exit(1)
		}

		fmt.Println("Migration Status:")
		fmt.Println("================")
		for _, migration := range status {
			status := "Pending"
			appliedAt := "Not applied"
			if migration.AppliedAt != nil {
				status = "Applied"
				appliedAt = migration.AppliedAt.Format("2006-01-02 15:04:05")
			}
			fmt.Printf("Version %d: %s [%s] - %s\n",
				migration.Version, migration.Name, status, appliedAt)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown migration command: %s\n", command)
		fmt.Fprintf(os.Stderr, "Available commands: up, down, status\n")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`%s %s - AI-powered development assistant

Usage:
  %s [command] [arguments]

Commands:
  serve, server         Start API server (default)
  cli, interactive      Start interactive CLI mode
  ask <question>        Ask a direct question
  migrate <up|down|status>  Database migration commands
  version              Show version information
  help                 Show this help message

Examples:
  %s serve                           # Start API server
  %s cli                             # Start interactive CLI
  %s ask "Explain Go's memory model" # Ask direct question

For more information, visit: https://github.com/koopa0/assistant
`, appName, cli.GetVersion(), os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
