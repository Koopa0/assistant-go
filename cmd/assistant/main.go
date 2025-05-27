package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/koopa0/assistant-go/internal/assistant"
	"github.com/koopa0/assistant-go/internal/cli"
	"github.com/koopa0/assistant-go/internal/config"
	"github.com/koopa0/assistant-go/internal/observability"
	"github.com/koopa0/assistant-go/internal/server"
	"github.com/koopa0/assistant-go/internal/storage/postgres"
)

const (
	appName    = "GoAssistant"
	appVersion = "0.1.0"
)

func main() {
	// Initialize context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle help and version commands early (before config loading)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("%s %s\n", appName, appVersion)
			return
		case "help", "-h", "--help":
			printUsage()
			return
		}
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup structured logging
	logger := observability.SetupLogging(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	logger.Info("Starting GoAssistant",
		slog.String("version", appVersion),
		slog.String("mode", cfg.Mode))

	// Initialize database connection
	db, err := postgres.NewClient(ctx, cfg.Database)
	if err != nil {
		logger.Error("Failed to initialize database", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	// Run database migrations
	if err := db.Migrate(ctx); err != nil {
		logger.Error("Failed to run database migrations", slog.Any("error", err))
		os.Exit(1)
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
	// Initialize web server
	srv, err := server.New(cfg.Server, assistant, logger)
	if err != nil {
		logger.Error("Failed to initialize server", slog.Any("error", err))
		os.Exit(1)
	}

	// Start server in goroutine
	go func() {
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
	response, err := assistant.ProcessQuery(ctx, query)
	if err != nil {
		logger.Error("Query processing failed", slog.Any("error", err))
		os.Exit(1)
	}

	fmt.Println(response)
}

func printUsage() {
	fmt.Printf(`%s %s - AI-powered development assistant

Usage:
  %s [command] [arguments]

Commands:
  serve, server, web    Start web server (default)
  cli, interactive      Start interactive CLI mode
  ask <question>        Ask a direct question
  version              Show version information
  help                 Show this help message

Examples:
  %s serve                           # Start web server
  %s cli                             # Start interactive CLI
  %s ask "Explain Go's memory model" # Ask direct question

For more information, visit: https://github.com/koopa0/assistant-go
`, appName, appVersion, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
