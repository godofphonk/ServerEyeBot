package app

import (
	"context"
	"fmt"

	"github.com/servereye/servereyebot/internal/models"
	"github.com/servereye/servereyebot/internal/services"
	"github.com/servereye/servereyebot/pkg/domain"
)

// createRemoveServerKeyboard creates inline keyboard for server removal
func (b *Bot) createRemoveServerKeyboard(servers []models.ServerWithDetails) interface{} {
	var buttons [][]map[string]string

	for _, server := range servers {
		button := []map[string]string{
			{
				"text":          fmt.Sprintf("🗑️ %s", server.Server.Name),
				"callback_data": fmt.Sprintf("remove_server:%s", server.Server.ID),
			},
		}
		buttons = append(buttons, button)
	}

	return buttons
}

// handleRemoveServerCallback handles callback from remove server keyboard
func (b *Bot) handleRemoveServerCallback(ctx context.Context, query *domain.CallbackQuery) error {
	telegramID := query.From.ID
	chatID := query.Message.Chat.ID

	// Parse callback data
	data := query.Data
	if len(data) < 14 || data[:14] != "remove_server:" {
		return b.telegramSvc.AnswerCallbackQuery(ctx, query.ID, "❌ Неверный запрос")
	}

	serverID := data[14:]

	b.logger.Info("Removing server", "server_id", serverID, "telegram_id", telegramID, "chat_id", chatID)

	// Get user from database to get correct user_id
	if adapter, ok := b.userService.(*services.UserServiceAdapter); ok {
		user, err := adapter.GetUser(ctx, int64(telegramID))
		if err != nil {
			b.logger.Error("Failed to get user", "error", err, "telegram_id", telegramID)
			return b.telegramSvc.AnswerCallbackQuery(ctx, query.ID, "❌ Внутренняя ошибка")
		}

		// Remove server from user
		if err := adapter.RemoveServerFromUser(ctx, int64(user.ID), serverID); err != nil {
			b.logger.Error("Failed to remove server", "error", err, "server_id", serverID, "user_id", user.ID)
			return b.telegramSvc.AnswerCallbackQuery(ctx, query.ID, "❌ Не удалось удалить сервер")
		}

		// Answer callback and update message
		callbackMsg := fmt.Sprintf("✅ Сервер `%s` удален", serverID)
		if err := b.telegramSvc.AnswerCallbackQuery(ctx, query.ID, callbackMsg); err != nil {
			b.logger.Error("Failed to answer callback", "error", err)
		}

		// Update original message to show server was removed
		newMessage := fmt.Sprintf("🗑️ Сервер `%s` успешно удален из вашего списка.", serverID)
		return b.telegramSvc.EditMessage(ctx, chatID, query.Message.MessageID, newMessage, nil)
	}

	return b.telegramSvc.AnswerCallbackQuery(ctx, query.ID, "❌ Внутренняя ошибка сервиса")
}
