package middleware

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"time"
)

type AiResponder func(ctx context.Context, msg *tgbotapi.Message) error

func LoggingMiddleware(logger *zap.SugaredLogger) func(AiResponder) AiResponder {
	return func(next AiResponder) AiResponder {
		return func(ctx context.Context, msg *tgbotapi.Message) error {
			start := time.Now()
			logger.Infof("[AI] incoming message id (%d) from chat (%d)", msg.MessageID, msg.Chat.ID)

			err := next(ctx, msg)

			if err != nil {
				logger.Infof("[AI] error processing message (%d): %v", msg.MessageID, err)
			} else {
				logger.Infof("[AI] message (%d) processed in %v", msg.MessageID, time.Since(start))
			}

			return err
		}
	}
}

func RetryMiddleware(logger *zap.SugaredLogger, maxRetries int, delay time.Duration) func(AiResponder) AiResponder {
	return func(next AiResponder) AiResponder {
		return func(ctx context.Context, msg *tgbotapi.Message) error {
			var err error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				err = next(ctx, msg)
				if err == nil {
					return nil
				}
				// Если дедлайн истёк — можно попробовать повторить
				if ctx.Err() != nil {
					logger.Infof("[Retry] context error: %v", ctx.Err())
					break
				}
				logger.Infof("[Retry] attempt %d failed: %v", attempt+1, err)
				if attempt < maxRetries {
					time.Sleep(delay)
				}
			}
			return err
		}
	}
}
