package services

import (
	"context"
	"fmt"

	"github.com/vlks-dev/mytelegrambotapi/deepseek"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"github.com/vlks-dev/mytelegrambotapi/utils"
	"go.uber.org/zap"
)

// AIService интерфейс для работы с AI
type AIService interface {
	// GenerateAnswer генерирует ответ на основе диалога
	GenerateAnswer(ctx context.Context, dialog []domain.Message) (string, error)
}

// aiService реализация AIService на основе deepseek.R1
type aiService struct {
	r1     deepseek.R1
	logger *zap.SugaredLogger
}

// NewAIService создает новый AIService
func NewAIService(r1 deepseek.R1, logger *zap.SugaredLogger) AIService {
	return &aiService{
		r1:     r1,
		logger: logger.Named("ai_service"),
	}
}

// GenerateAnswer генерирует ответ на основе диалога
func (s *aiService) GenerateAnswer(ctx context.Context, dialog []domain.Message) (string, error) {
	// Если диалог пустой, возвращаем ошибку
	if len(dialog) == 0 {
		return "", fmt.Errorf("empty dialog")
	}

	// Преобразуем историю диалога в формат ChatMessage с ролями
	chatMessages := make([]deepseek.ChatMessage, 0, len(dialog))
	
	for _, msg := range dialog {
		if msg.Text == "" {
			continue // Пропускаем пустые сообщения
		}
		
		// Определяем роль сообщения: если from_id == 0 или from_username == "bot", то это ответ ассистента
		role := "user"
		if msg.FromID == 0 || msg.FromUsername == "bot" {
			role = "assistant"
		}
		
		chatMessages = append(chatMessages, deepseek.ChatMessage{
			Role:    role,
			Content: msg.Text,
		})
	}

	// Если после фильтрации нет сообщений, возвращаем ошибку
	if len(chatMessages) == 0 {
		return "", fmt.Errorf("empty dialog after filtering")
	}

	s.logger.Debugw("Generating answer with context", 
		"total_messages", len(chatMessages),
		"user_messages", countMessagesByRole(chatMessages, "user"),
		"assistant_messages", countMessagesByRole(chatMessages, "assistant"),
	)

	// Вызываем R1 для генерации ответа с историей
	answerJSON, err := s.r1.AnswerWithHistory(ctx, chatMessages)
	if err != nil {
		s.logger.Errorw("Failed to get AI answer", "error", err)
		return "", fmt.Errorf("failed to get AI answer: %w", err)
	}

	// Парсим ответ
	choices, err := utils.ParseChoices(answerJSON)
	if err != nil {
		s.logger.Errorw("Failed to parse AI response", "error", err)
		return "", fmt.Errorf("failed to parse AI response: %w", err)
	}

	if len(choices) == 0 {
		s.logger.Warn("No response generated")
		return "Не удалось сгенерировать ответ", nil
	}

	// Объединяем все части ответа
	var result string
	for i, choice := range choices {
		if i > 0 {
			result += "\n"
		}
		result += choice
	}

	return result, nil
}

// countMessagesByRole подсчитывает количество сообщений с определенной ролью
func countMessagesByRole(messages []deepseek.ChatMessage, role string) int {
	count := 0
	for _, msg := range messages {
		if msg.Role == role {
			count++
		}
	}
	return count
}

// aiServiceStub заглушка реализации AIService
type aiServiceStub struct{}

// NewAIServiceStub создает заглушку AIService
func NewAIServiceStub() AIService {
	return &aiServiceStub{}
}

// GenerateAnswer заглушка метода генерации ответа
func (s *aiServiceStub) GenerateAnswer(ctx context.Context, dialog []domain.Message) (string, error) {
	return "Заглушка ответа AI", nil
}
