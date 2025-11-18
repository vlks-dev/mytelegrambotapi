package usecases

import (
	"context"
	"time"

	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"github.com/vlks-dev/mytelegrambotapi/internal/services"
	// "github.com/vlks-dev/mytelegrambotapi/utils"
	"go.uber.org/zap"
)

// ChatWithAI use case для обработки текстовых сообщений с AI
type ChatWithAI struct {
	aiService            services.AIService
	dialogHistoryService services.DialogHistoryService
	logger               *zap.SugaredLogger
}

// NewChatWithAI создает новый ChatWithAI use case
func NewChatWithAI(aiService services.AIService, dialogHistoryService services.DialogHistoryService, logger *zap.SugaredLogger) *ChatWithAI {
	return &ChatWithAI{
		aiService:            aiService,
		dialogHistoryService: dialogHistoryService,
		logger:               logger.Named("chat_with_ai"),
	}
}

// Execute выполняет use case
func (u *ChatWithAI) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	textEvent, ok := event.(domain.TextMessageReceived)
	if !ok {
		return domain.NewTextResponse("Ошибка: ожидалось текстовое сообщение"), nil
	}

	// Сохраняем входящее сообщение в историю
	message := &domain.Message{
		ChatID:       textEvent.ChatID(),
		MessageID:    textEvent.MessageID,
		FromID:       textEvent.UserID(),
		FromUsername: textEvent.Username,
		Text:         textEvent.Text,
		Timestamp:    time.Now(),
	}

	err := u.dialogHistoryService.SaveMessage(ctx, textEvent.UserID(), message)
	if err != nil {
		u.logger.Errorw("Failed to save message", "error", err)
		// Продолжаем обработку даже если не удалось сохранить
	}
	u.logger.Debugw("Saving history message", "FromUsername", message.FromUsername)

	// Получаем историю диалога
	history, err := u.dialogHistoryService.GetHistory(ctx, textEvent.UserID())
	if err != nil {
		u.logger.Errorw("Failed to get history", "error", err)
		return domain.NewTextResponse("Ошибка получения истории диалога"), err
	}

	// Добавляем текущее сообщение в историю для генерации ответа
	history = append(history, *message)

	// Генерируем ответ через AI
	answer, err := u.aiService.GenerateAnswer(ctx, history)
	if err != nil {
		u.logger.Errorw("Failed to generate answer", "error", err)
		return domain.NewTextResponse("Ошибка генерации ответа. Попробуйте позже."), err
	}

	// Сохраняем ответ бота в историю
	botMessage := &domain.Message{
		ChatID:       textEvent.ChatID(),
		MessageID:    0, // Будет установлено после отправки
		FromID:       0, // ID бота
		FromUsername: "bot",
		Text:         answer, // utils.Truncate(answer, 200)
		Timestamp:    time.Now(),
	}

	err = u.dialogHistoryService.SaveMessage(ctx, textEvent.UserID(), botMessage)
	if err != nil {
		u.logger.Warnw("Failed to save bot message", "error", err)
		// Не критично, продолжаем
	}

	return domain.NewTextResponse(answer), nil
}
