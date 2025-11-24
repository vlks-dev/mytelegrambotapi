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
	// Фильтрует сообщения после последнего сброса контекста
	GetHistory(ctx context.Context, userID int64) ([]domain.Message, error)
	// SaveMessage сохраняет сообщение в историю (PostgreSQL)
	SaveMessage(ctx context.Context, userID int64, message *domain.Message) error
	// GetState возвращает текущее состояние диалога пользователя (Redis)
	GetState(ctx context.Context, userID int64) (string, error)
	// SetState устанавливает состояние диалога пользователя (Redis)
	SetState(ctx context.Context, userID int64, state string) error
	// ResetDialogContext сбрасывает контекст диалога (устанавливает timestamp сброса)
	// Сообщения остаются в БД, но не будут использоваться в контексте AI
	ResetDialogContext(ctx context.Context, userID int64) error
	// GetLastResetTime возвращает время последнего сброса контекста
	GetLastResetTime(ctx context.Context, userID int64) (time.Time, error)
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
// Включает как сообщения пользователя, так и ответы бота в том же чате
// Фильтрует сообщения после последнего сброса контекста
func (s *dialogHistoryService) GetHistory(ctx context.Context, userID int64) ([]domain.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Получаем время последнего сброса контекста
	lastResetTime, err := s.GetLastResetTime(ctx, userID)
	if err != nil {
		s.logger.Warnw("Failed to get last reset time, using all history", "error", err)
		// Если не удалось получить время сброса, используем нулевое время (все сообщения)
		lastResetTime = time.Time{}
	}

	// Получаем все сообщения из чатов пользователя (включая ответы бота)
	// Фильтруем только те, которые были после последнего сброса
	// Роль определяется динамически в ai_service.go на основе from_id и from_username
	query := `SELECT um.chat_id, um.message_id, um.from_id, um.from_username, um.text, um.time_stamp
		 FROM updates_messages um
		 WHERE um.chat_id IN (
			 SELECT DISTINCT chat_id 
			 FROM updates_messages 
			 WHERE from_id = $1
		 )`

	args := []interface{}{userID}

	// Если есть время сброса, фильтруем по нему
	if !lastResetTime.IsZero() {
		query += ` AND um.time_stamp > $2`
		args = append(args, lastResetTime)
	}

	query += ` ORDER BY um.time_stamp DESC LIMIT 30`

	rows, err := s.pool.Query(ctx, query, args...)
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

	s.logger.Debugw("Retrieved dialog history",
		"user_id", userID,
		"total_messages", len(messages),
	)

	return messages, nil
}

// SaveMessage сохраняет сообщение в историю PostgreSQL
func (s *dialogHistoryService) SaveMessage(ctx context.Context, userID int64, message *domain.Message) error {
	ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Определяем роль сообщения: если from_id == 0 или from_username == "bot", то это ответ ассистента
	role := "user"
	if message.FromID == 0 || message.FromUsername == "bot" {
		role = "assistant"
	}

	// Пытаемся вставить с role, если поле существует
	// Используем COALESCE для обратной совместимости, если поле role еще не создано
	_, err := s.pool.Exec(ctx,
		`INSERT INTO updates_messages (chat_id, message_id, from_id, from_username, text, time_stamp, db_time_stamp, role) 
		 VALUES ($1, $2, $3, $4, $5, $6, current_timestamp, $7)
		 ON CONFLICT DO NOTHING`,
		message.ChatID,
		message.MessageID,
		message.FromID,
		message.FromUsername,
		message.Text,
		message.Timestamp,
		role,
	)

	// Если ошибка связана с отсутствием поля role, пробуем без него (для обратной совместимости)
	if err != nil {
		// Пробуем вставить без role
		_, err2 := s.pool.Exec(ctx,
			`INSERT INTO updates_messages (chat_id, message_id, from_id, from_username, text, time_stamp, db_time_stamp) 
			 VALUES ($1, $2, $3, $4, $5, $6, current_timestamp)
			 ON CONFLICT DO NOTHING`,
			message.ChatID,
			message.MessageID,
			message.FromID,
			message.FromUsername,
			message.Text,
			message.Timestamp,
		)
		if err2 != nil {
			return fmt.Errorf("failed to save message: %w", err2)
		}
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

// ResetDialogContext сбрасывает контекст диалога (устанавливает timestamp сброса)
// Сообщения остаются в БД, но не будут использоваться в контексте AI
func (s *dialogHistoryService) ResetDialogContext(ctx context.Context, userID int64) error {
	resetTime := time.Now()

	// Если Redis не настроен, используем PostgreSQL как fallback
	if s.redisClient == nil {
		return s.setResetTimeToPostgreSQL(ctx, userID, resetTime)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:reset_time:%d", userID)
	// Устанавливаем время сброса с TTL 365 дней (долгоживущее, но не вечное)
	err := s.redisClient.Set(ctx, key, resetTime.Format(time.RFC3339), 2*24*time.Hour).Err()
	if err != nil {
		s.logger.Errorw("Failed to set reset time in Redis, falling back to PostgreSQL",
			"user_id", userID,
			"error", err,
		)
		// Fallback на PostgreSQL при ошибке Redis
		return s.setResetTimeToPostgreSQL(ctx, userID, resetTime)
	}

	s.logger.Debugw("Reset dialog context",
		"user_id", userID,
		"reset_time", resetTime,
	)
	return nil
}

// GetLastResetTime возвращает время последнего сброса контекста
func (s *dialogHistoryService) GetLastResetTime(ctx context.Context, userID int64) (time.Time, error) {
	// Если Redis не настроен, используем PostgreSQL как fallback
	if s.redisClient == nil {
		return s.getResetTimeFromPostgreSQL(ctx, userID)
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	key := fmt.Sprintf("user:reset_time:%d", userID)
	resetTimeStr, err := s.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// Ключ не найден, возвращаем нулевое время (нет сброса)
		return time.Time{}, nil
	}
	if err != nil {
		s.logger.Errorw("Failed to get reset time from Redis, falling back to PostgreSQL",
			"user_id", userID,
			"error", err,
		)
		// Fallback на PostgreSQL при ошибке Redis
		return s.getResetTimeFromPostgreSQL(ctx, userID)
	}

	resetTime, err := time.Parse(time.RFC3339, resetTimeStr)
	if err != nil {
		s.logger.Errorw("Failed to parse reset time from Redis",
			"user_id", userID,
			"reset_time_str", resetTimeStr,
			"error", err,
		)
		return time.Time{}, nil
	}

	return resetTime, nil
}

// setResetTimeToPostgreSQL сохраняет время сброса в PostgreSQL (fallback)
func (s *dialogHistoryService) setResetTimeToPostgreSQL(ctx context.Context, userID int64, resetTime time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Сохраняем время сброса в поле context таблицы users как JSON
	_, err := s.pool.Exec(ctx,
		`INSERT INTO users (id, context, updated_at) 
		 VALUES ($1, jsonb_build_object('last_reset_time', $2::text), current_timestamp)
		 ON CONFLICT (id) DO UPDATE SET 
		 context = jsonb_set(
			 COALESCE(users.context, '{}'::jsonb),
			 '{last_reset_time}',
			 to_jsonb($2::text)
		 ),
		 updated_at = current_timestamp`,
		userID,
		resetTime.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to set reset time in PostgreSQL: %w", err)
	}

	return nil
}

// getResetTimeFromPostgreSQL получает время сброса из PostgreSQL (fallback)
func (s *dialogHistoryService) getResetTimeFromPostgreSQL(ctx context.Context, userID int64) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var resetTimeStr string
	err := s.pool.QueryRow(ctx,
		`SELECT context->>'last_reset_time' 
		 FROM users 
		 WHERE id = $1`,
		userID,
	).Scan(&resetTimeStr)

	if err != nil {
		// Если пользователь не найден или поле пустое, возвращаем нулевое время
		return time.Time{}, nil
	}

	if resetTimeStr == "" {
		return time.Time{}, nil
	}

	resetTime, err := time.Parse(time.RFC3339, resetTimeStr)
	if err != nil {
		s.logger.Warnw("Failed to parse reset time from PostgreSQL",
			"user_id", userID,
			"reset_time_str", resetTimeStr,
			"error", err,
		)
		return time.Time{}, nil
	}

	return resetTime, nil
}
