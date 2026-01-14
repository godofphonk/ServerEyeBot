package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/servereye/servereyebot/internal/config"
	"github.com/servereye/servereyebot/internal/logger"
	"github.com/servereye/servereyebot/internal/service"
	"github.com/servereye/servereyebot/internal/storage"
	"github.com/servereye/servereyebot/internal/telegram"
	"github.com/servereye/servereyebot/pkg/domain"
	"github.com/servereye/servereyebot/pkg/errors"
)

// Bot represents the updated bot with PostgreSQL integration
type Bot struct {
	config        *config.Config
	logger        logger.Logger
	telegramSvc   domain.TelegramService
	serverService *service.ServerService
	userService   domain.UserService
	updateHandler UpdateHandler
	commandRouter CommandRouter
	postgres      *storage.PostgreSQL
}

// UpdateHandler handles telegram updates
type UpdateHandler interface {
	HandleUpdate(ctx context.Context, update *telegram.Update) error
}

// CommandRouter routes commands to handlers
type CommandRouter interface {
	RegisterCommand(cmd *domain.Command) error
	RouteCommand(ctx context.Context, commandName string, args []string, user *domain.User) error
}

// New creates a new bot instance with PostgreSQL
func New(cfg *config.Config, log logger.Logger) (*Bot, error) {
	// Create PostgreSQL connection
	postgres, err := storage.NewPostgreSQL(cfg.Database.URL)
	if err != nil {
		return nil, errors.NewInternalError("failed to create postgres connection", err)
	}

	// Create telegram service
	telegramSvc, err := telegram.NewTelegramService(cfg.Telegram.Token, &logrusAdapter{logger: log})
	if err != nil {
		return nil, errors.NewInternalError("failed to create telegram service", err)
	}

	// Create repositories
	userRepo := storage.NewUserRepositoryAdapter(postgres)
	serverRepo := storage.NewServerRepositoryAdapter(postgres)
	userServerRepo := storage.NewUserServerRepositoryAdapter(postgres)

	// Create services
	serverService := service.NewServerService(serverRepo, userRepo, userServerRepo)
	userService := NewSimpleUserService(cfg)

	// Create command router
	commandRouter := NewDefaultCommandRouterNew(log, telegramSvc, userService, serverService)

	// Create update handler
	updateHandler := NewDefaultUpdateHandlerNew(log, telegramSvc, userService, commandRouter)

	bot := &Bot{
		config:        cfg,
		logger:        log,
		telegramSvc:   telegramSvc,
		serverService: serverService,
		userService:   userService,
		updateHandler: updateHandler,
		commandRouter: commandRouter,
		postgres:      postgres,
	}

	// Register commands
	if err := bot.registerCommands(); err != nil {
		return nil, errors.NewInternalError("failed to register commands", err)
	}

	return bot, nil
}

// registerCommands registers all bot commands
func (b *Bot) registerCommands() error {
	commands := []*domain.Command{
		{
			Name:        "start",
			Description: "Start bot and show welcome message",
			Handler:     b.handleStartCommand,
			Permissions: []string{},
		},
		{
			Name:        "help",
			Description: "Show available commands",
			Handler:     b.handleHelpCommand,
			Permissions: []string{},
		},
		{
			Name:        "servers",
			Description: "List your servers",
			Handler:     b.handleServersCommand,
			Permissions: []string{},
		},
		{
			Name:        "add",
			Description: "Add server to monitor",
			Handler:     b.handleAddServerCommand,
			Permissions: []string{},
		},
	}

	for _, cmd := range commands {
		if err := b.commandRouter.RegisterCommand(cmd); err != nil {
			return errors.NewInternalError("failed to register command", err)
		}
	}

	return nil
}

// getCommandList returns the list of bot commands
func (b *Bot) getCommandList() []domain.BotCommand {
	return []domain.BotCommand{
		{Command: "start", Description: "Start bot and show welcome message"},
		{Command: "help", Description: "Show available commands"},
		{Command: "servers", Description: "List your servers"},
		{Command: "add", Description: "Add server to monitor"},
	}
}

// Command handlers

func (b *Bot) handleStartCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	chatID := ctx.Value("chat_id").(int64)

	message := `👋 *Добро пожаловать в ServerEyeBot!*

Я помогу вам мониторить ваши серверы.

*Доступные команды:*
/start - Показать это сообщение
/help - Помощь
/servers - Список ваших серверов
/add <server_id> - Добавить сервер

Начните с команды /servers чтобы увидеть ваши серверы!`

	return b.telegramSvc.SendMessage(ctx, chatID, message)
}

func (b *Bot) handleHelpCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	chatID := ctx.Value("chat_id").(int64)

	message := `📖 *Помощь ServerEyeBot*

*Команды:*
• /start - Приветствие
• /help - Эта справка
• /servers - Показать ваши серверы
• /add <server_id> - Добавить сервер (например: /add srv_12313)

*Как добавить сервер:*
1. Используйте команду /add srv_12313
2. Бот добавит сервер в ваш список
3. Проверьте через /servers

*Управление серверами:*
Один пользователь может иметь много серверов, и один сервер может быть доступен многим пользователям.

Нужна помощь? Свяжитесь с администратором.`

	return b.telegramSvc.SendMessage(ctx, chatID, message)
}

func (b *Bot) handleServersCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	userID := ctx.Value("user_id").(int64)
	chatID := ctx.Value("chat_id").(int64)

	b.logger.Info("Getting user servers", "user_id", userID, "chat_id", chatID)

	// Get user servers
	servers, err := b.serverService.ListUserServers(ctx, userID)
	if err != nil {
		b.logger.Error("Failed to get user servers", "error", err, "user_id", userID)
		return b.telegramSvc.SendMessage(ctx, chatID, "❌ Произошла ошибка при получении списка серверов. Попробуйте позже.")
	}

	// Format and send servers list
	message := b.serverService.FormatServersForMessage(servers)
	return b.telegramSvc.SendMessage(ctx, chatID, message)
}

func (b *Bot) handleAddServerCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	if len(args) < 1 {
		chatID := ctx.Value("chat_id").(int64)
		return b.telegramSvc.SendMessage(ctx, chatID, "❌ Укажите ID сервера. Пример: /add srv_12313")
	}

	serverID := strings.TrimSpace(args[0])
	userID := ctx.Value("user_id").(int64)
	chatID := ctx.Value("chat_id").(int64)

	b.logger.Info("Adding server", "server_id", serverID, "user_id", userID, "chat_id", chatID)

	// Add server to user
	if err := b.serverService.AddServerToUser(ctx, userID, serverID, "owner"); err != nil {
		b.logger.Error("Failed to add server to user", "error", err, "server_id", serverID, "user_id", userID)
		return b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Не удалось добавить сервер `%s`. Попробуйте позже.", serverID))
	}

	successMsg := fmt.Sprintf("✅ Сервер `%s` успешно добавлен в ваш список!\n\nИспользуйте /servers для просмотра всех ваших серверов.", serverID)
	return b.telegramSvc.SendMessage(ctx, chatID, successMsg)
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) error {
	// Set bot commands
	if err := b.telegramSvc.SetCommands(ctx, b.getCommandList()); err != nil {
		b.logger.Error("Failed to set bot commands", "error", err)
	}

	// Start receiving updates
	return b.telegramSvc.StartReceivingUpdates(ctx, b.updateHandler)
}

// Stop stops the bot
func (b *Bot) Stop() {
	b.telegramSvc.StopReceivingUpdates()
	if err := b.postgres.Close(); err != nil {
		b.logger.Error("Failed to close database connection", "error", err)
	}
}

// DefaultUpdateHandler implements UpdateHandler
type DefaultUpdateHandler struct {
	logger        logger.Logger
	telegramSvc   domain.TelegramService
	userService   domain.UserService
	commandRouter CommandRouter
}

func NewDefaultUpdateHandlerNew(log logger.Logger, telegramSvc domain.TelegramService, userService domain.UserService, commandRouter CommandRouter) *DefaultUpdateHandler {
	return &DefaultUpdateHandler{
		logger:        log,
		telegramSvc:   telegramSvc,
		userService:   userService,
		commandRouter: commandRouter,
	}
}

func (h *DefaultUpdateHandler) HandleUpdate(ctx context.Context, update *telegram.Update) error {
	if update.Message != nil {
		return h.handleMessage(ctx, update.Message)
	}

	if update.CallbackQuery != nil {
		return h.handleCallback(ctx, update.CallbackQuery)
	}

	return nil
}

func (h *DefaultUpdateHandler) handleMessage(ctx context.Context, message *telegram.Message) error {
	// Register user if needed
	user := &domain.User{
		ID:         int(message.From.ID), // Convert to int for domain.User
		TelegramID: message.From.ID,
		Username:   message.From.Username,
		FirstName:  message.From.FirstName,
		LastName:   message.From.LastName,
		IsAdmin:    h.userService.IsAdmin(message.From.ID),
		CreatedAt:  time.Now(),
		LastSeen:   time.Now(),
	}

	if err := h.userService.RegisterUser(ctx, user); err != nil {
		h.logger.WithFields(map[string]interface{}{"error": err, "user_id": user.ID}).Warn("Failed to register user")
	}

	// Handle command
	if strings.HasPrefix(message.Text, "/") {
		parts := strings.Fields(message.Text)
		commandName := strings.TrimPrefix(parts[0], "/")
		args := parts[1:]

		return h.commandRouter.RouteCommand(ctx, commandName, args, user)
	}

	// Handle regular message
	return h.handleRegularMessage(ctx, message, user)
}

func (h *DefaultUpdateHandler) handleCallback(ctx context.Context, callback *telegram.CallbackQuery) error {
	// Answer callback
	if err := h.telegramSvc.AnswerCallback(ctx, callback.ID, "Processing..."); err != nil {
		return err
	}

	// Handle callback data
	return h.handleCallbackData(ctx, callback)
}

func (h *DefaultUpdateHandler) handleRegularMessage(ctx context.Context, message *telegram.Message, user *domain.User) error {
	// Help message for non-commands
	helpMsg := `🤔 Я не понимаю обычные сообщения.

Используйте команды:
/start - Начать
/help - Помощь
/servers - Ваши сервера
/add <server_id> - Добавить сервер`
	return h.telegramSvc.SendMessage(ctx, message.Chat.ID, helpMsg)
}

func (h *DefaultUpdateHandler) handleCallbackData(ctx context.Context, callback *telegram.CallbackQuery) error {
	// Handle button callbacks
	switch callback.Data {
	default:
		return h.telegramSvc.SendMessage(ctx, callback.Message.Chat.ID, "Unknown callback")
	}
}

// DefaultCommandRouter implements CommandRouter
type DefaultCommandRouter struct {
	logger        logger.Logger
	telegramSvc   domain.TelegramService
	userService   domain.UserService
	serverService *service.ServerService
	commands      map[string]*domain.Command
}

func NewDefaultCommandRouterNew(log logger.Logger, telegramSvc domain.TelegramService, userService domain.UserService, serverService *service.ServerService) *DefaultCommandRouter {
	return &DefaultCommandRouter{
		logger:        log,
		telegramSvc:   telegramSvc,
		userService:   userService,
		serverService: serverService,
		commands:      make(map[string]*domain.Command),
	}
}

func (r *DefaultCommandRouter) RegisterCommand(cmd *domain.Command) error {
	r.commands[cmd.Name] = cmd
	r.logger.WithField("name", cmd.Name).Debug("Command registered")
	return nil
}

func (r *DefaultCommandRouter) RouteCommand(ctx context.Context, commandName string, args []string, user *domain.User) error {
	cmd, exists := r.commands[commandName]
	if !exists {
		return r.telegramSvc.SendMessage(ctx, user.TelegramID, fmt.Sprintf("❌ Неизвестная команда: /%s\n\nИспользуйте /help для списка команд.", commandName))
	}

	// Check permissions
	if len(cmd.Permissions) > 0 {
		for _, perm := range cmd.Permissions {
			if perm == "admin" && !user.IsAdmin {
				return r.telegramSvc.SendMessage(ctx, user.TelegramID, "Эта команда требует прав администратора")
			}
		}
	}

	// Add user info to context
	ctx = context.WithValue(ctx, "user_id", user.TelegramID)
	ctx = context.WithValue(ctx, "chat_id", user.TelegramID)

	// Execute command
	return cmd.Handler(ctx, cmd, args)
}

// Helper types and implementations

// logrusAdapter adapts our logger interface to logrus
type logrusAdapter struct {
	logger logger.Logger
}

func (l *logrusAdapter) Debug(msg string, fields ...interface{}) {
	fieldMap := make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				fieldMap[key] = fields[i+1]
			}
		}
	}
	l.logger.WithFields(fieldMap).Debug(msg)
}

func (l *logrusAdapter) Info(msg string, fields ...interface{}) {
	fieldMap := make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				fieldMap[key] = fields[i+1]
			}
		}
	}
	l.logger.WithFields(fieldMap).Info(msg)
}

func (l *logrusAdapter) Warn(msg string, fields ...interface{}) {
	fieldMap := make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				fieldMap[key] = fields[i+1]
			}
		}
	}
	l.logger.WithFields(fieldMap).Warn(msg)
}

func (l *logrusAdapter) Error(msg string, fields ...interface{}) {
	fieldMap := make(map[string]interface{})
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				fieldMap[key] = fields[i+1]
			}
		}
	}
	l.logger.WithFields(fieldMap).Error(msg)
}

// SimpleUserService implements domain.UserService
type SimpleUserService struct {
	config *config.Config
}

func NewSimpleUserService(cfg *config.Config) *SimpleUserService {
	return &SimpleUserService{config: cfg}
}

func (s *SimpleUserService) IsAdmin(userID int64) bool {
	return s.config.Telegram.AdminUserID == userID
}

func (s *SimpleUserService) IsAuthorized(userID int64) bool {
	if s.config.Telegram.AdminUserID == userID {
		return true
	}

	for _, allowedID := range s.config.Telegram.AllowedUserIDs {
		if allowedID == userID {
			return true
		}
	}

	return !s.config.Telegram.PrivateMode
}

func (s *SimpleUserService) RegisterUser(ctx context.Context, user *domain.User) error {
	// Simple implementation - in production, store in database
	return nil
}

func (s *SimpleUserService) GetUser(ctx context.Context, userID int64) (*domain.User, error) {
	// Simple implementation - in production, fetch from database
	return nil, errors.NewNotFoundError("user")
}
