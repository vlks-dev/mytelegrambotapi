package usecases

import (
	"context"
	"strings"

	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// HandleCallback use case для обработки callback queries
type HandleCallback struct {
	logger *zap.SugaredLogger
}

// NewHandleCallback создает новый HandleCallback use case
func NewHandleCallback(logger *zap.SugaredLogger) *HandleCallback {
	return &HandleCallback{
		logger: logger.Named("handle_callback"),
	}
}

// Execute выполняет use case
func (u *HandleCallback) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	callbackEvent, ok := event.(domain.CallbackReceived)
	if !ok {
		u.logger.Warn("Received non-callback event in HandleCallback")
		return domain.NewTextResponse("Ошибка: ожидался callback"), nil
	}

	u.logger.Debugw("Processing callback",
		"query_id", callbackEvent.QueryID,
		"data", callbackEvent.Data,
		"chat_id", callbackEvent.ChatID(),
		"user_id", callbackEvent.UserID(),
	)

	// Парсим callback data (формат: "action:param1:param2")
	parts := strings.Split(callbackEvent.Data, ":")
	if len(parts) == 0 {
		u.logger.Warnw("Empty callback data",
			"query_id", callbackEvent.QueryID,
		)
		return domain.NewCallbackAnswer(callbackEvent.QueryID, "Ошибка: пустые данные", false), nil
	}

	action := parts[0]

	// Обработка различных действий
	switch action {
	case "button_click":
		// Обработка нажатия кнопки
		u.logger.Debugw("Button clicked",
			"query_id", callbackEvent.QueryID,
			"data", callbackEvent.Data,
		)
		return domain.NewCallbackAnswer(callbackEvent.QueryID, "Кнопка нажата", false), nil

	case "confirm":
		// Подтверждение действия
		u.logger.Debugw("Action confirmed",
			"query_id", callbackEvent.QueryID,
			"data", callbackEvent.Data,
		)
		return domain.NewCallbackAnswer(callbackEvent.QueryID, "Действие подтверждено", false), nil

	case "cancel":
		// Отмена действия
		u.logger.Debugw("Action cancelled",
			"query_id", callbackEvent.QueryID,
			"data", callbackEvent.Data,
		)
		return domain.NewCallbackAnswer(callbackEvent.QueryID, "Действие отменено", false), nil

	default:
		u.logger.Warnw("Unknown callback action",
			"query_id", callbackEvent.QueryID,
			"action", action,
			"data", callbackEvent.Data,
		)
		return domain.NewCallbackAnswer(callbackEvent.QueryID, "Неизвестное действие", false), nil
	}
}
