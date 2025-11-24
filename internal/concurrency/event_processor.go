package concurrency

import (
	"context"

	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"github.com/vlks-dev/mytelegrambotapi/internal/outbound"
	"github.com/vlks-dev/mytelegrambotapi/internal/routing"
	"go.uber.org/zap"
)

// eventProcessor обрабатывает события через router и presenter
type eventProcessor struct {
	router    *routing.EventRouter
	presenter *outbound.TelegramPresenter
	logger    *zap.SugaredLogger
}

// NewEventProcessor создает новый EventProcessor
func NewEventProcessor(router *routing.EventRouter, presenter *outbound.TelegramPresenter, logger *zap.SugaredLogger) EventProcessor {
	return &eventProcessor{
		router:    router,
		presenter: presenter,
		logger:    logger.Named("event_processor"),
	}
}

// Process обрабатывает событие
func (ep *eventProcessor) Process(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	eventType := event.Type()
	chatID := event.ChatID()
	userID := event.UserID()

	ep.logger.Debugw("Processing event",
		"event_type", eventType,
		"chat_id", chatID,
		"user_id", userID,
	)

	// Получаем обработчик из router
	handler, err := ep.router.Route(ctx, event)
	if err != nil {
		ep.logger.Errorw("Failed to route event",
			"error", err,
			"event_type", eventType,
			"chat_id", chatID,
			"user_id", userID,
		)
		return nil, err
	}

	if handler == nil {
		ep.logger.Warnw("No handler found for event",
			"event_type", eventType,
			"chat_id", chatID,
			"user_id", userID,
		)
		return domain.NewTextResponse("Не могу обработать Ваше сообщение"), nil
	}

	// Выполняем use case
	response, err := handler.Execute(ctx, event)
	if err != nil {
		ep.logger.Errorw("Failed to execute use case",
			"error", err,
			"event_type", eventType,
			"chat_id", chatID,
			"user_id", userID,
		)
		return nil, err
	}

	// Возвращаем response; финальная отправка/редактирование выполняется выше (в месте приема Response из workerPool)
	if response == nil {
		ep.logger.Warnw("Use case returned nil response",
			"event_type", eventType,
			"chat_id", chatID,
			"user_id", userID,
		)
	} else {
		ep.logger.Debugw("Use case produced a response",
			"event_type", eventType,
			"chat_id", chatID,
			"user_id", userID,
		)
	}

	return response, nil
}
