package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"github.com/servereye/servereyebot/internal/config"

	// "github.com/servereye/servereyebot/pkg/kafka"
	"github.com/sirupsen/logrus"
)

// Bot represents the Telegram bot instance with dependency injection
type Bot struct {
	// Configuration
	config *config.BotConfig

	// Dependencies (interfaces for better testability)
	logger      Logger
	telegramAPI TelegramAPI
	database    Database
	agentClient AgentClient
	validator   Validator
	metrics     Metrics

	// Direct database access for internal methods
	db     *sql.DB
	keysDB *sql.DB

	// Streams client for new architecture (deprecated, using HTTP now)
	// streamsClient *streams.Client

	// Kafka components disabled - using HTTP instead
	// commandProducer  *kafka.CommandProducer
	// responseConsumer *kafka.ResponseConsumer
	metricsConsumer *KafkaConsumer
	useKafka        bool

	// Context management
	ctx    context.Context
	cancel context.CancelFunc

	// Graceful shutdown
	wg       sync.WaitGroup
	shutdown chan struct{}

	// Ensure single initialization
	telegramStarted sync.Once
}

// BotOptions contains options for creating a new bot instance
type BotOptions struct {
	Config      *config.BotConfig
	Logger      Logger
	TelegramAPI TelegramAPI
	Database    Database
	AgentClient AgentClient
	Validator   Validator
	Metrics     Metrics
}

// New creates a new bot instance with dependency injection
func New(opts BotOptions) (*Bot, error) {
	if opts.Config == nil {
		return nil, NewValidationError("config is required", nil)
	}

	ctx, cancel := context.WithCancel(context.Background())

	bot := &Bot{
		config:      opts.Config,
		logger:      opts.Logger,
		telegramAPI: opts.TelegramAPI,
		database:    opts.Database,
		agentClient: opts.AgentClient,
		validator:   opts.Validator,
		metrics:     opts.Metrics,
		ctx:         ctx,
		cancel:      cancel,
		shutdown:    make(chan struct{}),
	}

	// Set defaults if not provided
	if bot.logger == nil {
		logrusLogger := logrus.New()
		logrusLogger.SetLevel(logrus.InfoLevel)
		bot.logger = NewStructuredLogger(logrusLogger)
	}

	if bot.validator == nil {
		bot.validator = NewInputValidator()
	}

	if bot.metrics == nil {
		bot.metrics = NewInMemoryMetrics()
	}

	return bot, nil
}

// NewFromConfig creates a bot instance from configuration (legacy constructor)
func NewFromConfig(cfg *config.BotConfig, logger *logrus.Logger) (*Bot, error) {
	// Initialize Telegram bot
	tgBot, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, NewTelegramError("failed to create Telegram bot", err)
	}

	logger.WithField("username", tgBot.Self.UserName).Info("Telegram bot authorized")

	// Redis client removed - using Kafka only

	// Initialize database connection
	db, err := sql.Open("postgres", cfg.Database.URL)
	if err != nil {
		return nil, NewDatabaseError("failed to connect to database", err)
	}

	if err := db.Ping(); err != nil {
		return nil, NewDatabaseError("failed to ping database", err)
	}

	logger.Info("Database connection established")

	// Initialize keys database connection
	var keysDB *sql.DB
	if cfg.Database.KeysURL != "" {
		keysDB, err = sql.Open("postgres", cfg.Database.KeysURL)
		if err != nil {
			return nil, NewDatabaseError("failed to connect to keys database", err)
		}

		if err := keysDB.Ping(); err != nil {
			return nil, NewDatabaseError("failed to ping keys database", err)
		}

		// Set connection pooling for keys database (read-only access)
		keysDB.SetMaxOpenConns(5)
		keysDB.SetMaxIdleConns(2)
		keysDB.SetConnMaxLifetime(5 * time.Minute)

		logger.Info("Keys database connection established")
	} else {
		logger.Warn("KEYS_DATABASE_URL not configured, using main database for keys")
		keysDB = db
	}

	// Create a temporary bot instance for adapters
	tempBot := &Bot{}

	// Create adapters
	dbAdapter := NewDatabaseAdapter(db, tempBot)
	agentAdapter := NewAgentClientAdapter(tempBot)

	// Create bot with real implementations
	bot, err := New(BotOptions{
		Config:      cfg,
		Logger:      NewStructuredLogger(logger),
		TelegramAPI: tgBot,
		Database:    dbAdapter,
		AgentClient: agentAdapter,
		Validator:   NewInputValidator(),
		Metrics:     NewInMemoryMetrics(),
	})

	if err != nil {
		return nil, err
	}

	// Update adapter references and set direct DB access
	dbAdapter.bot = bot
	agentAdapter.bot = bot
	bot.db = db
	bot.keysDB = keysDB

	// Initialize Kafka components if enabled
	// var useKafka bool
	// var commandProducer *kafka.CommandProducer
	// var responseConsumer *kafka.ResponseConsumer
	var metricsConsumer *KafkaConsumer

	if false && cfg.Kafka.Enabled && len(cfg.Kafka.Brokers) > 0 {
		// Kafka initialization disabled
		/*
			// Initialize command producer
			producerConfig := kafka.CommandProducerConfig{
				Brokers:      cfg.Kafka.Brokers,
				Topic:        "servereye.commands", // Will be overridden per command with server-specific topic
				Compression:  cfg.Kafka.Compression,
				BatchSize:    1, // Send commands immediately
				BatchTimeout: 10 * time.Millisecond,
			}

			producer, err := kafka.NewCommandProducer(producerConfig, logger)
			if err != nil {
				logger.WithError(err).Error("Failed to create Kafka command producer")
			} else {
				commandProducer = producer
				logger.Info("Kafka command producer initialized")
			}

			// Initialize response consumer
			consumerConfig := kafka.ResponseConsumerConfig{
				Brokers:        cfg.Kafka.Brokers,
				GroupID:        "bot-response-handlers",
				ServerKey:      "bot",                 // Bot receives responses for all servers
				Topic:          "servereye.responses", // Базовый топик, будет использовать wildcard
				MinBytes:       10e3,                  // 10KB
				MaxBytes:       10e6,                  // 10MB
				CommitInterval: time.Second,
			}

			consumer, err := kafka.NewResponseConsumer(consumerConfig, logger)
			if err != nil {
				logger.WithError(err).Error("Failed to create Kafka response consumer")
			} else {
				responseConsumer = consumer
				useKafka = true
				logger.Info("Kafka response consumer initialized")
			}

			// Initialize metrics consumer
			metricsConsumer, err = NewKafkaConsumer(
				cfg.Kafka.Brokers,
				"metrics",
				"bot-metrics-consumers",
				logger,
			)
			if err != nil {
				logger.WithError(err).Error("Failed to create Kafka metrics consumer")
			} else {
				logger.Info("Kafka metrics consumer initialized")
			}
		*/
	}

	// Set Kafka components
	// bot.commandProducer = commandProducer
	// bot.responseConsumer = responseConsumer
	bot.metricsConsumer = metricsConsumer
	bot.useKafka = false // Force disable Kafka

	return bot, nil
}

// Start starts the bot with graceful shutdown handling
func (b *Bot) Start() error {
	b.logger.Info("Starting ServerEye Telegram bot")

	// Start Kafka response consumer if enabled
	if b.useKafka && b.responseConsumer != nil {
		go func() {
			if err := b.responseConsumer.Start(); err != nil {
				b.logger.Error("Failed to start Kafka response consumer", err)
			}
		}()
		b.logger.Info("Starting Kafka response consumer")
		b.logger.Info("Kafka response consumer started")
	}

	// Start Kafka metrics consumer if enabled
	if b.useKafka && b.metricsConsumer != nil {
		go func() {
			if err := b.metricsConsumer.Start(); err != nil {
				b.logger.Error("Failed to start Kafka metrics consumer", err)
			}
		}()
		b.logger.Info("Starting Kafka metrics consumer")
		b.logger.Info("Kafka metrics consumer started")
	}

	// Initialize database schema if database is available
	if b.database != nil {
		if err := b.database.InitSchema(); err != nil {
			return NewDatabaseError("failed to initialize database schema", err)
		}
	}

	// Setup graceful shutdown
	b.setupGracefulShutdown()

	// Setup bot menu commands
	if err := b.setMenuCommands(); err != nil {
		b.logger.Error("Failed to set menu commands", err)
		// Non-critical error, continue
	}

	// Start HTTP server for agent API
	go func() {
		b.logger.Info("About to start HTTP server goroutine...")
		b.startHTTPServer()
	}()

	// Start Telegram updates handler
	if err := b.startTelegramHandler(); err != nil {
		return NewTelegramError("failed to start Telegram handler", err)
	}

	b.logger.Info("ServerEye Telegram bot started successfully")

	// Wait for shutdown signal
	<-b.shutdown

	return nil
}

// startTelegramHandler starts the Telegram updates handler
func (b *Bot) startTelegramHandler() error {
	var initErr error

	b.telegramStarted.Do(func() {
		b.logger.Info("Initializing Telegram handler (sync.Once)")
		if b.telegramAPI == nil {
			initErr = NewTelegramError("Telegram API not initialized", nil)
			return
		}

		// Configure updates
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60

		updates := b.telegramAPI.GetUpdatesChan(u)

		// Debug: log that we got the updates channel
		b.logger.Info("Got updates channel from Telegram API")

		// Start handling updates in a separate goroutine
		b.wg.Add(1)
		go func() {
			defer b.wg.Done()
			b.handleUpdates(updates)
		}()

		b.logger.Info("Telegram updates handler started")
	})

	return initErr
}

// setupGracefulShutdown sets up graceful shutdown handling
func (b *Bot) setupGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		b.logger.Info("Received shutdown signal", StringField("signal", sig.String()))
		if err := b.Stop(); err != nil {
			b.logger.Error("Error during shutdown", err)
		}
	}()
}

// Stop gracefully stops the bot
func (b *Bot) Stop() error {
	b.logger.Info("Initiating graceful shutdown")

	// Cancel context to stop all operations
	b.cancel()

	// Stop receiving Telegram updates
	if b.telegramAPI != nil {
		b.telegramAPI.StopReceivingUpdates()
	}

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		b.logger.Info("All goroutines stopped gracefully")
	case <-time.After(30 * time.Second):
		b.logger.Warn("Timeout waiting for goroutines to stop")
	}

	// Close connections
	if b.useKafka && b.metricsConsumer != nil {
		if err := b.metricsConsumer.Stop(); err != nil {
			b.logger.Error("Error stopping Kafka metrics consumer", err)
		}
	}

	if b.database != nil {
		if err := b.database.Close(); err != nil {
			b.logger.Error("Error closing database connection", err)
		}
	}

	// Close keys database connection if it's separate from main DB
	if b.keysDB != nil && b.keysDB != b.db {
		if err := b.keysDB.Close(); err != nil {
			b.logger.Error("Error closing keys database connection", err)
		}
	}

	// Signal shutdown complete
	close(b.shutdown)

	b.logger.Info("Bot stopped successfully")
	return nil
}

// handleUpdates processes incoming Telegram updates with error handling and metrics
func (b *Bot) handleUpdates(updates tgbotapi.UpdatesChannel) {
	b.logger.Info("Starting Telegram updates processing")

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				b.logger.Info("Updates channel closed, stopping handler")
				return
			}

			// Debug log to confirm update received
			b.logger.Info("Received update",
				IntField("update_id", update.UpdateID),
				StringField("type", func() string {
					if update.Message != nil {
						return "message"
					}
					if update.CallbackQuery != nil {
						return "callback"
					}
					return "unknown"
				}()))

			// Process update with timeout and error handling
			ctx, cancel := context.WithTimeout(b.ctx, 30*time.Second)
			b.processUpdate(ctx, update)
			cancel()

		case <-b.ctx.Done():
			b.logger.Info("Context cancelled, stopping updates handler")
			return
		}
	}
}

// processUpdate processes a single update with error recovery
func (b *Bot) processUpdate(ctx context.Context, update tgbotapi.Update) {
	// Recover from panics to prevent bot crash
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			b.logger.Error("Panic recovered in update processing",
				fmt.Errorf("panic: %v", r),
				StringField("update_id", fmt.Sprintf("%d", update.UpdateID)),
				StringField("stack", string(stack)))

			if b.metrics != nil {
				b.metrics.IncrementError("PANIC_RECOVERED")
			}
		}
	}()

	// Process different update types
	switch {
	case update.Message != nil:
		b.processMessage(ctx, update.Message)
	case update.CallbackQuery != nil:
		b.processCallbackQuery(ctx, update.CallbackQuery)
	default:
		b.logger.Debug("Received update without message or callback query",
			IntField("update_id", update.UpdateID))
	}
}

// processMessage processes a message update
func (b *Bot) processMessage(ctx context.Context, message *tgbotapi.Message) {
	start := time.Now()

	// Log message details
	b.logger.Info("Processing message",
		Int64Field("user_id", message.From.ID),
		StringField("username", message.From.UserName),
		StringField("text", message.Text),
		IntField("message_id", message.MessageID))

	// Validate and sanitize input
	if b.validator != nil && message.Text != "" {
		if validator, ok := b.validator.(*InputValidator); ok {
			message.Text = validator.SanitizeInput(message.Text)
		}
	}

	// Handle message with error handling
	err := b.handleMessage(message)

	// Record metrics
	if b.metrics != nil {
		duration := time.Since(start).Seconds()
		b.metrics.RecordLatency("message_processing", duration)

		if err != nil {
			var botErr *BotError
			if errors.As(err, &botErr) {
				b.metrics.IncrementError(botErr.Code)
			} else {
				b.metrics.IncrementError("UNKNOWN_ERROR")
			}
		}
	}

	if err != nil {
		b.logger.Error("Error processing message", err,
			Int64Field("user_id", message.From.ID),
			StringField("text", message.Text))
	}
}

// processCallbackQuery processes a callback query update
func (b *Bot) processCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) {
	start := time.Now()

	// Log callback details
	b.logger.Info("Processing callback query",
		Int64Field("user_id", query.From.ID),
		StringField("username", query.From.UserName),
		StringField("data", query.Data),
		StringField("query_id", query.ID))

	// Handle callback with error handling
	err := b.handleCallbackQuery(query)

	// Record metrics
	if b.metrics != nil {
		duration := time.Since(start).Seconds()
		b.metrics.RecordLatency("callback_processing", duration)

		if err != nil {
			var botErr *BotError
			if errors.As(err, &botErr) {
				b.metrics.IncrementError(botErr.Code)
			} else {
				b.metrics.IncrementError("UNKNOWN_ERROR")
			}
		}
	}

	if err != nil {
		b.logger.Error("Error processing callback query", err,
			Int64Field("user_id", query.From.ID),
			StringField("data", query.Data))
	}
}

// setMenuCommands sets up the bot menu commands
func (b *Bot) setMenuCommands() error {
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Start bot and show welcome message"},
		{Command: "help", Description: "Show available commands"},
		{Command: "temp", Description: "Get CPU temperature"},
		{Command: "memory", Description: "Get memory usage"},
		{Command: "disk", Description: "Get disk usage"},
		{Command: "uptime", Description: "Get system uptime"},
		{Command: "processes", Description: "List running processes"},
		{Command: "containers", Description: "Manage Docker containers"},
		{Command: "update", Description: "Update agent to latest version"},
		{Command: "servers", Description: "List your servers"},
		{Command: "status", Description: "Get server status"},
	}

	cfg := tgbotapi.NewSetMyCommands(commands...)
	_, err := b.telegramAPI.Request(cfg)
	if err != nil {
		return err
	}

	b.logger.Info("Bot menu commands set successfully")
	return nil
}

// handleMessage processes a single message
