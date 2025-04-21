package bot

import (
	"context"
	"encoding/json"
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

type R1 interface {
	AnswerQuestion(ctx context.Context, message string) (string, error)
}

type Bot struct {
	storage Storage
	r1      R1
	api     *tgbotapi.BotAPI
}

func NewBot(config *config.Config, storage Storage, r1 R1) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, fmt.Errorf("parsing telegram bot token err: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 25

	botAPI.Debug = true

	return &Bot{
		storage: storage,
		r1:      r1,
		api:     botAPI,
	}, nil
}

func (b *Bot) GetUpdates(ctx context.Context) error {

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 25

	updates := b.api.GetUpdatesChan(u)

	var (
		sent     []tgbotapi.Message
		incoming []*tgbotapi.Message
	)

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				log.Println("update channel closed (can't get updates)")
				return nil
			}
			if update.Message != nil {
				log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
				incoming = append(incoming, update.Message)

				for _, msg := range incoming {
					updMessage := &models.Message{
						MessageID:    msg.MessageID,
						FromID:       strconv.FormatInt(msg.From.ID, 10),
						FromUsername: msg.From.UserName,
						Text:         msg.Text,
						Timestamp:    msg.Time(),
					}
					err := b.storage.Save(ctx, updMessage)
					if err != nil {
						return fmt.Errorf("failed to save update message\n(%v)\n[ERROR]: %w", updMessage, err)
					}
					updMessage = nil
				}

				answer, err := b.r1.AnswerQuestion(ctx, update.Message.Text)
				if err != nil {
					return fmt.Errorf("failed to answer on: %v\n[ERROR]: %w", update.Message.Text, err)
				}

				data := []byte(answer)
				var response models.CompletionResponse
				var errMsg models.ErrorR1Message

				err = json.Unmarshal(data, &response)
				if err != nil {
					return fmt.Errorf("failed to unmarshal answer on: %v\n[ERROR]: %w", update.Message.Text, err)
				}
				if response.Choices == nil || len(response.Choices) == 0 {
					err = json.Unmarshal(data, &errMsg)
					if err != nil {
						return fmt.Errorf("there is an error in AI service (CHECK TOKENS IF NOT SURE)\nfailed to unmarshal answer on: %v\n[ERROR]: %w", data, err)
					}
					errMessage := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("[ERROR]: Code: %v, Message: %v", errMsg.Error.Code, errMsg.Error.Message))
					message, err := b.api.Send(errMessage)
					if err != nil {
						return fmt.Errorf("failed to send error message\n(%v)\n[ERROR]: %w", errMessage, err)
					}
					fmt.Println(message)
				}

				for _, msg := range response.Choices {
					r1Content := msg.Message.Content
					newMessage := tgbotapi.NewMessage(update.Message.Chat.ID, r1Content)
					answerMessage, err := b.api.Send(newMessage)
					if err != nil {
						return fmt.Errorf("failed to send answer (%v), err: %w", sent, err)
					}
					sent = append(sent, answerMessage)
				}

				for _, msg := range sent {
					text := msg.Text
					parsedRunes := []rune(text)
					if len(parsedRunes) <= 200 {
						text = string(parsedRunes)
					} else {
						fmt.Printf("[WARNING]: answer message too large (%d), saving beginnig...\n", len(msg.Text))
						text = string(parsedRunes[:200])
					}

					ansMessage := &models.Message{
						MessageID:    msg.MessageID,
						FromID:       strconv.FormatInt(msg.From.ID, 10),
						FromUsername: msg.From.UserName,
						Text:         text,
						Timestamp:    msg.Time(),
					}

					err = b.storage.Save(ctx, ansMessage)
					if err != nil {
						return fmt.Errorf("failed to save answers\n(%v)\n[ERROR]: %w", ansMessage, err)
					}
					ansMessage = nil
				}
			}

		case <-ctx.Done():
			log.Printf("Bot stopped: %v", ctx.Err())
			return ctx.Err()
		}
	}

}

func (b *Bot) GetBotInfo() (string, error) {
	return fmt.Sprintf("bot: @%s (%s)", b.api.Self.UserName, b.api.Self.FirstName), nil
}
