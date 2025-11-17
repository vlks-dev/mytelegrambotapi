package domain

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// BotResponse представляет ответ бота пользователю
type BotResponse struct {
	Text     string
	Buttons  *tgbotapi.InlineKeyboardMarkup
	Keyboard *tgbotapi.ReplyKeyboardMarkup
	File     *FileReference
	Edit     *EditMessage
	Callback *CallbackAnswer
}

// FileReference представляет ссылку на файл для отправки
type FileReference struct {
	Type    FileType
	FileID  string
	Caption string
}

// FileType представляет тип файла
type FileType string

const (
	FileTypePhoto FileType = "photo"
	FileTypeVoice FileType = "voice"
	FileTypeVideo FileType = "video"
)

// EditMessage представляет информацию для редактирования сообщения
type EditMessage struct {
	MessageID int
	Text      string
	Buttons   *tgbotapi.InlineKeyboardMarkup
}

// CallbackAnswer представляет ответ на callback query
type CallbackAnswer struct {
	QueryID   string
	Text      string
	ShowAlert bool
}

// NewTextResponse создает текстовый ответ
func NewTextResponse(text string) *BotResponse {
	return &BotResponse{
		Text: text,
	}
}

// NewTextResponseWithButtons создает текстовый ответ с inline кнопками
func NewTextResponseWithButtons(text string, buttons *tgbotapi.InlineKeyboardMarkup) *BotResponse {
	return &BotResponse{
		Text:    text,
		Buttons: buttons,
	}
}

// NewTextResponseWithKeyboard создает текстовый ответ с клавиатурой
func NewTextResponseWithKeyboard(text string, keyboard *tgbotapi.ReplyKeyboardMarkup) *BotResponse {
	return &BotResponse{
		Text:     text,
		Keyboard: keyboard,
	}
}

// NewFileResponse создает ответ с файлом
func NewFileResponse(fileType FileType, fileID string, caption string) *BotResponse {
	return &BotResponse{
		File: &FileReference{
			Type:    fileType,
			FileID:  fileID,
			Caption: caption,
		},
	}
}

// NewEditResponse создает ответ для редактирования сообщения
func NewEditResponse(messageID int, text string, buttons *tgbotapi.InlineKeyboardMarkup) *BotResponse {
	return &BotResponse{
		Edit: &EditMessage{
			MessageID: messageID,
			Text:      text,
			Buttons:   buttons,
		},
	}
}

// NewCallbackAnswer создает ответ на callback query
func NewCallbackAnswer(queryID string, text string, showAlert bool) *BotResponse {
	return &BotResponse{
		Callback: &CallbackAnswer{
			QueryID:   queryID,
			Text:      text,
			ShowAlert: showAlert,
		},
	}
}
