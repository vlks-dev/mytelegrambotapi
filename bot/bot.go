package bot

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/go-telegram/bot"
	"github.com/mytelegrambot/config"
	"log"
	"time"
)

type BotAPI interface {
	GetUpdates(ctx context.Context) (<-chan tgbotapi.Update, error)
	SendMessage(chatID int64, text string) (*tgbotapi.Message, error)
	DeleteMessages(ctx context.Context, chatID int64, messageIDs []int) error
	HandleCommand(ctx context.Context, msg *tgbotapi.Message, msgIDs []int) (*tgbotapi.Message, error)
	GetMyCommands() ([]tgbotapi.BotCommand, error)
	DeleteMessage(ctx context.Context, chatID int64, msgID int) error
}

func (b *Bot) DeleteMessage(ctx context.Context, chatID int64, msgID int) error {
	_, err := b.tgBot.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: msgID,
	})
	if err != nil {
		return fmt.Errorf("tg bot delete message (%v) from chat (%v): %w", msgID, chatID, err)
	}
	return nil
}

type Bot struct {
	api   *tgbotapi.BotAPI
	tgBot *bot.Bot
}

func NewBot(config *config.Config) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, fmt.Errorf("parsing telegram bot token err: %v", err)
	}

	botAPI.Debug = config.BotEnv

	switch {
	case botAPI.Debug == true:
		log.Printf("authorized on account @%v in debug mode! (%v)\n",
			botAPI.Self.UserName, botAPI.Self.FirstName)
	case botAPI.Debug == false:
		log.Printf("authorized on account @%v! (%v)\n", botAPI.Self.UserName, botAPI.Self.FirstName)
	}

	var opts []bot.Option

	if config.BotEnv == true {
		opts = []bot.Option{
			bot.WithDebug(),
		}
	}

	b, err := bot.New(config.Token, opts...)
	if err != nil {
		return nil, fmt.Errorf("new bot with token and opts err: %w", err)
	}

	return &Bot{
		tgBot: b,
		api:   botAPI,
	}, nil
}

func (b *Bot) GetMyCommands() ([]tgbotapi.BotCommand, error) {
	commands, err := b.api.GetMyCommands()
	if err != nil {
		return nil, fmt.Errorf("get commands for %v: %w", b.api.Self.UserName, err)
	}
	log.Printf("got commands for %v", b.api.Self.UserName)
	return commands, nil
}

// GetUpdates запускает цикл получения апдейтов и делегирует их обработку
func (b *Bot) GetUpdates(ctx context.Context) (<-chan tgbotapi.Update, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 25
	updates := b.api.GetUpdatesChan(u)
	return updates, nil
}

func (b *Bot) HandleCommand(ctx context.Context, msg *tgbotapi.Message, msgIDs []int) (*tgbotapi.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	chatID := msg.Chat.ID

	log.Printf("входящая команда: %v, чат (%v)", msg.Text, chatID)

	switch msg.Command() {
	case "help":
		answer, err := b.SendMessage(chatID, "Я Простой чат-бот на основе Openai API, написанный на Golang, с используемой моделью - DeepSeek V3")
		if err != nil {
			return nil, fmt.Errorf("send answer error: %w", err)
		}
		return answer, nil
	case "start":
		answer, err := b.SendMessage(chatID, fmt.Sprint("Привет! Задавай мне вопросы, а постараюсь ответить на них правильно! (на базе DeepSeek v3)"))
		if err != nil {
			return nil, fmt.Errorf("send start command mock, chat (%v) error: %w", msg.Chat.ID, err)
		}
		return answer, nil
	case "restart":
		err := b.DeleteMessages(ctx, chatID, msgIDs)
		if err != nil {
			return nil, fmt.Errorf("%v command, chat (%v) error: %w", msg.Command(), msg.Chat.ID, err)
		}
	}

	log.Printf("%v command in %v-chat", msg.Command(), chatID)

	deadline, ok := ctx.Deadline()
	if !ok {
		log.Println("deadline not set in ctx")
	}

	answer, err := b.SendMessage(chatID, fmt.Sprintf("Команда %v выполнена, осталось времени: %v ", msg.Text, time.Until(deadline)))
	if err != nil {
		return nil, fmt.Errorf("send answer (%v) err: %w", answer.Text, err)
	}

	return answer, nil
}

func (b *Bot) DeleteMessages(ctx context.Context, chatID int64, messageIDs []int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := b.tgBot.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chatID,
		MessageIDs: messageIDs,
	})
	if err != nil {
		return fmt.Errorf("delete message err: %w", err)
	}

	return nil
}

// SendMessage отправляет текст в чат и возвращает отправленное сообщение
func (b *Bot) SendMessage(chatID int64, text string) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	message, err := b.api.Send(msg)
	if err != nil {
		return nil, fmt.Errorf("send message (%v), err: %w", msg, err)
	}

	return &message, nil

}
