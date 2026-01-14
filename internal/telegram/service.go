package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/servereye/servereyebot/pkg/domain"
	"github.com/servereye/servereyebot/pkg/errors"
)

// TelegramService implements domain.TelegramService
type TelegramService struct {
	bot    *tgbotapi.BotAPI
	logger Logger
}

// Logger interface for telegram service
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// NewTelegramService creates a new telegram service
func NewTelegramService(token string, logger Logger) (*TelegramService, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.NewTelegramAPIError("failed to create bot", err)
	}

	logger.Info("Telegram bot authorized", "username", bot.Self.UserName)

	return &TelegramService{
		bot:    bot,
		logger: logger,
	}, nil
}

// SendMessage sends a message to the specified chat
func (ts *TelegramService) SendMessage(ctx context.Context, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := ts.bot.Send(msg)
	if err != nil {
		return errors.NewTelegramAPIError("failed to send message", err)
	}
	return nil
}

// SendMessageWithKeyboard sends a message with keyboard to the specified chat
func (ts *TelegramService) SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)

	switch k := keyboard.(type) {
	case *tgbotapi.ReplyKeyboardMarkup:
		msg.ReplyMarkup = k
	case *tgbotapi.InlineKeyboardMarkup:
		msg.ReplyMarkup = k
	default:
		return errors.NewValidationError("invalid keyboard type", map[string]interface{}{"type": fmt.Sprintf("%T", k)})
	}

	_, err := ts.bot.Send(msg)
	if err != nil {
		return errors.NewTelegramAPIError("failed to send message with keyboard", err)
	}
	return nil
}

// AnswerCallback answers a callback query
func (ts *TelegramService) AnswerCallback(ctx context.Context, callbackID, text string) error {
	callback := tgbotapi.NewCallback(callbackID, text)
	_, err := ts.bot.Request(callback)
	if err != nil {
		return errors.NewTelegramAPIError("failed to answer callback", err)
	}
	return nil
}

// SetCommands sets bot commands
func (ts *TelegramService) SetCommands(ctx context.Context, commands []domain.BotCommand) error {
	botCommands := make([]tgbotapi.BotCommand, len(commands))
	for i, cmd := range commands {
		botCommands[i] = tgbotapi.BotCommand{
			Command:     cmd.Command,
			Description: cmd.Description,
		}
	}

	config := tgbotapi.NewSetMyCommands(botCommands...)
	_, err := ts.bot.Request(config)
	if err != nil {
		return errors.NewTelegramAPIError("failed to set commands", err)
	}

	ts.logger.Info("Bot commands set successfully", "count", len(commands))
	return nil
}

// GetBot returns the underlying bot instance for advanced usage
func (ts *TelegramService) GetBot() *tgbotapi.BotAPI {
	return ts.bot
}

// GetUpdatesChannel returns a channel for updates
func (ts *TelegramService) GetUpdatesChannel(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return ts.bot.GetUpdatesChan(config)
}

// Request makes a generic API request
func (ts *TelegramService) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return ts.bot.Request(c)
}

// Send sends a generic chattable
func (ts *TelegramService) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	return ts.bot.Send(c)
}

// StopReceivingUpdates stops receiving updates
func (ts *TelegramService) StopReceivingUpdates() {
	ts.bot.StopReceivingUpdates()
}

// CreateKeyboard creates a reply keyboard from buttons
func CreateKeyboard(buttons ...[]string) tgbotapi.ReplyKeyboardMarkup {
	var keyboard [][]tgbotapi.KeyboardButton

	for _, row := range buttons {
		var keyboardRow []tgbotapi.KeyboardButton
		for _, buttonText := range row {
			keyboardRow = append(keyboardRow, tgbotapi.NewKeyboardButton(buttonText))
		}
		keyboard = append(keyboard, keyboardRow)
	}

	return tgbotapi.NewReplyKeyboard(keyboard...)
}

// CreateInlineKeyboard creates an inline keyboard from buttons
func CreateInlineKeyboard(buttons ...[]InlineButton) tgbotapi.InlineKeyboardMarkup {
	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton

	for _, row := range buttons {
		var keyboardRow []tgbotapi.InlineKeyboardButton
		for _, button := range row {
			keyboardRow = append(keyboardRow, tgbotapi.NewInlineKeyboardButtonData(button.Text, button.Data))
		}
		inlineKeyboard = append(inlineKeyboard, keyboardRow)
	}

	return tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...)
}

// InlineButton represents an inline keyboard button
type InlineButton struct {
	Text string
	Data string
}

// Message represents a telegram message
type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int    `json:"date"`
}

// User represents a telegram user
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Chat represents a telegram chat
type Chat struct {
	ID int64 `json:"id"`
}

// CallbackQuery represents a telegram callback query
type CallbackQuery struct {
	ID      string  `json:"id"`
	From    User    `json:"from"`
	Message Message `json:"message"`
	Data    string  `json:"data"`
}

// Update represents a telegram update
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// ConvertUpdate converts tgbotapi.Update to our domain Update
func ConvertUpdate(update tgbotapi.Update) *Update {
	result := &Update{
		UpdateID: int64(update.UpdateID),
	}

	if update.Message != nil {
		result.Message = &Message{
			MessageID: update.Message.MessageID,
			From: User{
				ID:        update.Message.From.ID,
				Username:  update.Message.From.UserName,
				FirstName: update.Message.From.FirstName,
				LastName:  update.Message.From.LastName,
			},
			Chat: Chat{
				ID: update.Message.Chat.ID,
			},
			Text: update.Message.Text,
			Date: update.Message.Date,
		}
	}

	if update.CallbackQuery != nil {
		result.CallbackQuery = &CallbackQuery{
			ID: update.CallbackQuery.ID,
			From: User{
				ID:        update.CallbackQuery.From.ID,
				Username:  update.CallbackQuery.From.UserName,
				FirstName: update.CallbackQuery.From.FirstName,
				LastName:  update.CallbackQuery.From.LastName,
			},
			Data: update.CallbackQuery.Data,
		}

		if update.CallbackQuery.Message != nil {
			result.CallbackQuery.Message = Message{
				MessageID: update.CallbackQuery.Message.MessageID,
				From: User{
					ID:        update.CallbackQuery.Message.From.ID,
					Username:  update.CallbackQuery.Message.From.UserName,
					FirstName: update.CallbackQuery.Message.From.FirstName,
					LastName:  update.CallbackQuery.Message.From.LastName,
				},
				Chat: Chat{
					ID: update.CallbackQuery.Message.Chat.ID,
				},
				Text: update.CallbackQuery.Message.Text,
				Date: update.CallbackQuery.Message.Date,
			}
		}
	}

	return result
}

// StartReceivingUpdates starts receiving updates
func (ts *TelegramService) StartReceivingUpdates(ctx context.Context, handler interface{}) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := ts.bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case update, ok := <-updates:
				if !ok {
					return
				}

				domainUpdate := ConvertUpdate(update)
				if h, ok := handler.(func(context.Context, *Update) error); ok {
					if err := h(ctx, domainUpdate); err != nil {
						ts.logger.Error("Error handling update", "error", err)
						continue
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}
