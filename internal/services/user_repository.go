package services

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vlks-dev/mytelegrambotapi/internal/domain"
	"go.uber.org/zap"
)

// UserRepository интерфейс для работы с пользователями
type UserRepository interface {
	// GetByID возвращает пользователя по ID
	GetByID(ctx context.Context, userID int64) (*domain.User, error)
	// Save сохраняет пользователя
	Save(ctx context.Context, user *domain.User) error
	// IsAdmin проверяет, является ли пользователь администратором
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

// userRepository реализация UserRepository на PostgreSQL
type userRepository struct {
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
}

// NewUserRepository создает новый UserRepository
func NewUserRepository(pool *pgxpool.Pool, logger *zap.SugaredLogger) UserRepository {
	return &userRepository{
		pool:   pool,
		logger: logger.Named("user_repository"),
	}
}

// GetByID возвращает пользователя по ID
func (r *userRepository) GetByID(ctx context.Context, userID int64) (*domain.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var state string
	var isAdmin bool

	err := r.pool.QueryRow(ctx,
		"SELECT state, is_admin FROM users WHERE id = $1",
		userID,
	).Scan(&state, &isAdmin)

	if err != nil {
		// Если пользователь не найден, создаем нового
		user := domain.NewUser(userID)
		user.SetState("default")
		return user, nil
	}

	user := domain.NewUser(userID)
	user.SetState(state)
	if isAdmin {
		user.SetContext("is_admin", true)
	}

	return user, nil
}

// Save сохраняет пользователя
func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	isAdmin := false
	if val, ok := user.GetContext("is_admin"); ok {
		if b, ok := val.(bool); ok {
			isAdmin = b
		}
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, state, is_admin, updated_at) 
		 VALUES ($1, $2, $3, current_timestamp)
		 ON CONFLICT (id) DO UPDATE SET 
		 state = $2, 
		 is_admin = $3, 
		 updated_at = current_timestamp`,
		user.ID,
		user.GetState(),
		isAdmin,
	)

	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

// IsAdmin проверяет, является ли пользователь администратором
func (r *userRepository) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var isAdmin bool
	err := r.pool.QueryRow(ctx,
		"SELECT is_admin FROM users WHERE id = $1",
		userID,
	).Scan(&isAdmin)

	if err != nil {
		// Если пользователь не найден, возвращаем false
		return false, nil
	}

	return isAdmin, nil
}

// userRepositoryStub заглушка реализации UserRepository
type userRepositoryStub struct{}

// NewUserRepositoryStub создает заглушку UserRepository
func NewUserRepositoryStub() UserRepository {
	return &userRepositoryStub{}
}

// GetByID заглушка метода получения пользователя
func (r *userRepositoryStub) GetByID(ctx context.Context, userID int64) (*domain.User, error) {
	return domain.NewUser(userID), nil
}

// Save заглушка метода сохранения пользователя
func (r *userRepositoryStub) Save(ctx context.Context, user *domain.User) error {
	return nil
}

// IsAdmin заглушка метода проверки прав администратора
func (r *userRepositoryStub) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	return false, nil
}
