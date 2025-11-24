package inbound

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// TelegramGateway обрабатывает updates от Telegram и нормализует их в доменные события
type TelegramGateway struct {
	logger *zap.SugaredLogger
}

// NewTelegramGateway создает новый TelegramGateway
func NewTelegramGateway(logger *zap.SugaredLogger) *TelegramGateway {
	return &TelegramGateway{
		logger: logger.Named("gateway"),
	}
}

// ProcessUpdate обрабатывает update от Telegram и преобразует его в доменное событие
func (g *TelegramGateway) ProcessUpdate(update tgbotapi.Update) domain.Event {
	// Обработка callback query
	if update.CallbackQuery != nil {
		callback := update.CallbackQuery
		var event domain.Event
		// Проверяем наличие сообщения в callback
		if callback.Message != nil {
			event = domain.NewCallbackReceived(
				callback.Message.Chat.ID,
				callback.From.ID,
				callback.Data,
				callback.ID,
				callback.Message.MessageID,
			)
			g.logger.Debugw("Processed callback query",
				"chat_id", callback.Message.Chat.ID,
				"user_id", callback.From.ID,
				"data", callback.Data,
				"query_id", callback.ID,
			)
		} else {
			// Если сообщения нет, используем chat ID из From (для inline queries)
			event = domain.NewCallbackReceived(
				callback.From.ID,
				callback.From.ID,
				callback.Data,
				callback.ID,
				0, // Нет message ID для inline queries
			)
			g.logger.Debugw("Processed inline callback query",
				"user_id", callback.From.ID,
				"data", callback.Data,
				"query_id", callback.ID,
			)
		}
		return event
	}

	// Обработка сообщений
	if update.Message == nil {
		g.logger.Debugw("Received update without message or callback")
		return nil
	}

	msg := update.Message
	chatID := msg.Chat.ID
	userID := msg.From.ID
	messageID := msg.MessageID

	// Обработка команд
	if msg.IsCommand() {
		event := domain.NewCommandReceived(
			chatID,
			userID,
			msg.Command(),
			msg.CommandArguments(),
			messageID,
		)
		g.logger.Infow("Processed command",
			"chat_id", chatID,
			"user_id", userID,
			"command", msg.Command(),
			"arguments", msg.CommandArguments(),
		)
		return event
	}

	// Обработка голосовых сообщений
	if msg.Voice != nil {
		username := ""
		if msg.From != nil && msg.From.UserName != "" {
			username = msg.From.UserName
		}
		event := domain.NewVoiceReceived(
			chatID,
			userID,
			username,
			msg.Voice.FileID,
			msg.Voice.Duration,
			messageID,
		)
		g.logger.Infow("Processed voice message",
			"chat_id", chatID,
			"user_id", userID,
			"username", username,
			"file_id", msg.Voice.FileID,
			"duration", msg.Voice.Duration,
		)
		return event
	}

	// Обработка видео
	if msg.Video != nil {
		event := domain.NewVideoReceived(
			chatID,
			userID,
			msg.Video.FileID,
			msg.Video.Duration,
			messageID,
		)
		g.logger.Infow("Processed video message",
			"chat_id", chatID,
			"user_id", userID,
			"file_id", msg.Video.FileID,
			"duration", msg.Video.Duration,
		)
		return event
	}

	// Обработка текстовых сообщений
	if msg.Text != "" {
		username := ""
		if msg.From != nil {
			username = msg.From.UserName
		}
		event := domain.NewTextMessageReceived(
			chatID,
			userID,
			msg.Text,
			username,
			messageID,
		)
		g.logger.Infow("Processed text message",
			"chat_id", chatID,
			"user_id", userID,
			"username", username,
			"text_length", len(msg.Text),
		)
		return event
	}

	g.logger.Debugw("Received message without recognized content",
		"chat_id", chatID,
		"user_id", userID,
		"message_id", messageID,
	)
	return nil
}
