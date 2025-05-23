package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mytelegrambot/bot"
	"github.com/mytelegrambot/config"
	"github.com/mytelegrambot/database"
	"github.com/mytelegrambot/deepseek"
	"github.com/mytelegrambot/handlers"
	"github.com/mytelegrambot/logger"
	"github.com/mytelegrambot/service"
	"github.com/mytelegrambot/storage"
	"log"
	"os"
	"os/signal"
	"syscall"
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

	sugaredLogger := logger.NewLogger(botCfg, "main")
	defer sugaredLogger.Sync()

	sugaredLogger.Infoln(
		"tg bot startup by vlks",
		"configurated by .env",
	)

	b, err := bot.NewBot(botCfg)
	if err != nil {
		log.Fatal(err)
	}

	r1 := deepseek.NewR1(botCfg)

	pool, err := database.GetPool(ctx, botCfg)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	botStorage := storage.NewBotStorage(pool, botCfg)

	newService := service.NewService(sugaredLogger, botStorage, r1, b)

	handler := handlers.NewBotHandler(newService)

	router := gin.New()
	router.Use(gin.Recovery())
	engine := router.With()

	handler.RegisterRoutes(engine)

	errCh := make(chan error)

	go func() {
		sugaredLogger.Infow("starting web server", "addr", ":8080")
		errCh <- router.Run(":8080")
	}()

	go func() {
		sugaredLogger.Infow("waiting for incoming bot requests...", "debug", botCfg.BotEnv)
		errCh <- newService.SetBot(ctx)
	}()

	select {
	case err = <-errCh:
		log.Fatal(err)
	case <-ctx.Done():
		pool.Close()
		sugaredLogger.Infow("app shutdown complete")

	}

}
