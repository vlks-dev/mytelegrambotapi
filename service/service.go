package service

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mytelegrambot/bot"
	"github.com/mytelegrambot/deepseek"
	"github.com/mytelegrambot/models"
	"github.com/mytelegrambot/storage"
	"log"
	"time"
)

type Service struct {
	storage storage.Storage
	r1      deepseek.R1
	bot     bot.BotAPI
}

func NewService(storage storage.Storage, r1 deepseek.R1, b bot.BotAPI) *Service {
	return &Service{
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
	log.Printf("get updates from tg bot, w/ deadline (left:%v)", time.Until(deadline))
	if err != nil {
		return fmt.Errorf("bot setup, get updates: %w", err)
	}

	for {
		select {
		case update := <-updates:
			if update.Message == nil {
				continue
			}
			if err = s.ProcessMessage(ctx, update.Message); err != nil {
				_, err = s.bot.SendMessage(update.Message.Chat.ID, "Не могу обработать Ваше сообщение, попробуйте позднее!")
				if err != nil {
					return fmt.Errorf("sending main failure message error: %w", err)
				}
				log.Printf("Error processing message: %v", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}

func (s *Service) ProcessMessage(ctx context.Context, msg *tgbotapi.Message) error {
	const maxRunes = 200

	message := models.Message{
		ChatID:       msg.Chat.ID,
		MessageID:    msg.MessageID,
		FromID:       msg.From.ID,
		FromUsername: msg.From.UserName,
		Text:         truncate(msg.Text, maxRunes),
		Timestamp:    time.Time{},
	}

	err := s.storage.Save(ctx, &message)
	if err != nil {
		return fmt.Errorf(
			"saving message (%v) from chat (%v): %w",
			message.MessageID,
			message.ChatID,
			err,
		)
	}

	if msg.IsCommand() {
		return s.bot.HandleCommand(ctx, msg)
	}
	return s.bot.HandleRegularMessage(ctx, msg)
}

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

func (s *Service) ListCommands(ctx context.Context, msg *tgbotapi.Message) error {

	log.Printf("handling command: %s", msg.Command())

	return nil
}

func (s *Service) handleRegularMessage(ctx context.Context, msg *tgbotapi.Message) error {
	log.Printf("handling regular message: %s", msg.Command())
	return nil
}
