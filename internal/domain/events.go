package domain

// EventType представляет тип события
type EventType string

const (
	EventTypeCommandReceived     EventType = "CommandReceived"
	EventTypeTextMessageReceived EventType = "TextMessageReceived"
	EventTypeCallbackReceived    EventType = "CallbackReceived"
	EventTypeVoiceReceived       EventType = "VoiceReceived"
	EventTypeVideoReceived       EventType = "VideoReceived"
)

// Event представляет базовый интерфейс для всех событий
type Event interface {
	Type() EventType
	ChatID() int64
	UserID() int64
}

// CommandReceived представляет событие получения команды
type CommandReceived struct {
	chatID    int64
	userID    int64
	Command   string
	Arguments string
	MessageID int
}

func (e CommandReceived) Type() EventType {
	return EventTypeCommandReceived
}

func (e CommandReceived) ChatID() int64 {
	return e.chatID
}

func (e CommandReceived) UserID() int64 {
	return e.userID
}

// NewCommandReceived создает новое событие CommandReceived
func NewCommandReceived(chatID, userID int64, command, arguments string, messageID int) CommandReceived {
	return CommandReceived{
		chatID:    chatID,
		userID:    userID,
		Command:   command,
		Arguments: arguments,
		MessageID: messageID,
	}
}

// TextMessageReceived представляет событие получения текстового сообщения
type TextMessageReceived struct {
	chatID    int64
	userID    int64
	Text      string
	MessageID int
	Username  string
}

func (e TextMessageReceived) Type() EventType {
	return EventTypeTextMessageReceived
}

func (e TextMessageReceived) ChatID() int64 {
	return e.chatID
}

func (e TextMessageReceived) UserID() int64 {
	return e.userID
}

// NewTextMessageReceived создает новое событие TextMessageReceived
func NewTextMessageReceived(chatID, userID int64, text, username string, messageID int) TextMessageReceived {
	return TextMessageReceived{
		chatID:    chatID,
		userID:    userID,
		Text:      text,
		Username:  username,
		MessageID: messageID,
	}
}

// CallbackReceived представляет событие получения callback query
type CallbackReceived struct {
	chatID    int64
	userID    int64
	Data      string
	MessageID int
	QueryID   string
}

func (e CallbackReceived) Type() EventType {
	return EventTypeCallbackReceived
}

func (e CallbackReceived) ChatID() int64 {
	return e.chatID
}

func (e CallbackReceived) UserID() int64 {
	return e.userID
}

// NewCallbackReceived создает новое событие CallbackReceived
func NewCallbackReceived(chatID, userID int64, data, queryID string, messageID int) CallbackReceived {
	return CallbackReceived{
		chatID:    chatID,
		userID:    userID,
		Data:      data,
		QueryID:   queryID,
		MessageID: messageID,
	}
}

// VoiceReceived представляет событие получения голосового сообщения
type VoiceReceived struct {
	chatID    int64
	userID    int64
	Username  string
	FileID    string
	Duration  int
	MessageID int
}

func (e VoiceReceived) Type() EventType {
	return EventTypeVoiceReceived
}

func (e VoiceReceived) ChatID() int64 {
	return e.chatID
}

func (e VoiceReceived) UserID() int64 {
	return e.userID
}

// NewVoiceReceived создает новое событие VoiceReceived
func NewVoiceReceived(chatID, userID int64, username, fileID string, duration, messageID int) VoiceReceived {
	return VoiceReceived{
		chatID:    chatID,
		userID:    userID,
		Username:  username,
		FileID:    fileID,
		Duration:  duration,
		MessageID: messageID,
	}
}

// VideoReceived представляет событие получения видео
type VideoReceived struct {
	chatID    int64
	userID    int64
	FileID    string
	Duration  int
	MessageID int
}

func (e VideoReceived) Type() EventType {
	return EventTypeVideoReceived
}

func (e VideoReceived) ChatID() int64 {
	return e.chatID
}

func (e VideoReceived) UserID() int64 {
	return e.userID
}

// NewVideoReceived создает новое событие VideoReceived
func NewVideoReceived(chatID, userID int64, fileID string, duration, messageID int) VideoReceived {
	return VideoReceived{
		chatID:    chatID,
		userID:    userID,
		FileID:    fileID,
		Duration:  duration,
		MessageID: messageID,
	}
}
