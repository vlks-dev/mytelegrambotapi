package models

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"time"
)

type Message struct {
	ChatID       int64
	MessageID    int       `json:"message_id"`
	FromID       int64     `json:"from_id"`
	FromUsername string    `json:"from_username"`
	Text         string    `json:"text"`
	Timestamp    time.Time `json:"time_stamp"`
}

type Updates struct {
	tgbotapi.UpdatesChannel
}
