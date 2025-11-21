package outbound

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/vlks-dev/mytelegrambotapi/bot"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// TelegramPresenter преобразует BotResponse в вызовы Telegram API
type TelegramPresenter struct {
	botAPI bot.ExtendedBotAPI
	logger *zap.SugaredLogger
}

// NewTelegramPresenter создает новый TelegramPresenter
func NewTelegramPresenter(botAPI bot.ExtendedBotAPI, logger *zap.SugaredLogger) *TelegramPresenter {
	return &TelegramPresenter{
		botAPI: botAPI,
		logger: logger.Named("presenter"),
	}
}

// Send отправляет BotResponse пользователю и возвращает отправленное сообщение (если есть)
func (p *TelegramPresenter) Send(ctx context.Context, chatID int64, response *domain.BotResponse) (*tgbotapi.Message, error) {
	if response == nil {
		return nil, nil
	}

	// Обработка callback answer
	if response.Callback != nil {
		if err := p.AnswerCallbackQuery(ctx, response.Callback.QueryID, response.Callback.Text, response.Callback.ShowAlert); err != nil {
			p.logger.Errorw("Failed to answer callback", "error", err)
			return nil, err
		}
		// После ответа на callback, если есть текст, отправляем сообщение
		if response.Text != "" {
			return p.SendMessage(ctx, chatID, response.Text)
		}
		return nil, nil
	}

	// Обработка редактирования сообщения
	if response.Edit != nil {
		if err := p.EditMessage(ctx, chatID, response.Edit.MessageID, response.Edit.Text); err != nil {
			p.logger.Errorw("Failed to edit message",
				"error", err,
				"chat_id", chatID,
				"message_id", response.Edit.MessageID,
			)
			return nil, err
		}
		return nil, nil
	}

	// Обработка файла
	if response.File != nil {
		switch response.File.Type {
		case domain.FileTypePhoto:
			return p.SendPhoto(ctx, chatID, response.File.FileID, response.File.Caption)
		case domain.FileTypeVoice:
			return p.SendVoice(ctx, chatID, response.File.FileID, response.File.Caption)
		case domain.FileTypeVideo:
			return p.SendVideo(ctx, chatID, response.File.FileID, response.File.Caption)
		default:
			p.logger.Warnf("Unknown file type: %s", response.File.Type)
		}
	}

	// Обработка текстового сообщения
	if response.Text != "" {
		return p.SendMessage(ctx, chatID, response.Text)
	}

	return nil, nil
}

// SendMessage отправляет текстовое сообщение и возвращает отправленное сообщение
func (p *TelegramPresenter) SendMessage(ctx context.Context, chatID int64, text string) (*tgbotapi.Message, error) {
	msg, err := p.botAPI.SendMessage(chatID, text)
	if err != nil {
		p.logger.Errorw("Failed to send message", "error", err)
		return nil, err
	}
	return msg, nil
}

// SendPhoto отправляет фото и возвращает отправленное сообщение
func (p *TelegramPresenter) SendPhoto(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error) {
	p.logger.Debugw("Sending photo",
		"chat_id", chatID,
		"file_id", fileID,
		"caption", caption,
	)
	msg, err := p.botAPI.SendPhoto(ctx, chatID, fileID, caption)
	if err != nil {
		p.logger.Errorw("Failed to send photo",
			"chat_id", chatID,
			"file_id", fileID,
			"error", err,
		)
		return nil, fmt.Errorf("send photo: %w", err)
	}
	p.logger.Debugw("Photo sent successfully", "chat_id", chatID)
	return msg, nil
}

// SendVoice отправляет голосовое сообщение и возвращает отправленное сообщение
func (p *TelegramPresenter) SendVoice(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error) {
	p.logger.Debugw("Sending voice",
		"chat_id", chatID,
		"file_id", fileID,
		"caption", caption,
	)
	msg, err := p.botAPI.SendVoice(ctx, chatID, fileID, caption)
	if err != nil {
		p.logger.Errorw("Failed to send voice",
			"chat_id", chatID,
			"file_id", fileID,
			"error", err,
		)
		return nil, fmt.Errorf("send voice: %w", err)
	}
	p.logger.Debugw("Voice sent successfully", "chat_id", chatID)
	return msg, nil
}

// SendVideo отправляет видео и возвращает отправленное сообщение
func (p *TelegramPresenter) SendVideo(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error) {
	p.logger.Debugw("Sending video",
		"chat_id", chatID,
		"file_id", fileID,
		"caption", caption,
	)
	msg, err := p.botAPI.SendVideo(ctx, chatID, fileID, caption)
	if err != nil {
		p.logger.Errorw("Failed to send video",
			"chat_id", chatID,
			"file_id", fileID,
			"error", err,
		)
		return nil, fmt.Errorf("send video: %w", err)
	}
	p.logger.Debugw("Video sent successfully", "chat_id", chatID)
	return msg, nil
}

// EditMessage редактирует сообщение
func (p *TelegramPresenter) EditMessage(ctx context.Context, chatID int64, messageID int, text string) error {
	p.logger.Debugw("Editing message",
		"chat_id", chatID,
		"message_id", messageID,
		"text_length", len(text),
	)
	_, err := p.botAPI.EditMessageText(ctx, chatID, messageID, text, nil)
	if err != nil {
		p.logger.Errorw("Failed to edit message",
			"chat_id", chatID,
			"message_id", messageID,
			"error", err,
		)
		return fmt.Errorf("edit message: %w", err)
	}
	p.logger.Debugw("Message edited successfully",
		"chat_id", chatID,
		"message_id", messageID,
	)
	return nil
}

// AnswerCallbackQuery отвечает на callback query
func (p *TelegramPresenter) AnswerCallbackQuery(ctx context.Context, queryID string, text string, showAlert bool) error {
	p.logger.Debugw("Answering callback query",
		"query_id", queryID,
		"text", text,
		"show_alert", showAlert,
	)
	err := p.botAPI.AnswerCallbackQuery(ctx, queryID, text, showAlert)
	if err != nil {
		p.logger.Errorw("Failed to answer callback query",
			"query_id", queryID,
			"error", err,
		)
		return fmt.Errorf("answer callback query: %w", err)
	}
	p.logger.Debugw("Callback query answered successfully", "query_id", queryID)
	return nil
}
