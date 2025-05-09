package buttons

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func InitKeyboard() tgbotapi.ReplyKeyboardMarkup {
	var buttons []tgbotapi.KeyboardButton
	checkSubscription := tgbotapi.NewKeyboardButton("Проверить подписку")
	listSessions := tgbotapi.NewKeyboardButton("Доступные чаты")
	keyboardButtons := append(buttons, checkSubscription, listSessions)
	keyboard := tgbotapi.NewOneTimeReplyKeyboard(keyboardButtons)

	keyboard.InputFieldPlaceholder = "Выберите вариант"
	return keyboard
}
