package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/servereye/servereyebot/internal/bot"
	"github.com/servereye/servereyebot/internal/config"
	"github.com/servereye/servereyebot/internal/version"
	"github.com/sirupsen/logrus"
)

const (
	defaultConfigPath = "/etc/servereye/bot-config.yaml"
	defaultLogLevel   = "info"
)

func main() {
	var (
		configPath  = flag.String("config", defaultConfigPath, "Path to configuration file")
		logLevel    = flag.String("log-level", defaultLogLevel, "Log level (debug, info, warn, error)")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("ServerEye Bot %s\n", version.GetFullVersion())
		return
	}

	// Setup logger
	logger := setupLogger(*logLevel)

	// Load configuration
	cfg, err := loadConfigFromEnv()
	if err != nil {
		// Try to load from file if env vars not available
		cfg, err = config.LoadBotConfig(*configPath)
		if err != nil {
			logger.WithError(err).Fatal("Failed to load configuration")
		}
	}

	// Create and start bot
	botInstance, err := bot.NewFromConfig(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create bot")
	}

	if err := botInstance.Start(); err != nil {
		logger.WithError(err).Fatal("Failed to start bot")
	}

	logger.Info("ServerEye Bot started successfully")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down bot...")
	if err := botInstance.Stop(); err != nil {
		logger.WithError(err).Error("Error during shutdown")
	}
}

// setupLogger configures and returns a logger instance
func setupLogger(level string) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Set formatter for production (JSON) or development (text)
	if os.Getenv("ENVIRONMENT") == "production" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	}

	return logger
}

// loadConfigFromEnv loads configuration from environment variables
func loadConfigFromEnv() (*config.BotConfig, error) {
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	databaseURL := os.Getenv("DATABASE_URL")
	keysDatabaseURL := os.Getenv("KEYS_DATABASE_URL")

	// Kafka configuration
	kafkaEnabledStr := os.Getenv("KAFKA_ENABLED")
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	kafkaTopicPrefix := os.Getenv("KAFKA_TOPIC_PREFIX")
	kafkaCompression := os.Getenv("KAFKA_COMPRESSION")
	kafkaMaxAttemptsStr := os.Getenv("KAFKA_MAX_ATTEMPTS")
	kafkaBatchSizeStr := os.Getenv("KAFKA_BATCH_SIZE")
	kafkaRequiredAcksStr := os.Getenv("KAFKA_REQUIRED_ACKS")

	// Parse Kafka configuration
	var kafkaConfig config.KafkaConfig
	if kafkaEnabledStr == "true" || kafkaEnabledStr == "1" {
		kafkaConfig.Enabled = true

		// Set defaults for optional fields
		if kafkaBrokers == "" {
			kafkaConfig.Brokers = []string{"localhost:9092"}
		} else {
			// Split comma-separated brokers
			kafkaConfig.Brokers = strings.Split(kafkaBrokers, ",")
		}

		if kafkaTopicPrefix == "" {
			kafkaConfig.TopicPrefix = "metrics"
		} else {
			kafkaConfig.TopicPrefix = kafkaTopicPrefix
		}

		if kafkaCompression == "" {
			kafkaConfig.Compression = "snappy"
		} else {
			kafkaConfig.Compression = kafkaCompression
		}

		if kafkaMaxAttemptsStr != "" {
			if maxAttempts, err := strconv.Atoi(kafkaMaxAttemptsStr); err == nil {
				kafkaConfig.MaxAttempts = maxAttempts
			} else {
				kafkaConfig.MaxAttempts = 3
			}
		} else {
			kafkaConfig.MaxAttempts = 3
		}

		if kafkaBatchSizeStr != "" {
			if batchSize, err := strconv.Atoi(kafkaBatchSizeStr); err == nil {
				kafkaConfig.BatchSize = batchSize
			} else {
				kafkaConfig.BatchSize = 100
			}
		} else {
			kafkaConfig.BatchSize = 100
		}

		if kafkaRequiredAcksStr != "" {
			if requiredAcks, err := strconv.Atoi(kafkaRequiredAcksStr); err == nil {
				kafkaConfig.RequiredAcks = requiredAcks
			} else {
				kafkaConfig.RequiredAcks = 1
			}
		} else {
			kafkaConfig.RequiredAcks = 1
		}
	}

	if telegramToken == "" || databaseURL == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	return &config.BotConfig{
		Telegram: config.TelegramConfig{
			Token: telegramToken,
		},
		Database: config.DatabaseConfig{
			URL:     databaseURL,
			KeysURL: keysDatabaseURL,
		},
		Kafka: kafkaConfig,
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}, nil
}
