package usecases

import (
	"context"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
)

// FallbackHandler use case для обработки неизвестных событий
type FallbackHandler struct{}

// NewFallbackHandler создает новый FallbackHandler use case
func NewFallbackHandler() *FallbackHandler {
	return &FallbackHandler{}
}

// Execute выполняет use case
func (u *FallbackHandler) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	return domain.NewTextResponse("Не могу обработать Ваше сообщение, попробуйте позднее!"), nil
}

