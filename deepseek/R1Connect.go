package deepseek

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/vlks-dev/mytelegrambotapi/config"
	"go.uber.org/zap"
)

type R1 interface {
	AnswerQuestion(ctx context.Context, question string) (string, error)
	AnswerWithHistory(ctx context.Context, messages []ChatMessage) (string, error)
}

// ChatMessage представляет сообщение в диалоге с ролью
type ChatMessage struct {
	Role    string // "user" или "assistant"
	Content string
}

type R1Client struct {
	logger *zap.SugaredLogger
	client openai.Client
}

func NewR1(config *config.Config, logger *zap.SugaredLogger) *R1Client {
	var client openai.Client

	client = openai.NewClient(
		option.WithBaseURL(
			config.AIApiUrl,
		),
		option.WithAPIKey(
			config.R1ProToken,
		),
	)

	log := logger.Named("AI")
	log.Debugf("Create new R1-AI client, from API: (%v)", config.AIApiUrl)
	return &R1Client{
		logger: log,
		client: client,
	}
}

func (c *R1Client) AnswerQuestion(ctx context.Context, message string) (string, error) {
	// Для обратной совместимости используем новый метод с одним сообщением
	return c.AnswerWithHistory(ctx, []ChatMessage{
		{Role: "user", Content: message},
	})
}

// AnswerWithHistory генерирует ответ на основе истории диалога
func (c *R1Client) AnswerWithHistory(ctx context.Context, messages []ChatMessage) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	c.logger.Debugf("[Chat Completion] Answering with history (%d messages): (%v)",
		len(messages), time.Now().Local().Round(time.Second))

	// Преобразуем ChatMessage в формат OpenAI API
	apiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)

	// Добавляем системное сообщение с инструкциями
	apiMessages = append(apiMessages, openai.SystemMessage(
		"Prepare answer in 2000 characters, or less. Respond STRICTLY in PLAIN text format without *bold*, _italics_, lists, or **headers** and ```code``` blocks, maybe you can use emojis if it's appropriate.",
	))

	// Добавляем историю диалога
	for _, msg := range messages {
		if msg.Content == "" {
			continue // Пропускаем пустые сообщения
		}

		if msg.Role == "user" {
			apiMessages = append(apiMessages, openai.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			apiMessages = append(apiMessages, openai.AssistantMessage(msg.Content))
		}
	}

	// Если нет сообщений, возвращаем ошибку
	if len(apiMessages) == 1 { // Только системное сообщение
		return "", fmt.Errorf("empty dialog history")
	}

	completion, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: apiMessages,
			Model:    "tngtech/deepseek-r1t2-chimera:free",
			// Model:    "deepseek/deepseek-chat-v3-0324:free",
			// Model: "deepseek/deepseek-chat-v3.1:free",
			// Model: "deepseek/deepseek-r1-0528-qwen3-8b:free",
			// Model: "deepseek/deepseek-r1:free",
		})

	if err != nil {
		if ctx.Err() != nil {
			return "таймаут/отмена", ctx.Err()
		}
		return "ошибка получения ответа", fmt.Errorf("failed to get new deep-seek completion:\n%w", err)
	}

	deadline, _ := ctx.Deadline()
	c.logger.Debugf("deepseek completion: %s, time left: %v", completion.ID, time.Until(deadline).Round(time.Second))
	return completion.RawJSON(), nil
}
