package usecases

import (
	"context"

	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"github.com/vlks-dev/mytelegrambotapi/internal/services"
	"go.uber.org/zap"
)

// ResetDialog use case для сброса контекста диалога
type ResetDialog struct {
	dialogHistoryService services.DialogHistoryService
	logger               *zap.SugaredLogger
}

// NewResetDialog создает новый ResetDialog use case
func NewResetDialog(dialogHistoryService services.DialogHistoryService, logger *zap.SugaredLogger) *ResetDialog {
	return &ResetDialog{
		dialogHistoryService: dialogHistoryService,
		logger:               logger.Named("reset_dialog"),
	}
}

// Execute выполняет use case
func (u *ResetDialog) Execute(ctx context.Context, event domain.Event) (*domain.BotResponse, error) {
	commandEvent, ok := event.(domain.CommandReceived)
	if !ok {
		return domain.NewTextResponse("Ошибка: ожидалась команда"), nil
	}

	// Сбрасываем контекст диалога
	err := u.dialogHistoryService.ResetDialogContext(ctx, commandEvent.UserID())
	if err != nil {
		u.logger.Errorw("Failed to reset dialog context", "error", err)
		return domain.NewTextResponse("Ошибка сброса контекста диалога. Попробуйте позже."), err
	}

	u.logger.Debugw("Dialog context reset",
		"user_id", commandEvent.UserID(),
	)

	return domain.NewTextResponse("✅ Контекст диалога сброшен!\n\nТеперь я не буду помнить предыдущие сообщения. Начнем новый диалог!"), nil
}
