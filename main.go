package main

import (
	"context"
	"fmt"
	"github.com/mytelegrambot/bot"
	"github.com/mytelegrambot/config"
	"github.com/mytelegrambot/database"
	"github.com/mytelegrambot/deepseek"
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

	newService := service.NewService(botStorage, r1, b)

	errCh := make(chan error)
	go func() { errCh <- newService.SetBot(ctx) }()
	select {
	case err = <-errCh:
		log.Fatal(err)
	case <-ctx.Done():
		// Дополнительные действия при завершении
	}

}
