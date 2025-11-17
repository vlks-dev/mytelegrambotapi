package usecases

import (
	"context"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"github.com/vlks-dev/mytelegrambotapi/internal/services"
	"go.uber.org/zap"
	"time"
)

// ProcessVoiceMessage use case для обработки голосовых сообщений
type ProcessVoiceMessage struct {
	speechToTextService  services.SpeechToTextService
	aiService            services.AIService
	dialogHistoryService services.DialogHistoryService
	logger               *zap.SugaredLogger
}

// NewProcessVoiceMessage создает новый ProcessVoiceMessage use case
func NewProcessVoiceMessage(speechToTextService services.SpeechToTextService, aiService services.AIService, dialogHistoryService services.DialogHistoryService, logger *zap.SugaredLogger) *ProcessVoiceMessage {
	return &ProcessVoiceMessage{
		speechToTextService:  speechToTextService,
		aiService:            aiService,
		dialogHistoryService: dialogHistoryService,
		logger:               logger.Named("process_voice"),
	}
}

// Execute выполняет use case
func (u *ProcessVoiceMessage) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	voiceEvent, ok := event.(domain.VoiceReceived)
	if !ok {
		return domain.NewTextResponse("Ошибка: ожидалось голосовое сообщение"), nil
	}

	// Преобразуем голос в текст
	text, err := u.speechToTextService.Transcribe(ctx, voiceEvent.FileID)
	if err != nil {
		u.logger.Errorw("Failed to transcribe voice", "error", err)
		return domain.NewTextResponse("Ошибка преобразования голоса в текст"), err
	}

	// Сохраняем распознанный текст как сообщение
	message := &domain.Message{
		ChatID:       voiceEvent.ChatID(),
		MessageID:    voiceEvent.MessageID,
		FromID:       voiceEvent.UserID(),
		FromUsername: "",
		Text:         text,
		Timestamp:    time.Now(),
	}

	err = u.dialogHistoryService.SaveMessage(ctx, voiceEvent.UserID(), message)
	if err != nil {
		u.logger.Warnw("Failed to save transcribed message", "error", err)
	}

	// Получаем историю диалога
	history, err := u.dialogHistoryService.GetHistory(ctx, voiceEvent.UserID())
	if err != nil {
		u.logger.Errorw("Failed to get history", "error", err)
		return domain.NewTextResponse("Ошибка получения истории диалога"), err
	}

	// Добавляем текущее сообщение
	history = append(history, *message)

	// Генерируем ответ через AI
	answer, err := u.aiService.GenerateAnswer(ctx, history)
	if err != nil {
		u.logger.Errorw("Failed to generate answer", "error", err)
		return domain.NewTextResponse("Ошибка генерации ответа"), err
	}

	return domain.NewTextResponse("Текст из голоса: " + text + "\n\nОтвет: " + answer), nil
}

