-- SQL-скрипт для оптимизации работы с памятью и контекстом диалога
-- Создан для улучшения работы нейросети с историей диалога

-- ============================================
-- 0. Создание таблицы updates_messages (если не существует)
-- ============================================

-- Создаем таблицу для хранения всех сообщений диалога
CREATE TABLE IF NOT EXISTS updates_messages (
    id BIGSERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    message_id INTEGER NOT NULL,
    from_id BIGINT NOT NULL,
    from_username VARCHAR(255),
    text TEXT NOT NULL,
    time_stamp TIMESTAMP NOT NULL,
    db_time_stamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    -- Поле role будет добавлено в следующей секции
    role VARCHAR(20) DEFAULT 'user' NOT NULL,
    -- Поле session_id будет добавлено позже (опционально)
    session_id BIGINT,
    
    -- Индексы для оптимизации запросов
    CONSTRAINT check_role CHECK (role IN ('user', 'assistant', 'system'))
);

-- Создаем уникальный индекс для предотвращения дубликатов сообщений
CREATE UNIQUE INDEX IF NOT EXISTS idx_updates_messages_unique 
ON updates_messages(chat_id, message_id, from_id) 
WHERE message_id > 0;

-- ============================================
-- 1. Улучшение таблицы updates_messages
-- ============================================

-- Добавляем поле role для явного указания роли сообщения (user/assistant)
-- Это улучшит определение ролей и работу с контекстом
-- Если таблица уже существует без поля role, добавляем его
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'updates_messages' 
        AND column_name = 'role'
    ) THEN
        ALTER TABLE updates_messages 
        ADD COLUMN role VARCHAR(20) DEFAULT 'user';
        
        -- Обновляем существующие записи: если from_id = 0 или from_username = 'bot', то role = 'assistant'
        UPDATE updates_messages 
        SET role = 'assistant' 
        WHERE (from_id = 0 OR from_username = 'bot') AND role = 'user';
        
        -- Устанавливаем NOT NULL после заполнения данных
        ALTER TABLE updates_messages 
        ALTER COLUMN role SET NOT NULL;
        
        -- Добавляем CHECK constraint для валидации ролей (если еще не существует)
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE table_name = 'updates_messages' 
            AND constraint_name = 'check_role'
        ) THEN
            ALTER TABLE updates_messages 
            ADD CONSTRAINT check_role CHECK (role IN ('user', 'assistant', 'system'));
        END IF;
    END IF;
END $$;

-- Добавляем индекс для быстрого поиска истории диалога по пользователю и времени
-- Это критично для работы GetHistory
CREATE INDEX IF NOT EXISTS idx_updates_messages_user_time 
ON updates_messages(from_id, time_stamp DESC);

-- Добавляем индекс для поиска по chat_id (для получения всех сообщений в чате)
CREATE INDEX IF NOT EXISTS idx_updates_messages_chat_time 
ON updates_messages(chat_id, time_stamp DESC);

-- Добавляем индекс для поиска по роли (опционально, для аналитики)
CREATE INDEX IF NOT EXISTS idx_updates_messages_role 
ON updates_messages(role) WHERE role = 'assistant';

-- ============================================
-- 2. Улучшение таблицы users
-- ============================================

-- Убеждаемся, что таблица users существует и имеет необходимые поля
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY,
    state VARCHAR(100) DEFAULT 'default' NOT NULL,
    is_admin BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- Добавляем поле для хранения контекста пользователя в JSON формате
    context JSONB DEFAULT '{}'::jsonb
);

-- Создаем индекс для быстрого поиска по состоянию
CREATE INDEX IF NOT EXISTS idx_users_state 
ON users(state);

-- Создаем индекс для поиска администраторов
CREATE INDEX IF NOT EXISTS idx_users_is_admin 
ON users(is_admin) WHERE is_admin = TRUE;

-- GIN индекс для эффективного поиска в JSONB контексте
CREATE INDEX IF NOT EXISTS idx_users_context_gin 
ON users USING GIN (context);

-- ============================================
-- 3. Таблица для хранения сессий диалога (опционально)
-- ============================================

-- Создаем таблицу для группировки сообщений по сессиям
-- Это позволит лучше управлять контекстом и очищать старые сессии
CREATE TABLE IF NOT EXISTS dialog_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    message_count INTEGER DEFAULT 0,
    -- Метаданные сессии (тема, контекст и т.д.)
    metadata JSONB DEFAULT '{}'::jsonb,
    
    CONSTRAINT fk_dialog_sessions_user 
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Индексы для таблицы сессий
CREATE INDEX IF NOT EXISTS idx_dialog_sessions_user 
ON dialog_sessions(user_id, last_activity DESC);

CREATE INDEX IF NOT EXISTS idx_dialog_sessions_chat 
ON dialog_sessions(chat_id, last_activity DESC);

-- Добавляем связь между сообщениями и сессиями (опционально)
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'updates_messages' 
        AND column_name = 'session_id'
    ) THEN
        ALTER TABLE updates_messages 
        ADD COLUMN session_id BIGINT;
        
        -- Внешний ключ на dialog_sessions
        ALTER TABLE updates_messages 
        ADD CONSTRAINT fk_updates_messages_session 
        FOREIGN KEY (session_id) REFERENCES dialog_sessions(id) ON DELETE SET NULL;
        
        -- Индекс для быстрого поиска сообщений по сессии
        CREATE INDEX IF NOT EXISTS idx_updates_messages_session 
        ON updates_messages(session_id);
    END IF;
END $$;

-- ============================================
-- 4. Функция для автоматического обновления updated_at
-- ============================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Триггер для автоматического обновления updated_at в таблице users
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- 5. Функция для очистки старых сообщений (опционально)
-- ============================================

-- Функция для удаления сообщений старше определенного периода
-- Можно вызывать периодически через cron или планировщик задач
CREATE OR REPLACE FUNCTION cleanup_old_messages(days_to_keep INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM updates_messages
    WHERE time_stamp < CURRENT_TIMESTAMP - (days_to_keep || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 6. Представление для удобного получения истории диалога
-- ============================================

-- Создаем представление для упрощения запросов истории
CREATE OR REPLACE VIEW dialog_history_view AS
SELECT 
    um.id,
    um.chat_id,
    um.message_id,
    um.from_id,
    um.from_username,
    um.role,
    um.text,
    um.time_stamp,
    um.session_id,
    u.id AS user_id,
    u.state AS user_state
FROM updates_messages um
LEFT JOIN users u ON um.from_id = u.id
ORDER BY um.time_stamp ASC;

-- ============================================
-- 7. Комментарии к таблицам и полям
-- ============================================

COMMENT ON TABLE updates_messages IS 'Таблица для хранения всех сообщений диалога (пользователь и бот)';
COMMENT ON COLUMN updates_messages.role IS 'Роль сообщения: user (пользователь), assistant (бот), system (система)';
COMMENT ON COLUMN updates_messages.session_id IS 'ID сессии диалога для группировки сообщений';

COMMENT ON TABLE users IS 'Таблица пользователей бота';
COMMENT ON COLUMN users.context IS 'JSON объект с контекстом пользователя (предпочтения, настройки и т.д.)';
COMMENT ON COLUMN users.state IS 'Текущее состояние пользователя в боте';

COMMENT ON TABLE dialog_sessions IS 'Таблица сессий диалога для группировки сообщений по временным периодам';

-- ============================================
-- Готово!
-- ============================================
-- После выполнения этого скрипта:
-- 1. История диалога будет правильно определять роли сообщений
-- 2. Запросы к истории будут оптимизированы через индексы
-- 3. Контекст пользователя можно сохранять в поле context (JSONB)
-- 4. Можно группировать сообщения по сессиям для лучшего управления контекстом
-- ============================================

