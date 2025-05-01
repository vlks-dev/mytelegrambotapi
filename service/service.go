package service

import (
	"context"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mytelegrambot/bot"
	"github.com/mytelegrambot/deepseek"
	"github.com/mytelegrambot/storage"
	"github.com/mytelegrambot/utils"
	"go.uber.org/zap"
	"log"
	"time"
)

type Service struct {
	logger  *zap.SugaredLogger
	storage storage.Storage
	r1      deepseek.R1
	bot     bot.BotAPI
}

func NewService(logger  *zap.SugaredLogger, storage storage.Storage, r1 deepseek.R1, b bot.BotAPI) *Service {
	return &Service{
		logger:  logger,
		storage: storage,
		r1:      r1,
		bot:     b,
	}
}

func (s *Service) SetBot(ctx context.Context) error {
	updates, err := s.bot.GetUpdates(ctx)
	deadline, ok := ctx.Deadline()
	if !ok {
		log.Println("get updates from tg bot, deadline is not set")
	}
	if ok {
		log.Printf("get updates from tg bot, w/ deadline (left:%v)", time.Until(deadline))
	}

	if err != nil {
		return fmt.Errorf("bot setup, get updates: %w", err)
	}

	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			s.logger.Infoln("Get update from telegram bot!")
			if err = s.ProcessMessage(ctx, update.Message); err != nil {
				msgErr, sendMsgErr := s.bot.SendMessage(update.Message.Chat.ID, "Не могу обработать Ваше сообщение, попробуйте позднее!")
				if sendMsgErr != nil {
					return fmt.Errorf("sending main failure message error: %w", sendMsgErr)
				}
				saveMsgErr := s.storage.Save(ctx, utils.BotMessageToModel(msgErr))
				if saveMsgErr != nil {
					return fmt.Errorf("saving error message: %w", saveMsgErr)
				}
				if !errors.Is(err, context.DeadlineExceeded) {
					return fmt.Errorf("processing message: %w", err)
				}
				log.Printf("processing message, context: %v", ctx.Err())

			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}

func (s *Service) ProcessMessage(ctx context.Context, msg *tgbotapi.Message) error {
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

	var _ *tgbotapi.Message

	if msg.IsCommand() {
		processCommandErr := s.processCommand(ctx, msg)
		if processCommandErr != nil {
			return fmt.Errorf("processing command: %w", processCommandErr)
		}
		return nil
	}

	mockMsg, err := s.bot.SendMessage(msg.Chat.ID, "Ваш ответ генерируется, подождите!")
	if err != nil {
		return fmt.Errorf("sending mock message: %w", err)
	}
	err = s.storage.Save(ctx, utils.BotMessageToModel(mockMsg))
	if err != nil {
		return fmt.Errorf("saving mock message: %w", err)
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err = s.getAiResponse(ctx, msg)
		// получили ответ
		if err == nil {
			break
		}
		// если вышел дедлайн, ретраим
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Attempt %d: timeout, retrying…", attempt+1)
			if attempt < maxRetries {
				continue
			}
			// все попытки исчерпаны
			answer, sendErr := s.bot.SendMessage(msg.Chat.ID,
				"Время ожидания вышло, попробуем ещё раз?")
			err := s.storage.Save(ctx, utils.BotMessageToModel(answer))
			if err != nil {
				return fmt.Errorf("saving answer: %w", err)
			}
			if sendErr != nil {
				return fmt.Errorf("sending message: %w", sendErr)
			}
			return fmt.Errorf("timeout after %d retries: %w", maxRetries+1, err)
		}

		// любая другая ошибка, вылетаем
		return fmt.Errorf("getting Ai response for (%v): %w", msg.MessageID, err)
	}

	if err = s.bot.DeleteMessage(ctx, msg.Chat.ID, mockMsg.MessageID); err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}

	return nil
}

func (s *Service) ListCommands(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var list []string

	commands, err := s.bot.GetMyCommands()
	if err != nil {
		return nil, fmt.Errorf("get commands error: %w", err)
	}

	for _, command := range commands {
		list = append(list, command.Command)
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		log.Println("handle list of commands from tg bot, deadline is not set")
	} else {
		log.Printf("handle list of commands: %v, left: %v", list, time.Until(deadline))
	}

	return list, nil
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

	for _, choice := range choices {
		if len(choices) == 0 {
			log.Printf("no response generated for q: %v", msg.MessageID)
			continue
		}
		message, err := s.bot.SendMessage(msg.Chat.ID, choice)
		if err != nil {
			return fmt.Errorf("sending answer from AI: %w", err)
		}
		err = s.storage.Save(ctx, utils.BotMessageToModel(message))
		if err != nil {
			return fmt.Errorf("saving answer message: %w", err)
		}
	}
	return nil
}

func (s *Service) processCommand(ctx context.Context, msg *tgbotapi.Message) error {
	commands, err := s.bot.GetMyCommands()
	if err != nil {
		return fmt.Errorf("getting commands: %w", err)
	}
	if len(commands) == 0 {
		log.Printf("no commands found for @%v", msg.From.UserName)
		return nil
	}
	for _, command := range commands {

		if command.Command == msg.Command() {
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
		}
	}

	return nil
}
