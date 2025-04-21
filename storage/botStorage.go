package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mytelegrambot/config"
	"github.com/mytelegrambot/models"
	"log"
	"time"
)

type BotStorage struct {
	pool   *pgxpool.Pool
	config *config.Config
}

func NewBotStorage(pool *pgxpool.Pool, config *config.Config) *BotStorage {
	return &BotStorage{pool: pool, config: config}
}

func (b *BotStorage) Save(ctx context.Context, message *models.Message) error {
	saveCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	exec, err := b.pool.Exec(
		ctx,
		`INSERT INTO updates_messages (message_id, from_id, from_username, text, time_stamp, db_time_stamp) VALUES ($1, $2, $3, $4, $5, current_timestamp)`,
		message.MessageID,
		message.FromID,
		message.FromUsername,
		message.Text,
		message.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("storage insert updates err: %v", err)
	}

	if exec.RowsAffected() != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", exec.RowsAffected())
	}

	deadline, ok := saveCtx.Deadline()
	if !ok {
		log.Println("no deadline in context")
	}
	log.Printf("saved message %d, time left: %v", message.MessageID, time.Until(deadline))
	return nil
}
