package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/servereye/servereyebot/internal/api"
	"github.com/servereye/servereyebot/internal/config"
	"github.com/servereye/servereyebot/internal/httpserver"
	"github.com/servereye/servereyebot/internal/logger"
	"github.com/servereye/servereyebot/internal/models"
	"github.com/servereye/servereyebot/internal/repository"
	"github.com/servereye/servereyebot/internal/service"
	"github.com/servereye/servereyebot/internal/services"
	"github.com/servereye/servereyebot/internal/storage"
	"github.com/servereye/servereyebot/internal/telegram"
	"github.com/servereye/servereyebot/pkg/domain"
	"github.com/servereye/servereyebot/pkg/errors"
)

// Context keys
type contextKey string

const (
	userIDKey contextKey = "user_id"
	chatIDKey contextKey = "chat_id"
)

// Bot represents the updated bot with PostgreSQL integration
type Bot struct {
	config         *config.Config
	logger         logger.Logger
	telegramSvc    domain.TelegramService
	serverService  *service.ServerService
	userService    domain.UserService
	metricsService *services.MetricsServiceImpl
	updateHandler  UpdateHandler
	commandRouter  CommandRouter
	postgres       *storage.PostgreSQL
	httpServer     *httpserver.HttpServer
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
	postgresRepo, err := repository.NewPostgresRepository(cfg.Database.URL)
	if err != nil {
		return nil, errors.NewInternalError("failed to create postgres repository", err)
	}

	// Create API client
	apiClient := api.NewClient(cfg.API.BaseURL, &logrusAdapter{logger: log})

	realUserService := services.NewUserService(postgresRepo, apiClient)
	serverService := service.NewServerService(serverRepo, userRepo, userServerRepo)
	userService := services.NewUserServiceAdapter(realUserService)

	// Create metrics service
	metricsService := services.NewMetricsService(apiClient, &logrusAdapter{logger: log})

	// Create command router
	commandRouter := NewDefaultCommandRouterNew(log, telegramSvc, userService, serverService, metricsService)

	// Create update handler
	updateHandler := NewDefaultUpdateHandlerNew(log, telegramSvc, userService, commandRouter, serverService, metricsService)

	// Create HTTP server for health checks
	httpServer := httpserver.New(cfg.App.Port, log)

	bot := &Bot{
		config:         cfg,
		logger:         log,
		telegramSvc:    telegramSvc,
		serverService:  serverService,
		userService:    userService,
		metricsService: metricsService,
		updateHandler:  updateHandler,
		commandRouter:  commandRouter,
		postgres:       postgres,
		httpServer:     httpServer,
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
			Name:        "rename",
			Description: "Rename a server",
			Handler:     b.handleRenameCommand,
			Permissions: []string{},
		},
		{
			Name:        "add",
			Description: "Add server to monitor",
			Handler:     b.handleAddServerCommand,
			Permissions: []string{},
		},
		{
			Name:        "cpu",
			Description: "Show CPU metrics",
			Handler:     b.handleCPUCommand,
			Permissions: []string{},
		},
		{
			Name:        "memory",
			Description: "Show memory metrics",
			Handler:     b.handleMemoryCommand,
			Permissions: []string{},
		},
		{
			Name:        "disk",
			Description: "Show disk metrics",
			Handler:     b.handleDiskCommand,
			Permissions: []string{},
		},
		{
			Name:        "temp",
			Description: "Show temperature metrics",
			Handler:     b.handleTempCommand,
			Permissions: []string{},
		},
		{
			Name:        "network",
			Description: "Show network metrics",
			Handler:     b.handleNetworkCommand,
			Permissions: []string{},
		},
		{
			Name:        "system",
			Description: "Show system information",
			Handler:     b.handleSystemCommand,
			Permissions: []string{},
		},
		{
			Name:        "all",
			Description: "Show all metrics summary",
			Handler:     b.handleAllCommand,
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
		{Command: "rename", Description: "Rename a server"},
		{Command: "add", Description: "Add server to monitor"},
		{Command: "cpu", Description: "Show CPU metrics"},
		{Command: "memory", Description: "Show memory metrics"},
		{Command: "disk", Description: "Show disk metrics"},
		{Command: "temp", Description: "Show temperature metrics"},
		{Command: "network", Description: "Show network metrics"},
		{Command: "system", Description: "Show system information"},
		{Command: "all", Description: "Show all metrics summary"},
	}
}

// Command handlers

func (b *Bot) handleStartCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	chatID := ctx.Value(chatIDKey).(int64)

	message := `👋 *Добро пожаловать в ServerEyeBot!*

Я помогу вам мониторить ваши серверы.

*Доступные команды:*
/start - Показать это сообщение
/help - Помощь и список всех команд
/servers - Список ваших серверов
/add <server_id> - Добавить сервер

*Команды метрик:*
/cpu [server_id] - Загрузка процессора
/memory [server_id] - Использование памяти
/disk [server_id] - Дисковое пространство
/temp [server_id] - Температура системы
/network [server_id] - Сетевая активность
/system [server_id] - Системная информация
/all [server_id] - Все метрики (кратко)

Начните с команды /servers чтобы увидеть ваши серверы!`

	return b.telegramSvc.SendMessage(ctx, chatID, message)
}

func (b *Bot) handleHelpCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	chatID := ctx.Value(chatIDKey).(int64)

	message := `📖 *Помощь ServerEyeBot*

*Основные команды:*
• /start - Приветствие
• /help - Эта справка
• /servers - Показать ваши серверы
• /add <server_id> - Добавить сервер (например: /add srv_12313)

*Команды метрик:*
• /cpu [server_id] - Загрузка процессора
• /memory [server_id] - Использование памяти
• /disk [server_id] - Дисковое пространство
• /temp [server_id] - Температура системы
• /network [server_id] - Сетевая активность
• /system [server_id] - Системная информация
• /all [server_id] - Все метрики (кратко)

*Как добавить сервер:*
1. Используйте команду /add srv_12313
2. Бот добавит сервер в ваш список
3. Проверьте через /servers
4. Используйте команды метрик для просмотра данных

*Управление серверами:*
Один пользователь может иметь много серверов, и один сервер может быть доступен многим пользователям.

*Выбор сервера для метрик:*
• Если у вас один сервер - метрики показываются автоматически
• Если несколько серверов - используйте /cpu server_id для конкретного сервера
• При вызове без параметра - увидите список доступных серверов

Нужна помощь? Свяжитесь с администратором.`

	return b.telegramSvc.SendMessage(ctx, chatID, message)
}

func (b *Bot) handleServersCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	b.logger.Info("Getting user servers", "telegram_id", telegramID, "chat_id", chatID)

	// Get user servers using UserServiceAdapter
	if adapter, ok := b.userService.(*services.UserServiceAdapter); ok {
		// Get user from database to get correct user_id
		user, err := adapter.GetUser(ctx, telegramID)
		if err != nil {
			b.logger.Error("Failed to get user", "error", err, "telegram_id", telegramID)
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ Внутренняя ошибка. Попробуйте позже.")
		}

		servers, err := adapter.GetUserServers(ctx, int64(user.ID))
		if err != nil {
			b.logger.Error("Failed to get user servers", "error", err, "user_id", user.ID)
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ Произошла ошибка при получении списка серверов. Попробуйте позже.")
		}

		// Format and send servers list with remove button
		message := adapter.FormatServersListPlain(servers)

		if len(servers) > 0 {
			// Create inline keyboard with remove and rename buttons
			keyboard := [][]map[string]string{
				{
					{
						"text":          "Изменить имя сервера",
						"callback_data": "show_rename_servers",
					},
					{
						"text":          "Удалить сервер",
						"callback_data": "show_remove_servers",
					},
				},
			}
			return b.telegramSvc.SendMessageWithKeyboard(ctx, chatID, message, keyboard)
		}

		return b.telegramSvc.SendMessage(ctx, chatID, message)
	}

	return b.telegramSvc.SendMessage(ctx, chatID, "❌ Внутренняя ошибка сервиса. Попробуйте позже.")
}

func (b *Bot) handleAddServerCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	if len(args) < 1 {
		chatID := ctx.Value(chatIDKey).(int64)
		return b.telegramSvc.SendMessage(ctx, chatID, "❌ Укажите ID сервера. Пример: /add srv_12313")
	}

	serverID := strings.TrimSpace(args[0])
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	b.logger.Info("Adding server", "server_id", serverID, "telegram_id", telegramID, "chat_id", chatID)

	// Add server to user using UserServiceAdapter
	if adapter, ok := b.userService.(*services.UserServiceAdapter); ok {
		// Get user from database to get correct user_id
		user, err := adapter.GetUser(ctx, telegramID)
		if err != nil {
			b.logger.Error("Failed to get user", "error", err, "telegram_id", telegramID)
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ Внутренняя ошибка. Попробуйте позже.")
		}

		if err := adapter.AddServerToUser(ctx, int64(user.ID), serverID, "TGBot"); err != nil {
			b.logger.Error("Failed to add server to user", "error", err, "server_id", serverID, "user_id", user.ID)

			// Check error type and provide specific message
			errorMsg := err.Error()
			if strings.Contains(errorMsg, "not found") {
				return b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Сервер `%s` не найден.", serverID))
			} else if strings.Contains(errorMsg, "API error") {
				return b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Ошибка при проверке сервера `%s`. Попробуйте позже.", serverID))
			} else if strings.Contains(errorMsg, "Invalid server key") {
				return b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Неверный формат ключа сервера `%s`.", serverID))
			} else {
				return b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Не удалось добавить сервер `%s`. Попробуйте позже.", serverID))
			}
		}

		// Add Telegram ID to server identifiers
		if err := adapter.AddTelegramIdentifierToServer(ctx, int64(user.ID), serverID, fmt.Sprintf("%d", telegramID), user.Username, user.FirstName); err != nil {
			b.logger.Warn("Failed to add Telegram identifier to server", "error", err, "server_id", serverID, "telegram_id", telegramID)
			// Don't fail the operation, just log the warning
		}

		successMsg := fmt.Sprintf("✅ Сервер `%s` успешно добавлен в ваш список!\n\nИспользуйте /servers для просмотра всех ваших серверов.", serverID)
		return b.telegramSvc.SendMessage(ctx, chatID, successMsg)
	}

	return b.telegramSvc.SendMessage(ctx, chatID, "❌ Внутренняя ошибка сервиса. Попробуйте позже.")
}

func (b *Bot) handleRenameCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	if len(args) < 2 {
		chatID := ctx.Value(chatIDKey).(int64)
		return b.telegramSvc.SendMessage(ctx, chatID, "❌ Укажите ID сервера и новое имя. Пример: /rename key_12313 \"Мой сервер\"")
	}

	serverID := args[0]
	newName := strings.Join(args[1:], " ") // Объединяем все остальные аргументы как имя
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	b.logger.Info("Renaming server", "server_id", serverID, "new_name", newName, "telegram_id", telegramID)

	// Get user servers using UserServiceAdapter
	if adapter, ok := b.userService.(*services.UserServiceAdapter); ok {
		user, err := adapter.GetUser(ctx, telegramID)
		if err != nil {
			b.logger.Error("Failed to get user", "error", err, "telegram_id", telegramID)
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ Внутренняя ошибка. Попробуйте позже.")
		}

		servers, err := adapter.GetUserServers(ctx, int64(user.ID))
		if err != nil {
			b.logger.Error("Failed to get user servers", "error", err, "user_id", user.ID)
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ Произошла ошибка при получении списка серверов. Попробуйте позже.")
		}

		// Find the server to rename
		var serverToRename *models.ServerWithDetails
		for _, server := range servers {
			if server.ID == serverID {
				serverToRename = &server
				break
			}
		}

		if serverToRename == nil {
			return b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Сервер `%s` не найден в вашем списке.", serverID))
		}

		// Update server name in database
		err = adapter.UpdateServerName(ctx, int64(user.ID), serverID, newName)
		if err != nil {
			b.logger.Error("Failed to update server name", "error", err, "server_id", serverID, "new_name", newName)
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ Не удалось переименовать сервер. Попробуйте позже.")
		}

		successMsg := fmt.Sprintf("✅ Сервер `%s` успешно переименован в `%s`!", serverID, newName)
		return b.telegramSvc.SendMessage(ctx, chatID, successMsg)
	}

	return b.telegramSvc.SendMessage(ctx, chatID, "❌ Внутренняя ошибка сервиса. Попробуйте позже.")
}

// Start starts the bot
func (b *Bot) Start(ctx context.Context) error {
	// Start HTTP server for health checks
	if err := b.httpServer.Start(ctx); err != nil {
		b.logger.Error("Failed to start HTTP server", "error", err)
		return err
	}

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

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := b.httpServer.Stop(ctx); err != nil {
		b.logger.Error("Failed to stop HTTP server", "error", err)
	}

	if err := b.postgres.Close(); err != nil {
		b.logger.Error("Failed to close database connection", "error", err)
	}
}

// DefaultUpdateHandler implements UpdateHandler
type DefaultUpdateHandler struct {
	logger         logger.Logger
	telegramSvc    domain.TelegramService
	userService    domain.UserService
	commandRouter  CommandRouter
	serverService  *service.ServerService
	metricsService *services.MetricsServiceImpl
}

func NewDefaultUpdateHandlerNew(log logger.Logger, telegramSvc domain.TelegramService, userService domain.UserService, commandRouter CommandRouter, serverService *service.ServerService, metricsService *services.MetricsServiceImpl) *DefaultUpdateHandler {
	return &DefaultUpdateHandler{
		logger:         log,
		telegramSvc:    telegramSvc,
		userService:    userService,
		commandRouter:  commandRouter,
		serverService:  serverService,
		metricsService: metricsService,
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
	// Check if user is in rename mode (simplified approach)
	// For now, we'll handle rename requests with /rename command format

	// Help message for non-commands
	helpMsg := `🤔 Я не понимаю обычные сообщения.

Используйте команды:
/start - Начать
/help - Помощь
/servers - Ваши сервера
/add <server_id> - Добавить сервер
/rename <server_id> <new_name> - Переименовать сервер`
	return h.telegramSvc.SendMessage(ctx, message.Chat.ID, helpMsg)
}

func (h *DefaultUpdateHandler) handleCallbackData(ctx context.Context, callback *telegram.CallbackQuery) error {
	// Debug log to see what callback data we receive
	h.logger.Info("Received callback", "data", callback.Data, "from", callback.From.ID)

	// Handle button callbacks
	switch callback.Data {
	case "show_remove_servers":
		// Handle show remove servers callback - need to get bot instance differently
		return h.handleShowRemoveServersCallback(ctx, callback)
	case "show_rename_servers":
		// Handle show rename servers callback
		return h.handleShowRenameServersCallback(ctx, callback)
	default:
		// Handle server removal callbacks
		if len(callback.Data) > 14 && callback.Data[:14] == "remove_server:" {
			h.logger.Info("Processing remove server callback")
			return h.handleRemoveServerCallback(ctx, callback)
		}

		// Handle server rename callbacks
		if len(callback.Data) > 14 && callback.Data[:14] == "rename_server:" {
			h.logger.Info("Processing rename server callback")
			return h.handleRenameServerCallback(ctx, callback)
		}

		// Handle metrics callbacks
		if len(callback.Data) > 7 && callback.Data[:7] == "metric:" {
			h.logger.Info("Processing metric callback")
			return h.handleMetricCallback(ctx, callback)
		}

		h.logger.Warn("Unknown callback data", "data", callback.Data)
		return h.telegramSvc.SendMessage(ctx, callback.Message.Chat.ID, "Unknown callback")
	}
}

// handleShowRemoveServersCallback handles show remove servers callback
func (h *DefaultUpdateHandler) handleShowRemoveServersCallback(ctx context.Context, callback *telegram.CallbackQuery) error {
	// Get user servers using UserServiceAdapter
	if adapter, ok := h.userService.(*services.UserServiceAdapter); ok {
		// Get user from database to get correct user_id
		user, err := adapter.GetUser(ctx, callback.From.ID)
		if err != nil {
			h.logger.Error("Failed to get user", "error", err, "telegram_id", callback.From.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка")
		}

		servers, err := adapter.GetUserServers(ctx, int64(user.ID))
		if err != nil {
			h.logger.Error("Failed to get user servers", "error", err, "user_id", user.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Ошибка получения серверов")
		}

		if len(servers) == 0 {
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "У вас нет серверов для удаления")
		}

		// Create inline keyboard with server removal buttons
		keyboard := createRemoveServerKeyboard(servers)

		message := "Выберите сервер для удаления:\n\n"
		for _, server := range servers {
			message += fmt.Sprintf("• %s(%s)\n", server.Name, server.ID)
		}
		message += "\nНажмите на сервер который хотите удалить"

		// Answer callback and send new message
		if err := h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "Показываю серверы для удаления"); err != nil {
			h.logger.Error("Failed to answer callback", "error", err)
		}

		return h.telegramSvc.SendMessageWithKeyboard(ctx, callback.Message.Chat.ID, message, keyboard)
	}

	return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка сервиса")
}

// handleRemoveServerCallback handles remove server callback
func (h *DefaultUpdateHandler) handleRemoveServerCallback(ctx context.Context, callback *telegram.CallbackQuery) error {
	serverID := callback.Data[14:] // Remove "remove_server:" prefix

	// Get user from database to get correct user_id
	if adapter, ok := h.userService.(*services.UserServiceAdapter); ok {
		user, err := adapter.GetUser(ctx, callback.From.ID)
		if err != nil {
			h.logger.Error("Failed to get user", "error", err, "telegram_id", callback.From.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка")
		}

		servers, err := adapter.GetUserServers(ctx, int64(user.ID))
		if err != nil {
			h.logger.Error("Failed to get user servers", "error", err, "user_id", user.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Ошибка получения серверов")
		}

		// Find server name for better messaging
		var serverName string
		for _, server := range servers {
			if server.ID == serverID {
				serverName = server.Name
				break
			}
		}

		// If not found, use serverID as fallback
		if serverName == "" {
			serverName = serverID
		}

		// Remove server from user
		if err := adapter.RemoveServerFromUser(ctx, int64(user.ID), serverID); err != nil {
			h.logger.Error("Failed to remove server", "error", err, "server_id", serverID, "user_id", user.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Не удалось удалить сервер")
		}

		// Answer callback and update message
		callbackMsg := fmt.Sprintf("Сервер %s(%s) удален", serverName, serverID)
		if err := h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, callbackMsg); err != nil {
			h.logger.Error("Failed to answer callback", "error", err)
		}

		// Update original message to show server was removed
		newMessage := fmt.Sprintf("Сервер %s(%s) успешно удален из вашего списка.", serverName, serverID)
		return h.telegramSvc.EditMessage(ctx, callback.Message.Chat.ID, callback.Message.MessageID, newMessage, nil)
	}

	return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка сервиса")
}

// handleShowRenameServersCallback handles show rename servers callback
func (h *DefaultUpdateHandler) handleShowRenameServersCallback(ctx context.Context, callback *telegram.CallbackQuery) error {
	// Get user servers using UserServiceAdapter
	if adapter, ok := h.userService.(*services.UserServiceAdapter); ok {
		// Get user from database to get correct user_id
		user, err := adapter.GetUser(ctx, callback.From.ID)
		if err != nil {
			h.logger.Error("Failed to get user", "error", err, "telegram_id", callback.From.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка")
		}

		servers, err := adapter.GetUserServers(ctx, int64(user.ID))
		if err != nil {
			h.logger.Error("Failed to get user servers", "error", err, "user_id", user.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Ошибка получения серверов")
		}

		if len(servers) == 0 {
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "У вас нет серверов для переименования")
		}

		// Create inline keyboard with server rename buttons
		keyboard := createRenameServerKeyboard(servers)

		message := "Выберите сервер для переименования:\n\n"
		for _, server := range servers {
			message += fmt.Sprintf("• %s(%s)\n", server.Name, server.ID)
		}
		message += "\nНажмите на сервер который хотите переименовать"

		// Answer callback and send new message
		if err := h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "Показываю серверы для переименования"); err != nil {
			h.logger.Error("Failed to answer callback", "error", err)
		}

		return h.telegramSvc.SendMessageWithKeyboard(ctx, callback.Message.Chat.ID, message, keyboard)
	}

	return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка сервиса")
}

// handleRenameServerCallback handles server rename callback
func (h *DefaultUpdateHandler) handleRenameServerCallback(ctx context.Context, callback *telegram.CallbackQuery) error {
	serverID := callback.Data[14:] // Remove "rename_server:" prefix

	// Get user from database to get correct user_id
	if adapter, ok := h.userService.(*services.UserServiceAdapter); ok {
		user, err := adapter.GetUser(ctx, callback.From.ID)
		if err != nil {
			h.logger.Error("Failed to get user", "error", err, "telegram_id", callback.From.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка")
		}

		servers, err := adapter.GetUserServers(ctx, int64(user.ID))
		if err != nil {
			h.logger.Error("Failed to get user servers", "error", err, "user_id", user.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Ошибка получения серверов")
		}

		// Find the server to rename
		var serverToRename *models.ServerWithDetails
		for _, server := range servers {
			if server.ID == serverID {
				serverToRename = &server
				break
			}
		}

		if serverToRename == nil {
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Сервер не найден")
		}

		// Send instructions for renaming
		message := "📝 *Переименование сервера*\n\n"
		message += fmt.Sprintf("Текущий сервер: %s(%s)\n\n", serverToRename.Name, serverToRename.ID)
		message += "🔄 *Отправьте новое имя для этого сервера в следующем сообщении*\n\n"
		message += "💡 *Пример:* `Мой рабочий сервер`\n\n"
		message += "❌ *Отмена:* отправьте `/cancel`"

		if err := h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "Ожидаю новое имя сервера"); err != nil {
			h.logger.Error("Failed to answer callback", "error", err)
		}

		return h.telegramSvc.SendMessage(ctx, callback.Message.Chat.ID, message)
	}

	return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка сервиса")
}

// handleMetricCallback handles metric selection callbacks
func (h *DefaultUpdateHandler) handleMetricCallback(ctx context.Context, callback *telegram.CallbackQuery) error {
	h.logger.Info("handleMetricCallback called", "callback_data", callback.Data)

	// Parse callback data: metric:metric_type:server_id
	parts := strings.Split(callback.Data, ":")
	h.logger.Info("Callback parts", "parts", parts, "len", len(parts))

	if len(parts) != 3 {
		h.logger.Error("Invalid callback data format", "parts", parts)
		return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Неверный формат данных")
	}

	metricType := parts[1]
	serverID := parts[2]

	h.logger.Info("Parsed callback", "metric_type", metricType, "server_id", serverID)

	// Get user servers
	if adapter, ok := h.userService.(*services.UserServiceAdapter); ok {
		user, err := adapter.GetUser(ctx, callback.From.ID)
		if err != nil {
			h.logger.Error("Failed to get user", "error", err, "telegram_id", callback.From.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка")
		}

		servers, err := adapter.GetUserServers(ctx, int64(user.ID))
		if err != nil {
			h.logger.Error("Failed to get user servers", "error", err, "user_id", user.ID)
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Ошибка получения серверов")
		}

		// Find the requested server
		var selectedServer *models.ServerWithDetails
		for _, server := range servers {
			if server.ID == serverID {
				selectedServer = &server
				h.logger.Info("Found server", "server_id", server.ID, "server_name", server.Name, "server_key", server.ServerKey)
				break
			}
		}

		if selectedServer == nil {
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Сервер не найден")
		}

		// Get metrics for the selected server
		serverKey := selectedServer.ServerKey
		h.logger.Info("Using server key", "server_key", serverKey, "server_id", selectedServer.ID)
		metrics, err := h.metricsService.GetServerMetrics(serverKey)
		if err != nil {
			h.logger.Error("Failed to get server metrics", "error", err, "server_key", serverKey)

			errorMsg := "❌ Не удалось получить метрики"
			if strings.Contains(err.Error(), "not found") {
				errorMsg = fmt.Sprintf("❌ Сервер `%s` не найден", serverKey)
			} else if strings.Contains(err.Error(), "API error") {
				errorMsg = fmt.Sprintf("❌ Не удалось получить метрики для сервера `%s`", serverKey)
			}

			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, errorMsg)
		}

		// Format metrics based on type
		var formattedMetrics string
		switch metricType {
		case "cpu":
			formattedMetrics = h.metricsService.FormatCPU(&metrics.Metrics)
		case "memory":
			formattedMetrics = h.metricsService.FormatMemory(&metrics.Metrics)
		case "disk":
			formattedMetrics = h.metricsService.FormatDisk(&metrics.Metrics)
		case "temperature":
			formattedMetrics = h.metricsService.FormatTemperature(&metrics.Metrics)
		case "network":
			formattedMetrics = h.metricsService.FormatNetwork(&metrics.Metrics)
		case "system":
			formattedMetrics = h.metricsService.FormatSystem(&metrics.Metrics)
		case "all":
			formattedMetrics = h.metricsService.FormatAll(&metrics.Metrics)
		default:
			return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Неизвестный тип метрики")
		}

		// Answer callback and send metrics
		if err := h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, fmt.Sprintf("Метрики %s для %s", metricType, selectedServer.Name)); err != nil {
			h.logger.Error("Failed to answer callback", "error", err)
		}

		return h.telegramSvc.SendMessage(ctx, callback.Message.Chat.ID, formattedMetrics)
	}

	return h.telegramSvc.AnswerCallbackQuery(ctx, callback.ID, "❌ Внутренняя ошибка сервиса")
}

// createRemoveServerKeyboard creates inline keyboard for server removal
func createRemoveServerKeyboard(servers []models.ServerWithDetails) interface{} {
	var buttons [][]map[string]string

	for _, server := range servers {
		button := []map[string]string{
			{
				"text":          fmt.Sprintf("Удалить %s(%s)", server.Name, server.ID),
				"callback_data": fmt.Sprintf("remove_server:%s", server.ID),
			},
		}
		buttons = append(buttons, button)
	}

	return buttons
}

// createRenameServerKeyboard creates inline keyboard for server renaming
func createRenameServerKeyboard(servers []models.ServerWithDetails) interface{} {
	var buttons [][]map[string]string

	for _, server := range servers {
		button := []map[string]string{
			{
				"text":          fmt.Sprintf("Переименовать %s(%s)", server.Name, server.ID),
				"callback_data": fmt.Sprintf("rename_server:%s", server.ID),
			},
		}
		buttons = append(buttons, button)
	}

	return buttons
}

// DefaultCommandRouter implements CommandRouter
type DefaultCommandRouter struct {
	logger         logger.Logger
	telegramSvc    domain.TelegramService
	userService    domain.UserService
	serverService  *service.ServerService
	metricsService *services.MetricsServiceImpl
	commands       map[string]*domain.Command
}

func NewDefaultCommandRouterNew(log logger.Logger, telegramSvc domain.TelegramService, userService domain.UserService, serverService *service.ServerService, metricsService *services.MetricsServiceImpl) *DefaultCommandRouter {
	return &DefaultCommandRouter{
		logger:         log,
		telegramSvc:    telegramSvc,
		userService:    userService,
		serverService:  serverService,
		metricsService: metricsService,
		commands:       make(map[string]*domain.Command),
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
	ctx = context.WithValue(ctx, userIDKey, user.TelegramID)
	ctx = context.WithValue(ctx, chatIDKey, user.TelegramID)

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

// Metrics command handlers

func (b *Bot) handleCPUCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	return b.handleMetricsCommand(ctx, telegramID, chatID, "cpu", args, func(metrics *domain.ServerMetrics) string {
		return b.metricsService.FormatCPU(metrics)
	})
}

func (b *Bot) handleMemoryCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	return b.handleMetricsCommand(ctx, telegramID, chatID, "memory", args, func(metrics *domain.ServerMetrics) string {
		return b.metricsService.FormatMemory(metrics)
	})
}

func (b *Bot) handleDiskCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	return b.handleMetricsCommand(ctx, telegramID, chatID, "disk", args, func(metrics *domain.ServerMetrics) string {
		return b.metricsService.FormatDisk(metrics)
	})
}

func (b *Bot) handleTempCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	return b.handleMetricsCommand(ctx, telegramID, chatID, "temperature", args, func(metrics *domain.ServerMetrics) string {
		return b.metricsService.FormatTemperature(metrics)
	})
}

func (b *Bot) handleNetworkCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	return b.handleMetricsCommand(ctx, telegramID, chatID, "network", args, func(metrics *domain.ServerMetrics) string {
		return b.metricsService.FormatNetwork(metrics)
	})
}

func (b *Bot) handleSystemCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	return b.handleMetricsCommand(ctx, telegramID, chatID, "system", args, func(metrics *domain.ServerMetrics) string {
		return b.metricsService.FormatSystem(metrics)
	})
}

func (b *Bot) handleAllCommand(ctx context.Context, cmd *domain.Command, args []string) error {
	telegramID := ctx.Value(userIDKey).(int64)
	chatID := ctx.Value(chatIDKey).(int64)

	return b.handleMetricsCommand(ctx, telegramID, chatID, "all", args, func(metrics *domain.ServerMetrics) string {
		return b.metricsService.FormatAll(metrics)
	})
}

// selectServer handles server selection for metrics commands
func (b *Bot) selectServer(ctx context.Context, chatID int64, metricType string, servers []models.ServerWithDetails, args []string) (*models.ServerWithDetails, error) {
	// If only one server, use it
	if len(servers) == 1 {
		return &servers[0], nil
	}

	// If server ID provided in arguments, try to find it
	if len(args) > 0 {
		serverID := args[0]
		for _, server := range servers {
			if server.ID == serverID || server.Name == serverID {
				return &server, nil
			}
		}
		return nil, b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Сервер `%s` не найден в вашем списке.", serverID))
	}

	// Multiple servers and no specific server requested - show selection buttons
	var keyboard [][]map[string]string

	for _, server := range servers {
		callbackData := fmt.Sprintf("metric:%s:%s", metricType, server.ID)
		button := []map[string]string{
			{
				"text":          fmt.Sprintf("🖥️ %s(%s)", server.Name, server.ID),
				"callback_data": callbackData,
			},
		}
		keyboard = append(keyboard, button)
		b.logger.Info("Created button", "server", server.Name, "callback_data", callbackData)
	}

	message := fmt.Sprintf("📊 *Выберите сервер для метрики %s:*", metricType)
	b.logger.Info("Sending keyboard message", "servers_count", len(servers), "metric_type", metricType)

	return nil, b.telegramSvc.SendMessageWithKeyboard(ctx, chatID, message, keyboard)
}

// handleMetricsCommand is a generic handler for metrics commands
func (b *Bot) handleMetricsCommand(ctx context.Context, telegramID, chatID int64, metricType string, args []string, formatter func(*domain.ServerMetrics) string) error {
	b.logger.Info("Getting metrics", "type", metricType, "telegram_id", telegramID, "chat_id", chatID)

	// Get user servers
	if adapter, ok := b.userService.(*services.UserServiceAdapter); ok {
		user, err := adapter.GetUser(ctx, telegramID)
		if err != nil {
			b.logger.Error("Failed to get user", "error", err, "telegram_id", telegramID)
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ Внутренняя ошибка. Попробуйте позже.")
		}

		servers, err := adapter.GetUserServers(ctx, int64(user.ID))
		if err != nil {
			b.logger.Error("Failed to get user servers", "error", err, "user_id", user.ID)
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ Произошла ошибка при получении списка серверов. Попробуйте позже.")
		}

		if len(servers) == 0 {
			return b.telegramSvc.SendMessage(ctx, chatID, "❌ У вас нет добавленных серверов. Используйте /add <server_id> для добавления сервера.")
		}

		// Handle server selection
		server, err := b.selectServer(ctx, chatID, metricType, servers, args)
		if err != nil {
			return err
		}
		if server == nil {
			return nil // Server selection message sent
		}

		// Use server key for API calls
		serverKey := server.ServerKey

		b.logger.Info("Using server for metrics",
			"server_id", server.ID,
			"server_name", server.Name,
			"server_key", serverKey)

		// Get metrics
		metrics, err := b.metricsService.GetServerMetrics(serverKey)
		if err != nil {
			b.logger.Error("Failed to get server metrics", "error", err, "server_key", serverKey)

			// Check error type and provide specific message
			errorMsg := err.Error()
			if strings.Contains(errorMsg, "not found") {
				return b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Сервер `%s` не найден.", serverKey))
			} else if strings.Contains(errorMsg, "API error") {
				return b.telegramSvc.SendMessage(ctx, chatID, fmt.Sprintf("❌ Не удалось получить метрики для сервера `%s`. Попробуйте позже.", serverKey))
			} else {
				return b.telegramSvc.SendMessage(ctx, chatID, "❌ Не удалось получить метрики. Попробуйте позже.")
			}
		}

		// Format and send metrics
		formattedMetrics := formatter(&metrics.Metrics)
		return b.telegramSvc.SendMessage(ctx, chatID, formattedMetrics)
	}

	return b.telegramSvc.SendMessage(ctx, chatID, "❌ Внутренняя ошибка сервиса. Попробуйте позже.")
}
