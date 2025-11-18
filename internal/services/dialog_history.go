package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// DialogHistoryService интерфейс для работы с историей диалогов
type DialogHistoryService interface {
	// GetHistory возвращает историю сообщений пользователя (PostgreSQL)
	GetHistory(ctx context.Context, userID int64) ([]domain.Message, error)
	// SaveMessage сохраняет сообщение в историю (PostgreSQL)
	SaveMessage(ctx context.Context, userID int64, message *domain.Message) error
	// GetState возвращает текущее состояние диалога пользователя (Redis)
	GetState(ctx context.Context, userID int64) (string, error)
	// SetState устанавливает состояние диалога пользователя (Redis)
	SetState(ctx context.Context, userID int64, state string) error
}

// dialogHistoryService реализация DialogHistoryService
// PostgreSQL для долгоживущей истории, Redis для краткоживущего состояния
type dialogHistoryService struct {
	pool        *pgxpool.Pool
	redisClient *redis.Client
	logger      *zap.SugaredLogger
}

// NewDialogHistoryService создает новый DialogHistoryService
func NewDialogHistoryService(pool *pgxpool.Pool, redisClient *redis.Client, logger *zap.SugaredLogger) DialogHistoryService {
	return &dialogHistoryService{
		pool:        pool,
		redisClient: redisClient,
		logger:      logger.Named("dialog_history_service"),
	}
}

// GetHistory возвращает историю сообщений пользователя из PostgreSQL
func (s *dialogHistoryService) GetHistory(ctx context.Context, userID int64) ([]domain.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT chat_id, message_id, from_id, from_username, text, time_stamp 
		 FROM updates_messages 
		 WHERE from_id = $1 
		 ORDER BY time_stamp DESC 
		 LIMIT 20`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(
			&msg.ChatID,
			&msg.MessageID,
			&msg.FromID,
			&msg.FromUsername,
			&msg.Text,
			&msg.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	// Переворачиваем порядок, чтобы получить хронологический порядок
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// SaveMessage сохраняет сообщение в историю PostgreSQL
func (s *dialogHistoryService) SaveMessage(ctx context.Context, userID int64, message *domain.Message) error {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`INSERT INTO updates_messages (chat_id, message_id, from_id, from_username, text, time_stamp, db_time_stamp) 
		 VALUES ($1, $2, $3, $4, $5, $6, current_timestamp)`,
		message.ChatID,
		message.MessageID,
		message.FromID,
		message.FromUsername,
		message.Text,
		message.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetState возвращает текущее состояние диалога пользователя из Redis
func (s *dialogHistoryService) GetState(ctx context.Context, userID int64) (string, error) {
	// Если Redis не настроен, используем PostgreSQL как fallback
	if s.redisClient == nil {
		return s.getStateFromPostgreSQL(ctx, userID)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:state:%d", userID)
	state, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Ключ не найден, возвращаем дефолтное состояние
		s.logger.Debugw("State not found in Redis, returning default",
			"user_id", userID,
		)
		return "default", nil
	}
	if err != nil {
		s.logger.Errorw("Failed to get state from Redis, falling back to PostgreSQL",
			"user_id", userID,
			"error", err,
		)
		// Fallback на PostgreSQL при ошибке Redis
		return s.getStateFromPostgreSQL(ctx, userID)
	}

	s.logger.Debugw("Got state from Redis",
		"user_id", userID,
		"state", state,
	)
	return state, nil
}

// getStateFromPostgreSQL получает состояние из PostgreSQL (fallback)
func (s *dialogHistoryService) getStateFromPostgreSQL(ctx context.Context, userID int64) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var state string
	err := s.pool.QueryRow(ctx,
		"SELECT state FROM users WHERE id = $1",
		userID,
	).Scan(&state)

	if err != nil {
		// Если пользователь не найден, возвращаем дефолтное состояние
		return "default", nil
	}

	return state, nil
}

// SetState устанавливает состояние диалога пользователя в Redis
func (s *dialogHistoryService) SetState(ctx context.Context, userID int64, state string) error {
	// Если Redis не настроен, используем PostgreSQL как fallback
	if s.redisClient == nil {
		return s.setStateToPostgreSQL(ctx, userID, state)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:state:%d", userID)
	// Устанавливаем состояние с TTL 24 часа (краткоживущее состояние)
	err := s.redisClient.Set(ctx, key, state, 24*time.Hour).Err()
	if err != nil {
		s.logger.Errorw("Failed to set state in Redis, falling back to PostgreSQL",
			"user_id", userID,
			"state", state,
			"error", err,
		)
		// Fallback на PostgreSQL при ошибке Redis
		return s.setStateToPostgreSQL(ctx, userID, state)
	}

	s.logger.Debugw("Set state in Redis",
		"user_id", userID,
		"state", state,
	)
	return nil
}

// setStateToPostgreSQL сохраняет состояние в PostgreSQL (fallback)
func (s *dialogHistoryService) setStateToPostgreSQL(ctx context.Context, userID int64, state string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (id, state, updated_at) 
		 VALUES ($1, $2, current_timestamp)
		 ON CONFLICT (id) DO UPDATE SET 
		 state = $2, 
		 updated_at = current_timestamp`,
		userID,
		state,
	)

	if err != nil {
		return fmt.Errorf("failed to set state in PostgreSQL: %w", err)
	}

	return nil
}
