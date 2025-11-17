package outbound

import (
	"context"
	"fmt"

	"github.com/vlks-dev/mytelegrambotapi/bot"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// TelegramPresenter преобразует BotResponse в вызовы Telegram API
type TelegramPresenter struct {
	botAPI bot.AIBotAPI
	logger *zap.SugaredLogger
	// Для доступа к полному API нужен доступ к bot.api
	// Пока используем только AIBotAPI интерфейс
}

// NewTelegramPresenter создает новый TelegramPresenter
func NewTelegramPresenter(botAPI bot.AIBotAPI, logger *zap.SugaredLogger) *TelegramPresenter {
	return &TelegramPresenter{
		botAPI: botAPI,
		logger: logger.Named("presenter"),
	}
}

// Send отправляет BotResponse пользователю
func (p *TelegramPresenter) Send(ctx context.Context, chatID int64, response *domain.BotResponse) error {
	if response == nil {
		return nil
	}

	// Обработка callback answer
	if response.Callback != nil {
		err := p.AnswerCallbackQuery(ctx, response.Callback.QueryID, response.Callback.Text, response.Callback.ShowAlert)
		if err != nil {
			p.logger.Errorw("Failed to answer callback", "error", err)
			return err
		}
		// После ответа на callback, если есть текст, отправляем сообщение
		if response.Text != "" {
			return p.SendMessage(ctx, chatID, response.Text)
		}
		return nil
	}

	// Обработка редактирования сообщения
	if response.Edit != nil {
		err := p.EditMessage(ctx, chatID, response.Edit.MessageID, response.Edit.Text)
		if err != nil {
			p.logger.Errorw("Failed to edit message", "error", err)
			return err
		}
		return nil
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
		_, err := p.botAPI.SendMessage(chatID, response.Text)
		if err != nil {
			p.logger.Errorw("Failed to send message", "error", err)
			return err
		}
	}

	return nil
}

// SendMessage отправляет текстовое сообщение
func (p *TelegramPresenter) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := p.botAPI.SendMessage(chatID, text)
	return err
}

// SendPhoto отправляет фото
func (p *TelegramPresenter) SendPhoto(ctx context.Context, chatID int64, fileID string, caption string) error {
	// Используем tgbotapi напрямую через рефлексию или расширяем интерфейс
	// Пока заглушка, так как AIBotAPI не имеет метода SendPhoto
	p.logger.Debugf("Send photo to chat %d: %s", chatID, fileID)
	return fmt.Errorf("SendPhoto not implemented in AIBotAPI interface")
}

// SendVoice отправляет голосовое сообщение
func (p *TelegramPresenter) SendVoice(ctx context.Context, chatID int64, fileID string, caption string) error {
	// Пока заглушка
	p.logger.Debugf("Send voice to chat %d: %s", chatID, fileID)
	return fmt.Errorf("SendVoice not implemented in AIBotAPI interface")
}

// SendVideo отправляет видео
func (p *TelegramPresenter) SendVideo(ctx context.Context, chatID int64, fileID string, caption string) error {
	// Пока заглушка
	p.logger.Debugf("Send video to chat %d: %s", chatID, fileID)
	return fmt.Errorf("SendVideo not implemented in AIBotAPI interface")
}

// EditMessage редактирует сообщение
func (p *TelegramPresenter) EditMessage(ctx context.Context, chatID int64, messageID int, text string) error {
	// Пока заглушка
	p.logger.Debugf("Edit message %d in chat %d: %s", messageID, chatID, text)
	return fmt.Errorf("EditMessage not implemented in AIBotAPI interface")
}

// AnswerCallbackQuery отвечает на callback query
func (p *TelegramPresenter) AnswerCallbackQuery(ctx context.Context, queryID string, text string, showAlert bool) error {
	// Пока заглушка, так как AIBotAPI не имеет этого метода
	p.logger.Debugf("Answer callback %s: %s", queryID, text)
	// В реальной реализации нужно использовать bot.api.AnswerCallbackQuery
	return nil
}
