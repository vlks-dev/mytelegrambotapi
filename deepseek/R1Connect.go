package deepseek

import (
	"context"
	"fmt"
	"github.com/mytelegrambot/config"
	"github.com/openai/openai-go" // imported as openai
	"github.com/openai/openai-go/option"
	"log"
	"time"
)

type R1Client struct {
	client openai.Client
}

func NewR1(config *config.Config) *R1Client {
	var client openai.Client

	client = openai.NewClient(
		option.WithBaseURL(
			"https://openrouter.ai/api/v1",
		),
		option.WithAPIKey(
			config.R1ProToken,
		),
	)

	return &R1Client{client: client}
}

func (c *R1Client) AnswerQuestion(ctx context.Context, message string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()

	completion, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(message),
			},
			Model: "deepseek/deepseek-chat-v3-0324:free",
			//Model: "deepseek/deepseek-r1:free",
		},
		option.WithMaxRetries(3),
		option.WithRequestTimeout(100*time.Second))

	if err != nil {
		if ctx.Err() != nil {
			return "таймаут/отмена", ctx.Err()
		}
		return "ошибка получения ответа", fmt.Errorf("failed to get new deep-seek completion:\n%w", err)
	}

	deadline, _ := ctx.Deadline()

	log.Printf("deepseek completion: %s, time left: %v", completion.ID, time.Until(deadline).Round(time.Second))
	return completion.RawJSON(), nil
}
