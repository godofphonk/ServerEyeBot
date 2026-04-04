package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
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

	// Auto-connect to Docker network in local development
	if os.Getenv("AUTO_CONNECT_NETWORK") == "true" && cfg.App.Environment == "development" {
		log.Info("Auto-connecting to Docker network for local development")
		if err := autoConnectToNetwork(log); err != nil {
			log.Warn("Failed to auto-connect to network", "error", err)
		}
	}

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

// autoConnectToNetwork attempts to connect this container to the servereye-network network
func autoConnectToNetwork(log logger.Logger) error {
	// Get container ID from /proc/self/cgroup
	containerID, err := getContainerID()
	if err != nil {
		return fmt.Errorf("failed to get container ID: %w", err)
	}

	// Try to connect to servereye-network network
	cmd := exec.Command("docker", "network", "connect", "servereye-network", containerID)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Check if it's already connected or network doesn't exist (not an error in local dev)
		if len(output) > 0 {
			outputStr := string(output)
			if contains(outputStr, "already connected") || contains(outputStr, "No such network") {
				log.Info("Network connection status", "status", outputStr)
				return nil
			}
		}
		return fmt.Errorf("failed to connect to network: %w, output: %s", err, string(output))
	}

	log.Info("Successfully connected to servereye-network network", "container", containerID)
	return nil
}

// getContainerID extracts container ID from /proc/self/cgroup
func getContainerID() (string, error) {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", err
	}

	// Look for docker container ID in cgroup file
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.Contains(line, "docker") {
			parts := strings.Split(line, "/")
			for i, part := range parts {
				if part == "docker" && i+1 < len(parts) {
					containerID := parts[i+1]
					// Container ID might be long, take first 12 chars
					if len(containerID) > 12 {
						containerID = containerID[:12]
					}
					return containerID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("docker container ID not found in cgroup")
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
