package usecases

import (
	"context"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// HelpUser use case для обработки команды /help
type HelpUser struct {
	logger *zap.SugaredLogger
}

// NewHelpUser создает новый HelpUser use case
func NewHelpUser(logger *zap.SugaredLogger) *HelpUser {
	return &HelpUser{
		logger: logger.Named("help_user"),
	}
}

// Execute выполняет use case
func (u *HelpUser) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	helpText := "Я Простой чат-бот на основе Openai API, написанный на Golang, с используемой моделью - DeepSeek V3.\n\n" +
		"Доступные команды:\n" +
		"/start - Начать работу с ботом\n" +
		"/help - Показать эту справку\n\n" +
		"Просто отправьте мне текстовое сообщение, и я постараюсь ответить на ваш вопрос!"
	return domain.NewTextResponse(helpText), nil
}

