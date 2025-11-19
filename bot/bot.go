package bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/go-telegram/bot"
	"github.com/vlks-dev/mytelegrambotapi/bot/buttons"
	"github.com/vlks-dev/mytelegrambotapi/config"
	"go.uber.org/zap"
)

type AIBotAPI interface {
	GetUpdates(ctx context.Context) (<-chan tgbotapi.Update, error)
	SendMessage(chatID int64, text string) (*tgbotapi.Message, error)
	DeleteMessages(ctx context.Context, chatID int64, messageIDs []int) error
	HandleCommand(ctx context.Context, msg *tgbotapi.Message, msgIDs []int) (*tgbotapi.Message, error)
	GetMyCommands() ([]tgbotapi.BotCommand, error)
	DeleteMessage(ctx context.Context, chatID int64, msgID int) error
}

// ExtendedBotAPI расширенный интерфейс для дополнительных методов Telegram API
type ExtendedBotAPI interface {
	AIBotAPI
	SendPhoto(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error)
	SendVoice(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error)
	SendVideo(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error)
	EditMessageText(ctx context.Context, chatID int64, messageID int, text string, replyMarkup *tgbotapi.InlineKeyboardMarkup) (*tgbotapi.Message, error)
	AnswerCallbackQuery(ctx context.Context, queryID string, text string, showAlert bool) error
	GetFile(ctx context.Context, fileID string) (*tgbotapi.File, error)
	DownloadFile(ctx context.Context, filePath string, destPath string) error
}

type Bot struct {
	logger *zap.SugaredLogger
	api    *tgbotapi.BotAPI
	tgBot  *bot.Bot
}

func NewBot(config *config.Config, logger *zap.SugaredLogger) (*Bot, error) {
	log := logger.Named("bot")

	botAPI, err := tgbotapi.NewBotAPI(config.Token)
	if err != nil {
		return nil, fmt.Errorf("parsing telegram bot token err: %v", err)
	}

	botAPI.Debug = config.BotEnv

	switch {
	case botAPI.Debug == false:
		log.Debugf("authorized on account @%v in debug mode! (%v)\n",
			botAPI.Self.UserName, botAPI.Self.FirstName)
	case botAPI.Debug == true:
		log.Infof("authorized on account @%v! (%v)\n", botAPI.Self.UserName, botAPI.Self.FirstName)
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
		logger: log,
		tgBot:  b,
		api:    botAPI,
	}, nil
}

func (b *Bot) GetMyCommands() ([]tgbotapi.BotCommand, error) {
	commands, err := b.api.GetMyCommands()
	if err != nil {
		return nil, fmt.Errorf("get commands for %v: %w", b.api.Self.UserName, err)
	}

	keyboard := buttons.InitKeyboard()

	b.logger.Debugf("got commands for %v, startup keyboard buttons: %v", b.api.Self.UserName, keyboard.Keyboard)
	return commands, nil
}

// GetUpdates цикл обновлений с обработкой
func (b *Bot) GetUpdates(ctx context.Context) (<-chan tgbotapi.Update, error) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 25
	updates := b.api.GetUpdatesChan(u)

	return updates, nil
}

func (b *Bot) HandleCommand(ctx context.Context, msg *tgbotapi.Message, msgIDs []int) (*tgbotapi.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	chatID := msg.Chat.ID

	b.logger.Debugf("входящая команда: %v, чат (%v)", msg.Text, chatID)

	switch msg.Command() {
	case "help":
		commands, _ := b.GetMyCommands()
		answer, err := b.SendMessage(chatID, fmt.Sprintf("Я Простой чат-бот на основе Openai API, написанный на Golang, с используемой моделью - DeepSeek V3. Команды для бота:  %v", commands))
		if err != nil {
			return nil, fmt.Errorf("send answer error: %w", err)
		}
		return answer, nil
	case "start":
		message := tgbotapi.NewMessage(chatID, "Привет! Задавай мне вопросы, а постараюсь ответить на них правильно! (на базе DeepSeek v3)")
		message.ReplyMarkup = buttons.InitKeyboard()

		send, err := b.api.Send(message)

		if err != nil {
			return nil, fmt.Errorf("send start command mock, chat (%v) error: %w", msg.Chat.ID, err)
		}
		return &send, nil
	case "restart":
		err := b.DeleteMessages(ctx, chatID, msgIDs)
		if err != nil {
			return nil, fmt.Errorf("%v command, chat (%v) error: %w", msg.Command(), msg.Chat.ID, err)
		}
	}

	return nil, nil
}

func (b *Bot) DeleteMessages(ctx context.Context, chatID int64, messageIDs []int) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if messageIDs == nil {
		b.logger.Warnf("no message IDs provided from %v chat", chatID)
		return nil
	}
	_, err := b.tgBot.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     chatID,
		MessageIDs: messageIDs,
	})
	if err != nil {
		return fmt.Errorf("delete message err: %w", err)
	}

	return nil
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

// SendMessage отправляет текст в чат и возвращает отправленное сообщение
func (b *Bot) SendMessage(chatID int64, text string) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	message, err := b.api.Send(msg)
	if err != nil {
		return nil, fmt.Errorf("send message (%v), err: %w", msg, err)
	}

	return &message, nil
}

// SendPhoto отправляет фото
func (b *Bot) SendPhoto(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error) {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(fileID))
	if caption != "" {
		photo.Caption = caption
	}
	message, err := b.api.Send(photo)
	if err != nil {
		return nil, fmt.Errorf("send photo: %w", err)
	}
	return &message, nil
}

// SendVoice отправляет голосовое сообщение
func (b *Bot) SendVoice(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error) {
	voice := tgbotapi.NewVoice(chatID, tgbotapi.FileID(fileID))
	if caption != "" {
		voice.Caption = caption
	}
	message, err := b.api.Send(voice)
	if err != nil {
		return nil, fmt.Errorf("send voice: %w", err)
	}
	return &message, nil
}

// SendVideo отправляет видео
func (b *Bot) SendVideo(ctx context.Context, chatID int64, fileID string, caption string) (*tgbotapi.Message, error) {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FileID(fileID))
	if caption != "" {
		video.Caption = caption
	}
	message, err := b.api.Send(video)
	if err != nil {
		return nil, fmt.Errorf("send video: %w", err)
	}
	return &message, nil
}

// EditMessageText редактирует текстовое сообщение
func (b *Bot) EditMessageText(ctx context.Context, chatID int64, messageID int, text string, replyMarkup *tgbotapi.InlineKeyboardMarkup) (*tgbotapi.Message, error) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if replyMarkup != nil {
		edit.ReplyMarkup = replyMarkup
	}
	message, err := b.api.Send(edit)
	if err != nil {
		return nil, fmt.Errorf("edit message: %w", err)
	}
	return &message, nil
}

// AnswerCallbackQuery отвечает на callback query
func (b *Bot) AnswerCallbackQuery(ctx context.Context, queryID string, text string, showAlert bool) error {
	callback := tgbotapi.NewCallback(queryID, text)
	callback.ShowAlert = showAlert
	_, err := b.api.Request(callback)
	if err != nil {
		return fmt.Errorf("answer callback query: %w", err)
	}
	return nil
}

// GetFile получает информацию о файле по fileID
func (b *Bot) GetFile(ctx context.Context, fileID string) (*tgbotapi.File, error) {
	file, err := b.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, fmt.Errorf("get file: %w", err)
	}
	return &file, nil
}

// DownloadFile скачивает файл из Telegram по filePath
func (b *Bot) DownloadFile(ctx context.Context, filePath string, destPath string) error {
	url, err := b.api.GetFileDirectURL(filePath)
	if err != nil {
		return fmt.Errorf("get file URL: %w", err)
	}

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Выполняем запрос
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download file: status code %d", resp.StatusCode)
	}

	// Создаем файл для записи
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer outFile.Close()

	// Копируем данные
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	b.logger.Debugw("File downloaded",
		"file_path", filePath,
		"dest_path", destPath,
	)
	return nil
}
