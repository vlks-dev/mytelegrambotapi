package domain

// User представляет пользователя бота
type User struct {
	ID      int64
	State   string
	Context map[string]interface{}
}

// NewUser создает нового пользователя
func NewUser(id int64) *User {
	return &User{
		ID:      id,
		State:   "default",
		Context: make(map[string]interface{}),
	}
}

// SetState устанавливает состояние пользователя
func (u *User) SetState(state string) {
	u.State = state
}

// GetState возвращает текущее состояние пользователя
func (u *User) GetState() string {
	return u.State
}

// SetContext устанавливает значение в контекст пользователя
func (u *User) SetContext(key string, value interface{}) {
	if u.Context == nil {
		u.Context = make(map[string]interface{})
	}
	u.Context[key] = value
}

// GetContext возвращает значение из контекста пользователя
func (u *User) GetContext(key string) (interface{}, bool) {
	if u.Context == nil {
		return nil, false
	}
	value, ok := u.Context[key]
	return value, ok
}

// ClearContext очищает контекст пользователя
func (u *User) ClearContext() {
	u.Context = make(map[string]interface{})
}
