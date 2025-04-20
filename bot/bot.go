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

func NewBotApi(config *config.Config, storage Storage, r1 R1) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, fmt.Errorf("parsing telegram bot token err: %v", err)
	}

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
		aMsg []tgbotapi.Message
		uMsg []*tgbotapi.Message
	)

	for {
		select {
		case update := <-updates:
			if update.Message != nil {
				log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
				uMsg = append(uMsg, update.Message)

				for _, k := range uMsg {
					updMessage := &models.Message{
						MessageID:    k.MessageID,
						FromID:       strconv.FormatInt(k.From.ID, 10),
						FromUsername: k.From.UserName,
						Text:         k.Text,
						Timestamp:    k.Time(),
					}
					err := b.storage.Save(ctx, updMessage)
					if err != nil {
						return fmt.Errorf("failed to save update message\n(%v)\n[ERROR]: %v", updMessage, err)
					}
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
					return fmt.Errorf("failed to unmarshal answer on: %v\n[ERROR]: %v", update.Message.Text, err)
				}
				if response.Choices == nil {
					err = json.Unmarshal(data, &errMsg)
					if err != nil {
						return fmt.Errorf("there is an error in AI service (CHECK TOKENS IF NOT SURE)\nfailed to unmarshal answer on: %v\n[ERROR]: %v", update.Message.Text, err)
					}
					errMessage := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("[ERROR]: Code: %v, Message: %v", errMsg.Error.Code, errMsg.Error.Message))
					message, err := b.api.Send(errMessage)
					if err != nil {
						return fmt.Errorf("failed to send error message\n(%v)\n[ERROR]: %v", errMessage, err)
					}
					fmt.Println(message)
				}

				for _, k := range response.Choices {
					r1Content := k.Message.Content
					newMessage := tgbotapi.NewMessage(update.Message.Chat.ID, r1Content)
					answerMessage, err := b.api.Send(newMessage)
					if err != nil {
						return fmt.Errorf("failed to send answer (%v), err: %v", aMsg, err)
					}
					aMsg = append(aMsg, answerMessage)
				}

				for _, k := range aMsg {

					var parsedRunes []rune
					if len([]rune(k.Text)) > 200 {
						fmt.Printf("[WARNING]: answer message too large (%d), saving beginnig...\n", len(k.Text))
						parsedRunes = []rune(k.Text)[:200]

					}

					ansMessage := &models.Message{
						MessageID:    k.MessageID,
						FromID:       strconv.FormatInt(k.From.ID, 10),
						FromUsername: k.From.UserName,
						Text:         string(parsedRunes),
						Timestamp:    k.Time(),
					}

					err := b.storage.Save(ctx, ansMessage)
					if err != nil {
						return fmt.Errorf("failed to save answers\n(%v)\n[ERROR]: %v", ansMessage, err)
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
