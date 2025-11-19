package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vlks-dev/mytelegrambotapi/bot"
	"github.com/vlks-dev/mytelegrambotapi/config"
	"github.com/vlks-dev/mytelegrambotapi/database"
	"github.com/vlks-dev/mytelegrambotapi/deepseek"
	"github.com/vlks-dev/mytelegrambotapi/internal/concurrency"
	"github.com/vlks-dev/mytelegrambotapi/internal/inbound"
	"github.com/vlks-dev/mytelegrambotapi/internal/outbound"
	"github.com/vlks-dev/mytelegrambotapi/internal/routing"
	"github.com/vlks-dev/mytelegrambotapi/internal/services"
	"github.com/vlks-dev/mytelegrambotapi/internal/usecases"
	"github.com/vlks-dev/mytelegrambotapi/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		fmt.Println("Shutting down...")
		cancel()
	}()

	botCfg, err := config.LoadEnvCfg(".env")
	if err != nil {
		log.Fatal(err)
	}

	sugaredLogger := logger.NewLogger(botCfg)
	sugaredLogger.Infoln("tg bot startup by vlks", "configurated by .env")

	// Инициализация бота
	b, err := bot.NewBot(botCfg, sugaredLogger)
	if err != nil {
		sugaredLogger.Fatal(err)
	}

	// Инициализация базы данных
	pool, err := database.GetPool(ctx, botCfg, sugaredLogger)
	if err != nil {
		sugaredLogger.Fatal(err)
	}
	defer pool.Close()

	// Инициализация Redis
	redisClient, err := database.GetRedisClient(ctx, botCfg, sugaredLogger)
	if err != nil {
		sugaredLogger.Warnw("Failed to connect to Redis, continuing without Redis",
			"error", err,
		)
		redisClient = nil
	} else if redisClient != nil {
		defer redisClient.Close()
	}

	// Инициализация DeepSeek R1
	r1 := deepseek.NewR1(botCfg, sugaredLogger)

	// Инициализация сервисов
	aiService := services.NewAIService(r1, sugaredLogger)
	speechToTextService := services.NewWhisperHTTPClient(b, botCfg, sugaredLogger)
	dialogHistoryService := services.NewDialogHistoryService(pool, redisClient, sugaredLogger)
	userRepository := services.NewUserRepository(pool, sugaredLogger)

	// Инициализация use cases
	startUser := usecases.NewStartUser(userRepository, sugaredLogger)
	helpUser := usecases.NewHelpUser(sugaredLogger)
	chatWithAI := usecases.NewChatWithAI(aiService, dialogHistoryService, sugaredLogger)
	processVoice := usecases.NewProcessVoiceMessage(speechToTextService, aiService, dialogHistoryService, sugaredLogger)
	handleCallback := usecases.NewHandleCallback(sugaredLogger)
	adminUseCases := usecases.NewAdminUseCases(userRepository, sugaredLogger)
	fallbackHandler := usecases.NewFallbackHandler()

	// Инициализация router
	router := routing.NewEventRouter(sugaredLogger)
	router.RegisterCommandHandler("start", startUser)
	router.RegisterCommandHandler("help", helpUser)
	router.RegisterCommandHandler("admin", adminUseCases)
	router.RegisterMessageHandler(chatWithAI)
	router.RegisterVoiceHandler(processVoice)
	router.RegisterCallbackHandler("default", handleCallback)
	router.RegisterFallbackHandler(fallbackHandler)

	// Инициализация presenter
	presenter := outbound.NewTelegramPresenter(b, sugaredLogger)

	// Инициализация event processor
	eventProcessor := concurrency.NewEventProcessor(router, presenter, sugaredLogger)

	// Инициализация worker pool (10 воркеров)
	workerPool := concurrency.NewWorkerPool(10, eventProcessor, sugaredLogger)
	workerPool.Start(ctx)

	// Инициализация gateway
	gateway := inbound.NewTelegramGateway(sugaredLogger)

	// Получение updates от Telegram
	updates, err := b.GetUpdates(ctx)
	if err != nil {
		sugaredLogger.Fatal(err)
	}

	sugaredLogger.Infow("waiting for incoming bot requests...", "debug", botCfg.BotEnv)

	errCh := make(chan error)

	// Обработка updates
	go func() {
		sugaredLogger.Info("Starting updates processing loop")
		for {
			select {
			case update, ok := <-updates:
				if !ok {
					sugaredLogger.Warn("Updates channel closed")
					errCh <- nil
					return
				}
				sugaredLogger.Debugw("Received update from Telegram",
					"update_id", update.UpdateID,
					"has_message", update.Message != nil,
					"has_callback", update.CallbackQuery != nil,
				)

				// Преобразуем update в событие
				event := gateway.ProcessUpdate(update)
				if event != nil {
					// Отправляем событие в worker pool
					workerPool.Submit(event)
					sugaredLogger.Debugw("Event submitted to worker pool",
						"event_type", event.Type(),
						"chat_id", event.ChatID(),
						"user_id", event.UserID(),
					)
				} else {
					sugaredLogger.Debugw("Update processed but no event created",
						"update_id", update.UpdateID,
					)
				}
			case <-ctx.Done():
				sugaredLogger.Info("Updates processing loop stopped")
				return
			}
		}
	}()

	// Обработка результатов
	go func() {
		sugaredLogger.Info("Starting response processing loop")
		for {
			select {
			case response := <-workerPool.Responses():
				if response == nil {
					sugaredLogger.Warn("Received nil response from worker pool")
					continue
				}
				if response.Error != nil {
					sugaredLogger.Errorw("Error processing event",
						"error", response.Error,
						"event_type", response.Event.Type(),
						"chat_id", response.Event.ChatID(),
						"user_id", response.Event.UserID(),
					)
				} else {
					sugaredLogger.Debugw("Event processed successfully",
						"event_type", response.Event.Type(),
						"chat_id", response.Event.ChatID(),
						"user_id", response.Event.UserID(),
						"has_response", response.Response != nil,
					)
				}
			case <-ctx.Done():
				sugaredLogger.Info("Response processing loop stopped")
				return
			}
		}
	}()

	select {
	case err = <-errCh:
		if err != nil {
			sugaredLogger.Fatal(err)
		}
	case <-ctx.Done():
		workerPool.Stop()
		sugaredLogger.Sync()
		sugaredLogger.Infow("app shutdown complete")
	}
}
