package deepseek

import (
	"context"
	"fmt"
	"github.com/mytelegrambot/config"
	"github.com/openai/openai-go" // imported as openai
	"github.com/openai/openai-go/option"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		option.WithMaxRetries(3))
	if err != nil {
		return "", fmt.Errorf("[ERROR]: failed to get new deep-seek completion:\n%v", err)
	}

	return completion.RawJSON(), nil

}
