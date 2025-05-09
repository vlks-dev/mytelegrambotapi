package deepseek

import (
	"context"
	"fmt"
	"github.com/mytelegrambot/config"
	"github.com/openai/openai-go" // imported as openai
	"github.com/openai/openai-go/option"
	"go.uber.org/zap"
	"time"
)

type R1 interface {
	AnswerQuestion(ctx context.Context, question string) (string, error)
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
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	c.logger.Debugf("[Chat Completion] Answering question message: (%v)", message)
	completion, err := c.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(message),
			},
			Model: "deepseek/deepseek-chat-v3-0324:free",
			//Model: "deepseek/deepseek-r1:free",
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
