package services

import (
	"context"
)

// SpeechToTextService интерфейс для преобразования речи в текст
type SpeechToTextService interface {
	// Transcribe преобразует голосовое сообщение в текст
	Transcribe(ctx context.Context, fileID string) (string, error)
}

// speechToTextServiceStub заглушка реализации SpeechToTextService
type speechToTextServiceStub struct{}

// NewSpeechToTextServiceStub создает заглушку SpeechToTextService
func NewSpeechToTextServiceStub() SpeechToTextService {
	return &speechToTextServiceStub{}
}

// Transcribe заглушка метода преобразования речи в текст
func (s *speechToTextServiceStub) Transcribe(ctx context.Context, fileID string) (string, error) {
	return "заглушка для голосового сообщения", nil
}

