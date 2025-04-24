package tg_bot

import (
	"context"
	"fmt"
	"github.com/go-telegram/bot"
	"github.com/mytelegrambot/config"
	"log"
)

type Bot struct {
	*bot.Bot
}

func NewBot(cfg *config.Config) (*Bot, error) {
	var opts []bot.Option

	if cfg.BotEnv == true {
		opts = []bot.Option{
			bot.WithDebug(),
		}
	}

	b, err := bot.New(cfg.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("new bot with token and opts err: %w", err)
	}

	return &Bot{b}, nil
}

func (b *Bot) DeleteMockMessage(ctx context.Context, chatID any, messageID int) error {
	_, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: chatID, MessageID: messageID})
	if err != nil {
		log.Printf("chat (%v), message (%v) delete err: %v", chatID, messageID, err)
		return fmt.Errorf("message (%v) delete err: %w", messageID, err)
	}
	return nil
}
