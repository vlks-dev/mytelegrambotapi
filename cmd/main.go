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
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
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
					// Отправляем заглушку в чат и получаем messageID
					var placeholder string
					switch event.(type) {
					case domain.TextMessageReceived:
						placeholder = "Сообщение получено, обрабатываю..."
					case domain.VoiceReceived:
						placeholder = "Голосовое сообщение получено, распознаю и обрабатываю..."
					case domain.VideoReceived:
						placeholder = "Видео получено, обрабатываю..."
					default:
						placeholder = "Обрабатываю ваш запрос..."
					}

					placeholderMsgID := 0
					if placeholder != "" {
						msg, err := presenter.Send(ctx, event.ChatID(), domain.NewTextResponse(placeholder))
						if err != nil {
							sugaredLogger.Warnw("Failed to send processing placeholder", "error", err, "chat_id", event.ChatID())
						} else if msg != nil {
							placeholderMsgID = msg.MessageID
						}
					}

					// Отправляем событие в worker pool вместе с placeholderMsgID
					workerPool.Submit(event, placeholderMsgID)
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
				chatID := response.Event.ChatID()
				placeholderID := response.PlaceholderMessageID
				// Сформируем текст для обновления заглушки
				var newText string
				if response.Error != nil {
					newText = "Ошибка при обработке: " + response.Error.Error()
				} else if response.Response == nil {
					newText = "Обработка завершена."
				} else if response.Response.Text != "" {
					newText = response.Response.Text
				} else if response.Response.File != nil {
					newText = "Готово — отправляю файл..."
				} else {
					newText = "Обработка завершена."
				}

				// Если была заглушка — отредактируем её
				if placeholderID != 0 {
					if err := presenter.EditMessage(ctx, chatID, placeholderID, newText); err != nil {
						sugaredLogger.Warnw("Failed to edit placeholder message", "error", err, "chat_id", chatID, "message_id", placeholderID)
					}
				}

				// Если нужно отправить файл — отправляем и обновляем заглушку по результату
				if response.Error == nil && response.Response != nil && response.Response.File != nil {
					if _, err := presenter.Send(ctx, chatID, response.Response); err != nil {
						sugaredLogger.Errorw("Failed to send file response", "error", err, "chat_id", chatID)
						if placeholderID != 0 {
							_ = presenter.EditMessage(ctx, chatID, placeholderID, "Ошибка при отправке файла")
						}
					} else {
						if placeholderID != 0 {
							_ = presenter.EditMessage(ctx, chatID, placeholderID, "Файл отправлен")
						}
					}
				}

				sugaredLogger.Debugw("Event processed",
					"event_type", response.Event.Type(),
					"chat_id", chatID,
					"user_id", response.Event.UserID(),
					"has_response", response.Response != nil,
					"error", response.Error,
				)
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
