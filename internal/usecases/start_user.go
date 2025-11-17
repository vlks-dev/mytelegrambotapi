package usecases

import (
	"context"
	"github.com/vlks-dev/mytelegrambotapi/bot/buttons"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"github.com/vlks-dev/mytelegrambotapi/internal/services"
	"go.uber.org/zap"
)

// StartUser use case для обработки команды /start
type StartUser struct {
	userRepository services.UserRepository
	logger         *zap.SugaredLogger
}

// NewStartUser создает новый StartUser use case
func NewStartUser(userRepository services.UserRepository, logger *zap.SugaredLogger) *StartUser {
	return &StartUser{
		userRepository: userRepository,
		logger:          logger.Named("start_user"),
	}
}

// Execute выполняет use case
func (u *StartUser) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	commandEvent, ok := event.(domain.CommandReceived)
	if !ok {
		return domain.NewTextResponse("Ошибка: ожидалась команда"), nil
	}

	// Получаем или создаем пользователя
	user, err := u.userRepository.GetByID(ctx, commandEvent.UserID())
	if err != nil {
		u.logger.Errorw("Failed to get user", "error", err)
		return domain.NewTextResponse("Ошибка получения данных пользователя"), err
	}

	// Устанавливаем начальное состояние
	user.SetState("default")
	err = u.userRepository.Save(ctx, user)
	if err != nil {
		u.logger.Errorw("Failed to save user", "error", err)
	}

	// Создаем клавиатуру
	keyboard := buttons.InitKeyboard()

	// Возвращаем приветственное сообщение с клавиатурой
	return domain.NewTextResponseWithKeyboard(
		"Привет! Задавай мне вопросы, а я постараюсь ответить на них правильно! (на базе DeepSeek v3)",
		&keyboard,
	), nil
}

