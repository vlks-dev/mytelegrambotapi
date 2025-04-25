package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/go-telegram/bot"
	"github.com/mytelegrambot/config"
	"github.com/mytelegrambot/deepseek"
	"github.com/mytelegrambot/models"
	"github.com/mytelegrambot/storage"
	"log"
	"strconv"
)

type BotAPI interface {
	GetUpdates(ctx context.Context) error
	SendAnswer(chatID int64, text string) (*tgbotapi.Message, error)
}

type Bot struct {
	api   *tgbotapi.BotAPI
	tgBot *bot.Bot
}

func NewBot(config *config.Config, tgBot *bot.Bot, storage storage.Storage, r1 deepseek.R1) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, fmt.Errorf("parsing telegram bot token err: %v", err)
	}

	botAPI.Debug = config.BotEnv

	switch {
	case botAPI.Debug == true:
		log.Printf("authorized on account @%v in debug mode! (%v)\n", botAPI.Self.UserName, botAPI.Self.FirstName)
	case botAPI.Debug == false:
		log.Printf("authorized on account @%v! (%v)\n", botAPI.Self.UserName, botAPI.Self.FirstName)
	}

	return &Bot{
		tgBot: tgBot,
		api:   botAPI,
	}, nil
}

// GetUpdates запускает цикл получения апдейтов и делегирует их обработку
func (b *Bot) GetUpdates(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 25
	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case upd, ok := <-updates:
			if !ok {
				log.Println("update channel closed")
				return nil // канал закрылся
			}
			if upd.Message == nil {
				continue
			}
			if upd.Message.IsCommand() {
				commands, getCmdErr := b.api.GetMyCommands()
				if getCmdErr != nil {
					return fmt.Errorf("get commands: %w", getCmdErr)
				}
				if len(commands) == 0 {
					log.Printf("no commands found for: @%v", b.api.Self.UserName)
					_, err := b.SendAnswer(upd.Message.Chat.ID, fmt.Sprintf("нет доступных команд: @%v", b.api.Self.UserName))
					if err != nil {
						return fmt.Errorf("send answer err: %w", err)
					}
					break
				}

				log.Printf("can use next commands: %v", commands)
			}
			if err := b.processIncoming(ctx, upd.Message); err != nil {
				return fmt.Errorf("processing update: %w", err)
			}

		case <-ctx.Done():
			log.Printf("bot stopped: %v", ctx.Err())
			return ctx.Err()
		}
	}
}

// processIncoming обрабатывает одно входящее сообщение
func (b *Bot) processIncoming(ctx context.Context, msg *tgbotapi.Message) error {
	const maxRetries = 2
	var (
		raw string
		err error
	)

	text := truncate(msg.Text, 200)
	log.Printf("[%s] %s", msg.From.UserName, text+"...")

	// Сохранение входящего
	in := &models.Message{
		MessageID:    msg.MessageID,
		FromID:       strconv.FormatInt(msg.From.ID, 10),
		FromUsername: msg.From.UserName,
		Text:         text,
		Timestamp:    msg.Time(),
	}
	if err := b.saveMessage(ctx, in); err != nil {
		return fmt.Errorf("save incoming: %w", err)
	}

	mockID, sendMockErr := b.sendMock(msg.Chat.ID)
	if sendMockErr != nil {
		log.Printf("sendMockErr: %v", sendMockErr)
	}

	// Получаем ответ от AI
	for attempt := 0; attempt <= maxRetries; attempt++ {
		raw, err = b.r1.AnswerQuestion(ctx, msg.Text)
		if err == nil {
			// получили ответ
			break
		}

		// если вышел дедлайн, ретраим
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Attempt %d: timeout, retrying…", attempt+1)
			if attempt < maxRetries {
				continue
			}
			// все попытки исчерпаны
			if _, sendErr := b.SendAnswer(msg.Chat.ID,
				"Время ожидания вышло, попробуем ещё раз?"); sendErr != nil {
				return fmt.Errorf("send timeout notice failed: %w", sendErr)
			}
			return fmt.Errorf("timeout after %d retries: %w", maxRetries+1, err)
		}

		// любая другая ошибка, вылетаем
		return fmt.Errorf("AI error: %w", err)
	}

	err = b.deleteMock(ctx, msg.Chat.ID, mockID)
	if err != nil {
		return fmt.Errorf("delete mock: %w", err)
	}

	// Парсим и отправляем ответы
	choices, err := parseChoices(raw)
	if err != nil {
		return fmt.Errorf("parse AI response: %w", err)
	}

	for _, content := range choices {
		sent, err := b.SendAnswer(msg.Chat.ID, content)
		if err != nil {
			return fmt.Errorf("send answer: %w", err)
		}
		// Обрезаем длинный текст и сохраняем
		text = truncate(content, 200)
		out := &models.Message{
			MessageID:    sent.MessageID,
			FromID:       strconv.FormatInt(sent.From.ID, 10),
			FromUsername: sent.From.UserName,
			Text:         text,
			Timestamp:    sent.Time(),
		}
		if err := b.saveMessage(ctx, out); err != nil {
			return fmt.Errorf("save outgoing: %w", err)
		}
		log.Printf("[%s] %s", b.api.Self.UserName, text+"...")
	}
	return nil
}

/*// saveMessage сохраняет сообщение в хранилище
func (b *Bot) saveMessage(ctx context.Context, m *models.Message) error {
	return b.storage.Save(ctx, m)
}*/

// SendAnswer отправляет текст в чат и возвращает отправленное сообщение
func (b *Bot) SendAnswer(chatID int64, text string) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	message, err := b.api.Send(msg)
	if err != nil {
		return nil, fmt.Errorf("send message (%v), err: %w", msg, err)
	}
	return &message, nil // Telegram API сам учитывает ctx внутри

}

// parseChoices разбирает JSON-ответ AI и возвращает список текстов
func parseChoices(data string) ([]string, error) {
	var response models.CompletionResponse
	var errMsg models.ErrorR1Message

	err := json.Unmarshal([]byte(data), &response)
	if err == nil && len(response.Choices) > 0 {
		var text []string
		for _, choice := range response.Choices {
			if choice.Message.Content != "" {
				text = append(text, choice.Message.Content)
			}
		}
		text = append(text, fmt.Sprintf("потрачено %d токенов", response.Usage.TotalTokens))
		return text, nil
	}

	// Попытка распарсить ошибку
	if err = json.Unmarshal([]byte(data), &errMsg); err != nil {
		return nil, fmt.Errorf("не удалось распарсить ответ:\n%v\n[ОШИБКА]: %w", data, err)
	}
	return []string{"Закончились токены! Попробуйте завтра"}, nil
}

func (b *Bot) sendMock(chatID int64) (int, error) {
	mock, err := b.SendAnswer(chatID, fmt.Sprintf("Ваш ответ генерируется, подождите!\n@%v", b.api.Self.UserName))
	if err != nil {
		return 0, fmt.Errorf("send mock response err: %w", err)
	}
	return mock.MessageID, nil
}

func (b *Bot) deleteMock(ctx context.Context, chatID int64, messageID int) error {
	_, err := b.tgBot.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	})
	if err != nil {
		return fmt.Errorf("delete mock message err: %w", err)
	}
	return nil
}

// truncate обрезает строку до maxRunes
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}
