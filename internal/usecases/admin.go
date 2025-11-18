package usecases

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"github.com/vlks-dev/mytelegrambotapi/internal/services"
	"go.uber.org/zap"
)

// AdminUseCases use case для обработки административных команд
type AdminUseCases struct {
	userRepository services.UserRepository
	logger         *zap.SugaredLogger
}

// NewAdminUseCases создает новый AdminUseCases use case
func NewAdminUseCases(userRepository services.UserRepository, logger *zap.SugaredLogger) *AdminUseCases {
	return &AdminUseCases{
		userRepository: userRepository,
		logger:         logger.Named("admin_usecases"),
	}
}

// Execute выполняет use case
func (u *AdminUseCases) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	commandEvent, ok := event.(domain.CommandReceived)
	if !ok {
		return domain.NewTextResponse("Ошибка: ожидалась команда"), nil
	}

	// Проверяем права администратора
	isAdmin, err := u.userRepository.IsAdmin(ctx, commandEvent.UserID())
	if err != nil {
		u.logger.Errorw("Failed to check admin rights",
			"user_id", commandEvent.UserID(),
			"error", err,
		)
		return domain.NewTextResponse("Ошибка проверки прав"), err
	}

	if !isAdmin {
		u.logger.Warnw("Non-admin user tried to execute admin command",
			"user_id", commandEvent.UserID(),
			"command", commandEvent.Command,
		)
		return domain.NewTextResponse("У вас нет прав администратора"), nil
	}

	u.logger.Infow("Admin command executed",
		"user_id", commandEvent.UserID(),
		"command", commandEvent.Command,
		"arguments", commandEvent.Arguments,
	)

	// Обработка различных административных команд
	command := strings.ToLower(commandEvent.Command)

	// Извлекаем подкоманду из admin_* команды
	if strings.HasPrefix(command, "admin_") {
		subCommand := strings.TrimPrefix(command, "admin_")
		return u.handleAdminSubCommand(ctx, subCommand, commandEvent)
	}

	// Обработка команды /admin
	if command == "admin" {
		return u.handleAdminHelp(ctx, commandEvent)
	}

	return domain.NewTextResponse("Неизвестная административная команда: " + commandEvent.Command), nil
}

// handleAdminSubCommand обрабатывает подкоманды admin_*
func (u *AdminUseCases) handleAdminSubCommand(ctx context.Context, subCommand string, event domain.CommandReceived) (*domain.BotResponse, error) {
	switch subCommand {
	case "help":
		return u.handleAdminHelp(ctx, event)
	case "stats":
		return u.handleAdminStats(ctx, event)
	case "users":
		return u.handleAdminUsers(ctx, event)
	default:
		return domain.NewTextResponse(fmt.Sprintf("Неизвестная подкоманда: %s\nИспользуйте /admin_help для справки", subCommand)), nil
	}
}

// handleAdminHelp показывает справку по административным командам
func (u *AdminUseCases) handleAdminHelp(ctx context.Context, event domain.CommandReceived) (*domain.BotResponse, error) {
	helpText := "📋 Административные команды:\n\n" +
		"/admin_help - Показать эту справку\n" +
		"/admin_stats - Статистика бота\n" +
		"/admin_users - Информация о пользователях\n\n" +
		"Все команды требуют прав администратора."
	return domain.NewTextResponse(helpText), nil
}

// handleAdminStats показывает статистику (заглушка для расширения)
func (u *AdminUseCases) handleAdminStats(ctx context.Context, event domain.CommandReceived) (*domain.BotResponse, error) {
	u.logger.Debugw("Admin stats requested", "user_id", event.UserID())
	// TODO: реализовать сбор статистики
	statsText := "📊 Статистика бота:\n\n" +
		"Функция в разработке.\n" +
		"Здесь будет отображаться статистика использования бота."
	return domain.NewTextResponse(statsText), nil
}

// handleAdminUsers показывает информацию о пользователях (заглушка для расширения)
func (u *AdminUseCases) handleAdminUsers(ctx context.Context, event domain.CommandReceived) (*domain.BotResponse, error) {
	u.logger.Debugw("Admin users info requested", "user_id", event.UserID())

	// Если указан ID пользователя в аргументах
	if event.Arguments != "" {
		userID, err := strconv.ParseInt(strings.TrimSpace(event.Arguments), 10, 64)
		if err != nil {
			return domain.NewTextResponse("Неверный формат ID пользователя"), nil
		}

		user, err := u.userRepository.GetByID(ctx, userID)
		if err != nil {
			return domain.NewTextResponse(fmt.Sprintf("Ошибка получения пользователя: %v", err)), err
		}

		isAdmin, _ := u.userRepository.IsAdmin(ctx, userID)
		adminStatus := "Нет"
		if isAdmin {
			adminStatus = "Да"
		}

		userInfo := fmt.Sprintf("👤 Информация о пользователе:\n\n"+
			"ID: %d\n"+
			"Состояние: %s\n"+
			"Администратор: %s",
			user.ID, user.GetState(), adminStatus)

		return domain.NewTextResponse(userInfo), nil
	}

	// TODO: реализовать список всех пользователей
	usersText := "👥 Пользователи:\n\n" +
		"Функция в разработке.\n" +
		"Используйте /admin_users <user_id> для получения информации о конкретном пользователе."
	return domain.NewTextResponse(usersText), nil
}
