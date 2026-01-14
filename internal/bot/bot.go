package bot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/servereye/servereyebot/internal/config"
	"github.com/servereye/servereyebot/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// Bot represents the Telegram bot instance with dependency injection
type Bot struct {
	// Configuration
	config *config.BotConfig

	// Dependencies (interfaces for better testability)
	logger      Logger
	telegramAPI TelegramAPI
	validator   Validator
	metrics     Metrics

	// System monitors
	cpuMetrics    *metrics.CPUMetrics
	systemMonitor *metrics.SystemMonitor

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
		config:        opts.Config,
		logger:        opts.Logger,
		telegramAPI:   opts.TelegramAPI,
		validator:     opts.Validator,
		metrics:       opts.Metrics,
		cpuMetrics:    metrics.NewCPUMetrics(),
		systemMonitor: metrics.NewSystemMonitor(logrus.New()),
		ctx:           ctx,
		cancel:        cancel,
		shutdown:      make(chan struct{}),
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

	// Create bot with real implementations
	bot, err := New(BotOptions{
		Config:      cfg,
		Logger:      NewStructuredLogger(logger),
		TelegramAPI: tgBot,
		Validator:   NewInputValidator(),
		Metrics:     NewInMemoryMetrics(),
	})

	if err != nil {
		return nil, err
	}

	return bot, nil
}

// Start starts the bot with graceful shutdown handling
func (b *Bot) Start() error {
	b.logger.Info("Starting ServerEye Telegram bot")

	// Setup graceful shutdown
	b.setupGracefulShutdown()

	// Setup bot menu commands
	if err := b.setMenuCommands(); err != nil {
		b.logger.Error("Failed to set menu commands", err)
		// Non-critical error, continue
	}

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

	b.logger.Info("ServerEye Telegram bot stopped successfully")
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
		b.handleMessage(ctx, update.Message)
	case update.CallbackQuery != nil:
		b.handleCallback(ctx, update.CallbackQuery)
	default:
		b.logger.Debug("Received update without message or callback query",
			IntField("update_id", update.UpdateID))
	}
}

// handleMessage handles message processing with command routing
func (b *Bot) handleMessage(ctx context.Context, message *tgbotapi.Message) {
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
	// Simple command handling for now
	switch message.Text {
	case "/start":
		msg := tgbotapi.NewMessage(message.Chat.ID, "🚀 ServerEye Bot started!\n\nAvailable commands:\n/start - Show this message\n/help - Show help\n/temp - Get CPU temperature\n/memory - Get memory usage\n/disk - Get disk usage")
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("🌡️ Temperature"),
				tgbotapi.NewKeyboardButton("💾 Memory"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("💿 Disk"),
				tgbotapi.NewKeyboardButton("⏱️ Uptime"),
			),
		)
		if _, err := b.telegramAPI.Send(msg); err != nil {
			b.logger.Error("Failed to send start message", err)
		}
	case "/help":
		msg := tgbotapi.NewMessage(message.Chat.ID, "🤖 ServerEye Bot Help\n\nCommands:\n/start - Start bot\n/temp - CPU temperature\n/memory - Memory usage\n/disk - Disk usage\n/uptime - System uptime")
		if _, err := b.telegramAPI.Send(msg); err != nil {
			b.logger.Error("Failed to send help message", err)
		}
	case "/temp", "🌡️ Temperature":
		b.handleTemperatureCommand(ctx, message)
	case "/memory", "💾 Memory":
		b.handleMemoryCommand(ctx, message)
	case "/disk", "💿 Disk":
		b.handleDiskCommand(ctx, message)
	case "/uptime", "⏱️ Uptime":
		b.handleUptimeCommand(ctx, message)
	default:
		// Echo unknown messages
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Received: %s", message.Text))
		if _, err := b.telegramAPI.Send(msg); err != nil {
			b.logger.Error("Failed to send echo message", err)
		}
	}

	// Record metrics
	if b.metrics != nil {
		duration := time.Since(start).Seconds()
		b.metrics.RecordLatency("message_processing", duration)
	}

	b.logger.Info("Message processed successfully")
}

// handleCallback handles callback query processing
func (b *Bot) handleCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	start := time.Now()

	// Log callback details
	b.logger.Info("Processing callback query",
		Int64Field("user_id", query.From.ID),
		StringField("username", query.From.UserName),
		StringField("data", query.Data),
		StringField("query_id", query.ID))

	// Handle callback data
	callback := tgbotapi.NewCallback(query.ID, "Callback received")
	if _, err := b.telegramAPI.Request(callback); err != nil {
		b.logger.Error("Failed to answer callback query", err)
	}

	// Edit message to show callback was handled
	msg := tgbotapi.NewEditMessageText(query.Message.Chat.ID, query.Message.MessageID, fmt.Sprintf("Callback handled: %s", query.Data))
	if _, err := b.telegramAPI.Send(msg); err != nil {
		b.logger.Error("Failed to edit message for callback", err)
	}

	// Record metrics
	if b.metrics != nil {
		duration := time.Since(start).Seconds()
		b.metrics.RecordLatency("callback_processing", duration)
	}

	b.logger.Info("Callback processed successfully")
}

// handleTemperatureCommand handles temperature requests
func (b *Bot) handleTemperatureCommand(ctx context.Context, message *tgbotapi.Message) {
	temp, err := b.cpuMetrics.GetTemperature()
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Failed to get temperature: %v", err))
		b.telegramAPI.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("🌡️ CPU Temperature: %.1f°C", temp))
	if _, err := b.telegramAPI.Send(msg); err != nil {
		b.logger.Error("Failed to send temperature message", err)
	}
}

// handleMemoryCommand handles memory requests
func (b *Bot) handleMemoryCommand(ctx context.Context, message *tgbotapi.Message) {
	memInfo, err := b.systemMonitor.GetMemoryInfo()
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Failed to get memory info: %v", err))
		b.telegramAPI.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("💾 Memory Usage:\nTotal: %.1f GB\nAvailable: %.1f GB\nUsed: %.1f GB\nUsage: %.1f%%",
		float64(memInfo.Total)/1024/1024/1024,
		float64(memInfo.Available)/1024/1024/1024,
		float64(memInfo.Used)/1024/1024/1024,
		float64(memInfo.Used)/float64(memInfo.Total)*100))
	if _, err := b.telegramAPI.Send(msg); err != nil {
		b.logger.Error("Failed to send memory message", err)
	}
}

// handleDiskCommand handles disk requests
func (b *Bot) handleDiskCommand(ctx context.Context, message *tgbotapi.Message) {
	diskInfo, err := b.systemMonitor.GetDiskInfo()
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Failed to get disk info: %v", err))
		b.telegramAPI.Send(msg)
		return
	}

	var response string
	response = "💿 Disk Usage:\n"
	for _, disk := range diskInfo.Disks {
		response += fmt.Sprintf("%s: %.1f GB used / %.1f GB total (%.1f%%)\n",
			disk.Path,
			float64(disk.Used)/1024/1024/1024,
			float64(disk.Total)/1024/1024/1024,
			float64(disk.Used)/float64(disk.Total)*100)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response)
	if _, err := b.telegramAPI.Send(msg); err != nil {
		b.logger.Error("Failed to send disk message", err)
	}
}

// handleUptimeCommand handles uptime requests
func (b *Bot) handleUptimeCommand(ctx context.Context, message *tgbotapi.Message) {
	// Simple uptime implementation
	uptime, err := b.systemMonitor.GetUptime()
	if err != nil {
		msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("❌ Failed to get uptime: %v", err))
		b.telegramAPI.Send(msg)
		return
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("⏱️ System Uptime: %s", uptime.Formatted))
	if _, err := b.telegramAPI.Send(msg); err != nil {
		b.logger.Error("Failed to send uptime message", err)
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
