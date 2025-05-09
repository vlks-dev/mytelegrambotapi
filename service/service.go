package service

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mytelegrambot/bot"
	"github.com/mytelegrambot/deepseek"
	"github.com/mytelegrambot/middleware"
	"github.com/mytelegrambot/storage"
	"github.com/mytelegrambot/utils"
	"go.uber.org/zap"
)

type Service struct {
	logger  *zap.SugaredLogger
	storage storage.Storage
	r1      deepseek.R1
	bot     bot.AIBotAPI
}

func NewService(logger *zap.SugaredLogger, storage storage.Storage, r1 deepseek.R1, b bot.AIBotAPI) *Service {
	log := logger.Named("service")
	return &Service{
		logger:  log,
		storage: storage,
		r1:      r1,
		bot:     b,
	}
}

func (s *Service) SetBot(ctx context.Context) error {
	updates, err := s.bot.GetUpdates(ctx)
	s.logger.Infow("get updates")

	if err != nil {
		return fmt.Errorf("bot setup, get updates: %w", err)
	}

	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				s.logger.Warn("updates channel closed")
				return nil
			}
			s.logger.Infoln("get update from telegram bot!")
			if err = s.processMessage(ctx, update.Message); err != nil {
				mock, ssErr := s.sendAndSave(ctx, update.Message.Chat.ID, "Не могу обработать Ваше сообщение, попробуйте позднее!")
				if !errors.Is(err, context.DeadlineExceeded) {
					s.logger.Errorw("failed to send and save mock message", "message", update.Message, "chat id", update.Message.Chat.ID, "err", err)
					return fmt.Errorf("processing message: %w", ssErr)
				}
				s.logger.Warnw(
					"processing message",
					"chat id",
					mock.Chat.ID,
					"text",
					mock.Text,
				)

			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}

func (s *Service) processMessage(ctx context.Context, msg *tgbotapi.Message) error {
	const maxRetries = 2

	err := s.storage.Save(ctx, utils.BotMessageToModel(msg))
	if err != nil {
		return fmt.Errorf(
			"saving message (%v) from chat (%v): %w",
			msg.MessageID,
			msg.Chat.ID,
			err,
		)
	}

	if msg.IsCommand() {
		processCommandErr := s.processCommand(ctx, msg)
		if processCommandErr != nil {
			return fmt.Errorf("processing command: %w", processCommandErr)
		}
		return nil
	}

	mock, err := s.sendAndSave(ctx, msg.Chat.ID, "Ваш ответ генерируется, подождите!")
	if err != nil {

		s.logger.Warnw("failed send and save mock message", "message", mock, "chat id", msg.Chat.ID, "err", err)
		return err
	}

	responder := middleware.RetryMiddleware(s.logger, maxRetries, 0)(middleware.LoggingMiddleware(s.logger)(s.getAiResponse))

	if responderErr := responder(ctx, msg); responderErr != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			s.logger.Warnw("AI response with middleware failed", "max retries", maxRetries, "chat id", msg.Chat.ID, "err", err.Error())
		}
		save, ssErr := s.sendAndSave(ctx, msg.Chat.ID, "Время ожидания вышло\nПовторите вопрос или задайте новый!")
		if ssErr != nil {
			s.logger.Errorw("failed send and save retry question", "message", save, "chat id", msg.Chat.ID, "err", err)
			return ssErr
		}
	}

	if err = s.bot.DeleteMessage(ctx, msg.Chat.ID, mock.MessageID); err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}

	return nil
}

func (s *Service) getAiResponse(ctx context.Context, msg *tgbotapi.Message) error {
	answerQuestion, err := s.r1.AnswerQuestion(ctx, msg.Text)
	if err != nil {
		return fmt.Errorf("getting answer question: %w", err)
	}

	choices, err := utils.ParseChoices(answerQuestion)
	if err != nil {
		return fmt.Errorf("parsing answer question: %w", err)
	}

	if len(choices) == 0 {
		s.logger.Warnf("no response generated for q: %v", msg.MessageID)
		return nil
	}

	for _, choice := range choices {
		save, ssErr := s.sendAndSave(ctx, msg.Chat.ID, choice)
		if ssErr != nil {
			s.logger.Errorf("sending to (%v), and saving (%v): %v", msg.Chat.ID, choice, ssErr)
			return ssErr
		}
		s.logger.Debugf("saving (%v), and sending to: (%v)", utils.Truncate(save.Text, 10), msg.MessageID)
	}

	s.logger.Debugf("AI response sent for msg %v", msg.MessageID)
	return nil
}

func (s *Service) processCommand(ctx context.Context, msg *tgbotapi.Message) error {
	var msgIDs []int
	if msg.Command() == "restart" {
		dbIDs, err := s.storage.GetMsgIDs(ctx, msg.Chat.ID)
		if err != nil {
			return fmt.Errorf("getting msg ids: %w", err)
		}
		msgIDs = dbIDs
	}
	handleCommand, err := s.bot.HandleCommand(ctx, msg, msgIDs)
	if err != nil {
		return fmt.Errorf("handling command: %w", err)
	}
	if handleCommand == nil {
		ok, err := s.storage.MoveToRecover(ctx, msg.Chat.ID)
		if err != nil {
			return fmt.Errorf("moving recovery message: %w", err)
		}
		if !ok {
			return nil
		}
	} else {
		err = s.storage.Save(ctx, utils.BotMessageToModel(handleCommand))
		if err != nil {
			return fmt.Errorf("saving handled command: %w", err)
		}
	}

	return nil
}

func (s *Service) sendAndSave(ctx context.Context, chatID int64, text string) (*tgbotapi.Message, error) {
	s.logger.Debugw("sending message", "chat_id", chatID, "text", text)
	msg, err := s.bot.SendMessage(chatID, text)
	if err != nil {
		s.logger.Errorw("failed to send message", "chatID", chatID, "text", text, "error", err)
		return nil, fmt.Errorf("sending message: %w", err)
	}
	if err := s.storage.Save(ctx, utils.BotMessageToModel(msg)); err != nil {
		s.logger.Errorw("failed to save message", "msgID", msg.MessageID, "chatID", chatID, "error", err)
		return nil, fmt.Errorf("saving message: %w", err)
	}
	s.logger.Debugw("message sent and saved", "msgID", msg.MessageID, "chatID", chatID)
	return msg, nil
}
