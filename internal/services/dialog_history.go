package services

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
	"time"
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
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
	// TODO: добавить Redis клиент для состояния
	// redisClient *redis.Client
}

// NewDialogHistoryService создает новый DialogHistoryService
func NewDialogHistoryService(pool *pgxpool.Pool, logger *zap.SugaredLogger) DialogHistoryService {
	return &dialogHistoryService{
		pool:   pool,
		logger: logger.Named("dialog_history_service"),
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
// TODO: реализовать с Redis клиентом
func (s *dialogHistoryService) GetState(ctx context.Context, userID int64) (string, error) {
	// Временная реализация: получаем из PostgreSQL
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
// TODO: реализовать с Redis клиентом
func (s *dialogHistoryService) SetState(ctx context.Context, userID int64, state string) error {
	// Временная реализация: сохраняем в PostgreSQL
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
		return fmt.Errorf("failed to set state: %w", err)
	}

	return nil
}

// dialogHistoryServiceStub заглушка реализации DialogHistoryService
type dialogHistoryServiceStub struct{}

// NewDialogHistoryServiceStub создает заглушку DialogHistoryService
func NewDialogHistoryServiceStub() DialogHistoryService {
	return &dialogHistoryServiceStub{}
}

// GetHistory заглушка метода получения истории
func (s *dialogHistoryServiceStub) GetHistory(ctx context.Context, userID int64) ([]domain.Message, error) {
	return []domain.Message{}, nil
}

// SaveMessage заглушка метода сохранения сообщения
func (s *dialogHistoryServiceStub) SaveMessage(ctx context.Context, userID int64, message *domain.Message) error {
	return nil
}

// GetState заглушка метода получения состояния
func (s *dialogHistoryServiceStub) GetState(ctx context.Context, userID int64) (string, error) {
	return "default", nil
}

// SetState заглушка метода установки состояния
func (s *dialogHistoryServiceStub) SetState(ctx context.Context, userID int64, state string) error {
	return nil
}

