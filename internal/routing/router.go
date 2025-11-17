package routing

import (
	"context"

	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// UseCaseHandler представляет обработчик use case
type UseCaseHandler interface {
	Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error)
}

// EventRouter маршрутизирует события по соответствующим use cases
type EventRouter struct {
	commandHandlers  map[string]UseCaseHandler
	callbackHandlers map[string]UseCaseHandler
	messageHandler   UseCaseHandler
	voiceHandler     UseCaseHandler
	videoHandler     UseCaseHandler
	fallbackHandler  UseCaseHandler
	logger           *zap.SugaredLogger
}

// NewEventRouter создает новый EventRouter
func NewEventRouter(logger *zap.SugaredLogger) *EventRouter {
	return &EventRouter{
		commandHandlers:  make(map[string]UseCaseHandler),
		callbackHandlers: make(map[string]UseCaseHandler),
		logger:           logger.Named("router"),
	}
}

// RegisterCommandHandler регистрирует обработчик для команды
func (r *EventRouter) RegisterCommandHandler(command string, handler UseCaseHandler) {
	r.commandHandlers[command] = handler
	r.logger.Debugw("Registered command handler", "command", command)
}

// RegisterCallbackHandler регистрирует обработчик для callback
func (r *EventRouter) RegisterCallbackHandler(callback string, handler UseCaseHandler) {
	r.callbackHandlers[callback] = handler
	r.logger.Debugw("Registered callback handler", "callback", callback)
}

// RegisterMessageHandler регистрирует обработчик для текстовых сообщений
func (r *EventRouter) RegisterMessageHandler(handler UseCaseHandler) {
	r.messageHandler = handler
	r.logger.Debug("Registered message handler")
}

// RegisterVoiceHandler регистрирует обработчик для голосовых сообщений
func (r *EventRouter) RegisterVoiceHandler(handler UseCaseHandler) {
	r.voiceHandler = handler
	r.logger.Debug("Registered voice handler")
}

// RegisterVideoHandler регистрирует обработчик для видео
func (r *EventRouter) RegisterVideoHandler(handler UseCaseHandler) {
	r.videoHandler = handler
	r.logger.Debug("Registered video handler")
}

// RegisterFallbackHandler регистрирует обработчик для неизвестных событий
func (r *EventRouter) RegisterFallbackHandler(handler UseCaseHandler) {
	r.fallbackHandler = handler
	r.logger.Debug("Registered fallback handler")
}

// Route определяет обработчик для события и возвращает его
func (r *EventRouter) Route(ctx context.Context, event domain.Event) (UseCaseHandler, error) {
	if event == nil {
		r.logger.Warn("Received nil event, using fallback handler")
		if r.fallbackHandler != nil {
			return r.fallbackHandler, nil
		}
		return nil, nil
	}

	eventType := event.Type()
	chatID := event.ChatID()
	userID := event.UserID()

	switch e := event.(type) {
	case domain.CommandReceived:
		handler, ok := r.commandHandlers[e.Command]
		if ok {
			r.logger.Debugw("Routed command to handler",
				"command", e.Command,
				"chat_id", chatID,
				"user_id", userID,
			)
			return handler, nil
		}
		// Проверка на admin команды
		if len(e.Command) > 6 && e.Command[:6] == "admin_" {
			handler, ok := r.commandHandlers["admin"]
			if ok {
				r.logger.Debugw("Routed admin command to handler",
					"command", e.Command,
					"chat_id", chatID,
					"user_id", userID,
				)
				return handler, nil
			}
		}
		r.logger.Warnw("No handler found for command, using fallback",
			"command", e.Command,
			"chat_id", chatID,
			"user_id", userID,
		)
		return r.fallbackHandler, nil

	case domain.CallbackReceived:
		handler, ok := r.callbackHandlers[e.Data]
		if ok {
			r.logger.Debugw("Routed callback to handler",
				"data", e.Data,
				"chat_id", chatID,
				"user_id", userID,
			)
			return handler, nil
		}
		r.logger.Warnw("No handler found for callback, using fallback",
			"data", e.Data,
			"chat_id", chatID,
			"user_id", userID,
		)
		return r.fallbackHandler, nil

	case domain.TextMessageReceived:
		if r.messageHandler != nil {
			r.logger.Debugw("Routed text message to handler",
				"chat_id", chatID,
				"user_id", userID,
			)
			return r.messageHandler, nil
		}
		r.logger.Warnw("No message handler registered, using fallback",
			"chat_id", chatID,
			"user_id", userID,
		)
		return r.fallbackHandler, nil

	case domain.VoiceReceived:
		if r.voiceHandler != nil {
			r.logger.Debugw("Routed voice message to handler",
				"chat_id", chatID,
				"user_id", userID,
			)
			return r.voiceHandler, nil
		}
		r.logger.Warnw("No voice handler registered, using fallback",
			"chat_id", chatID,
			"user_id", userID,
		)
		return r.fallbackHandler, nil

	case domain.VideoReceived:
		if r.videoHandler != nil {
			r.logger.Debugw("Routed video message to handler",
				"chat_id", chatID,
				"user_id", userID,
			)
			return r.videoHandler, nil
		}
		r.logger.Warnw("No video handler registered, using fallback",
			"chat_id", chatID,
			"user_id", userID,
		)
		return r.fallbackHandler, nil

	default:
		r.logger.Warnw("Unknown event type, using fallback",
			"event_type", eventType,
			"chat_id", chatID,
			"user_id", userID,
		)
		return r.fallbackHandler, nil
	}
}
