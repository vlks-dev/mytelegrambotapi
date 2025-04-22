package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mytelegrambot/config"
	"github.com/mytelegrambot/models"
	"log"
	"strconv"
)

type Bot struct {
	api     *tgbotapi.BotAPI
	storage Storage
	r1      R1Client
}

func NewBot(config *config.Config, storage Storage, r1 R1Client) (*Bot, error) {
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
		storage: storage,
		r1:      r1,
		api:     botAPI,
	}, nil
}

type Storage interface {
	Save(ctx context.Context, msg *models.Message) error
}

type R1Client interface {
	AnswerQuestion(ctx context.Context, question string) (string, error)
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
				return nil // канал закрылся
			}
			if upd.Message == nil {
				continue
			}
			if err := b.processIncoming(ctx, upd.Message); err != nil {
				return err
			}

		case <-ctx.Done():
			log.Printf("Bot stopped: %v", ctx.Err())
			return ctx.Err()
		}
	}
}

// processIncoming обрабатывает одно входящее сообщение
func (b *Bot) processIncoming(ctx context.Context, msg *tgbotapi.Message) error {
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

	// Получаем ответ от AI
	raw, err := b.r1.AnswerQuestion(ctx, msg.Text)
	/*if errors.Is(err, ctx.Err()) == false {
		//retry logic
		log.Printf("Q/A context err: %v", err)
		_, ctxErr := b.sendAnswer(msg.Chat.ID, "кажется, время ожидания ответа вышло, повторить запрос?")
		if err != nil {
			return fmt.Errorf("send context timeout warning: %w", ctxErr)
		}
	}*/
	if err != nil && !errors.Is(err, ctx.Err()) {
		return fmt.Errorf("AI error: %w", err)
	}

	// Парсим и отправляем ответы
	choices, err := parseChoices(raw)
	if err != nil {
		return fmt.Errorf("parse AI response: %w", err)
	}

	for _, content := range choices {
		sent, err := b.sendAnswer(msg.Chat.ID, content)
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

// saveMessage сохраняет сообщение в хранилище
func (b *Bot) saveMessage(ctx context.Context, m *models.Message) error {
	return b.storage.Save(ctx, m)
}

// sendAnswer отправляет текст в чат и возвращает отправленное сообщение
func (b *Bot) sendAnswer(chatID int64, text string) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	message, err := b.api.Send(msg)
	if err != nil {
		return nil, fmt.Errorf("send message (%v), err: %w", msg, err)
	}
	return &message, nil // Telegram API сам учитывает ctx внутри

}

// parseChoices разбирает JSON-ответ AI и возвращает список текстов
func parseChoices(data string) ([]string, error) {
	bytes := []byte(data)
	var text []string
	var response models.CompletionResponse
	var errMsg models.ErrorR1Message

	err := json.Unmarshal(bytes, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response on, raw json:\n%v\n[ERROR]: %w", data, err)
	}
	if response.Choices == nil || len(response.Choices) == 0 {
		err = json.Unmarshal(bytes, &errMsg)
		if err != nil {
			return nil, fmt.Errorf("there is an error in AI service (CHECK TOKENS IF NOT SURE)\nfailed to unmarshal answer on: %v\n[ERROR]: %w", data, err)
		}

		text = append(text, strconv.Itoa(errMsg.Error.Code), errMsg.Error.Message)
		return text, nil // упрощённо
	}

	for _, k := range response.Choices {
		text = append(text, k.Message.Content, fmt.Sprintf("потрачено %d токенов", response.Usage.TotalTokens))
	}

	return text, nil
}

// truncate обрезает строку до maxRunes
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}
