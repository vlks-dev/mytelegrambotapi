package usecases

import (
	"context"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
)

// HandleCallback use case для обработки callback queries
type HandleCallback struct{}

// NewHandleCallback создает новый HandleCallback use case
func NewHandleCallback() *HandleCallback {
	return &HandleCallback{}
}

// Execute выполняет use case
func (u *HandleCallback) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	// Заглушка: обрабатываем callback
	callbackEvent, ok := event.(domain.CallbackReceived)
	if !ok {
		return domain.NewTextResponse("Ошибка: ожидался callback"), nil
	}

	// Отвечаем на callback query
	response := domain.NewCallbackAnswer(callbackEvent.QueryID, "Обработано", false)
	return response, nil
}

