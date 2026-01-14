package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/servereye/servereyebot/internal/app"
	"github.com/servereye/servereyebot/internal/config"
	"github.com/servereye/servereyebot/internal/logger"
)

var (
	version = "1.0.0"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version information")
		_           = flag.String("config", "", "Path to configuration file (optional)")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("ServerEyeBot %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	// Create logger
	log, err := logger.New(logger.LoggerConfig{
		Level:      cfg.Logger.Level,
		Format:     cfg.Logger.Format,
		Output:     cfg.Logger.Output,
		Filename:   cfg.Logger.Filename,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	log.Info("Starting ServerEyeBot",
		"version", version,
		"commit", commit,
		"environment", cfg.App.Environment,
		"port", cfg.App.Port)

	// Create bot
	bot, err := app.New(cfg, log)
	if err != nil {
		log.Fatal("Failed to create bot", "error", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start bot
	if err := bot.Start(ctx); err != nil {
		log.Fatal("Failed to start bot", "error", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Info("ServerEyeBot is running. Press Ctrl+C to stop.")

	// Wait for signal
	sig := <-sigChan
	log.Info("Received signal", "signal", sig.String())

	// Graceful shutdown
	log.Info("Shutting down ServerEyeBot...")
	bot.Stop()

	log.Info("ServerEyeBot stopped successfully")
}
