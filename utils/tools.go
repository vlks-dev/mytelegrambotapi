package utils

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/mytelegrambot/models"
	"time"
)

func Truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes])
}

func BotMessageToModel(message *tgbotapi.Message) *models.Message {
	const maxRunes = 200
	return &models.Message{
		ChatID:       message.Chat.ID,
		MessageID:    message.MessageID,
		FromID:       message.From.ID,
		FromUsername: message.From.UserName,
		Text:         Truncate(message.Text, maxRunes),
		Timestamp:    time.Now(),
	}
}

// ParseChoices разбирает JSON-ответ AI и возвращает список текстов
func ParseChoices(data string) ([]string, error) {
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
