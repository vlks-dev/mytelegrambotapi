package bot

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mytelegrambot/config"
	"github.com/mytelegrambot/models"
	"log"
	"strconv"
)

type Storage interface {
	Save(ctx context.Context, message *models.Message) error
}

type Bot struct {
	storage Storage
	api     *tgbotapi.BotAPI
}

func NewBotApi(config *config.Config, storage Storage) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, fmt.Errorf("parsing telegram bot token err: %v", err)
	}

	botAPI.Debug = true

	return &Bot{
		storage: storage,
		api:     botAPI,
	}, nil
}

func (b *Bot) GetUpdates(ctx context.Context) error {

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 25

	updates := b.api.GetUpdatesChan(u)

	var (
		aMsg []tgbotapi.Message
		uMsg []*tgbotapi.Message
	)

	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
				uMsg = append(uMsg, update.Message)

				newMessage := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

				answerMessage, err := b.api.Send(newMessage)
				if err != nil {
					return fmt.Errorf("failed to send answer message (%v), err: %v", aMsg, err)
				}
				aMsg = append(aMsg, answerMessage)

				for _, k := range uMsg {
					updMessage := &models.Message{
						MessageID:    k.MessageID,
						FromID:       strconv.FormatInt(k.From.ID, 10),
						FromUsername: k.From.UserName,
						Text:         k.Text,
						Timestamp:    k.Time(),
					}
					err = b.storage.Save(ctx, updMessage)
					if err != nil {
						return fmt.Errorf("failed to save update message\n(%v)\n[ERROR]: %v", updMessage, err)
					}
				}

				for _, k := range aMsg {
					ansMessage := &models.Message{
						MessageID:    k.MessageID,
						FromID:       strconv.FormatInt(k.From.ID, 10),
						FromUsername: k.From.UserName,
						Text:         k.Text,
						Timestamp:    k.Time(),
					}
					err = b.storage.Save(ctx, ansMessage)
					if err != nil {
						return fmt.Errorf("failed to save messages\n(%v)\n[ERROR]: %v", ansMessage, err)
					}
				}
			}

		case <-ctx.Done():
			log.Printf("Bot stopped: %v", ctx.Err())
			return ctx.Err()
		}
	}

}

func (b *Bot) GetBotInfo() (string, error) {
	userName := b.api.Self.UserName
	firstName := b.api.Self.FirstName
	return fmt.Sprintf("bot user name: %s, first name: %v ", userName, firstName), nil
}
