package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramAPI defines the interface for Telegram bot operations
type TelegramAPI interface {
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	StopReceivingUpdates()
}

// Logger defines the interface for structured logging
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, err error, fields ...Field)
	Fatal(msg string, err error, fields ...Field)
}

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// MessageHandler defines the interface for handling different message types
type MessageHandler interface {
	HandleMessage(ctx context.Context, message *tgbotapi.Message) (string, error)
	HandleCallbackQuery(ctx context.Context, query *tgbotapi.CallbackQuery) error
}

// CommandRouter defines the interface for command routing
type CommandRouter interface {
	Route(command string) (CommandHandler, bool)
	RegisterHandler(command string, handler CommandHandler)
}

// CommandHandler defines the interface for individual command handlers
type CommandHandler interface {
	Handle(ctx context.Context, message *tgbotapi.Message) (string, error)
	Description() string
	Usage() string
}

// Validator defines the interface for input validation
type Validator interface {
	ValidateServerKey(key string) error
	ValidateServerName(name string) error
	ValidateContainerID(id string) error
}

// Metrics defines the interface for metrics collection
type Metrics interface {
	IncrementCommand(command string)
	IncrementError(errorType string)
	RecordLatency(operation string, duration float64)
	RecordActiveUsers(count int64)
}
