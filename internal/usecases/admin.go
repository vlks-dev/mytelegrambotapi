package usecases

import (
	"context"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"github.com/vlks-dev/mytelegrambotapi/internal/services"
)

// AdminUseCases use case для обработки административных команд
type AdminUseCases struct {
	userRepository services.UserRepository
}

// NewAdminUseCases создает новый AdminUseCases use case
func NewAdminUseCases(userRepository services.UserRepository) *AdminUseCases {
	return &AdminUseCases{
		userRepository: userRepository,
	}
}

// Execute выполняет use case
func (u *AdminUseCases) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	// Заглушка: базовая структура для расширения
	commandEvent, ok := event.(domain.CommandReceived)
	if !ok {
		return domain.NewTextResponse("Ошибка: ожидалась команда"), nil
	}

	// Проверяем права администратора
	isAdmin, err := u.userRepository.IsAdmin(ctx, commandEvent.UserID())
	if err != nil {
		return domain.NewTextResponse("Ошибка проверки прав"), err
	}

	if !isAdmin {
		return domain.NewTextResponse("У вас нет прав администратора"), nil
	}

	// Базовая структура для расширения административных команд
	return domain.NewTextResponse("Административная команда: " + commandEvent.Command), nil
}

