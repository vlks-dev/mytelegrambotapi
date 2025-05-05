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

type Storage interface {
	Save(ctx context.Context, msg *models.Message) error
	GetMsgIDs(ctx context.Context, id int64) ([]int, error)
	MoveToRecover(ctx context.Context, chatID int64) (bool, error)
}

func (b *BotStorage) MoveToRecover(ctx context.Context, chatID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	tag, err := b.pool.Exec(ctx, "WITH selection AS (DELETE FROM updates_messages WHERE chat_id = $1 RETURNING *) INSERT INTO archive_messages SELECT * FROM selection", chatID)
	if err != nil {
		return false, fmt.Errorf("db operation: move to recover: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return false, nil
	}

	return true, nil
}

func (b *BotStorage) GetMsgIDs(ctx context.Context, id int64) ([]int, error) {
	getCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	rows, err := b.pool.Query(getCtx, "SELECT updates_messages.message_id from updates_messages WHERE chat_id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("db getting messages ids: %w", err)
	}
	defer rows.Close()
	result := make([]int, 0)

	for rows.Next() {
		var ids int
		if err := rows.Scan(&ids); err != nil {
			return nil, fmt.Errorf("db scanning messages ids: %w", err)
		}
		result = append(result, ids)
	}
	deadline, ok := getCtx.Deadline()
	if !ok {
		log.Println("deadline not set for get messages ids")
		return result, nil
	}

	log.Printf("get messages ids, from %d, time left: %s", id, time.Until(deadline))
	return result, nil
}

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
		`INSERT INTO updates_messages (chat_id, message_id, from_id, from_username, text, time_stamp, db_time_stamp) VALUES ($1, $2, $3, $4, $5, $6, current_timestamp)`,
		message.ChatID,
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
		return nil
	}
	log.Printf("saved message %d, time left: %v", message.MessageID, time.Until(deadline))
	return nil
}
