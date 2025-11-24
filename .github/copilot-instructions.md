# Copilot instructions for mytelegrambotapi

Короткие и практичные заметки для AI-агентов, работающих с этим репозиторием.

1. Цель проекта
- Это Telegram-бот на Go, который принимает обновления от Telegram, нормализует их в доменные события, маршрутизирует в use-cases и отправляет ответы через Telegram API.

2. Большая архитектура (кратко)
- Входящие обновления: `cmd/main.go` получает `updates` из `bot.GetUpdates` и передаёт их в `internal/inbound/TelegramGateway` (`internal/inbound/gateway.go`) — он нормализует `tgbotapi.Update` в `internal/domain.Event`.
- Маршрутизация: `internal/routing/EventRouter` (`internal/routing/router.go`) решает, какой use-case выполнять (команда, текст, voice, callback и т.д.).
- Use-cases: папка `internal/usecases/*` содержит обработчики бизнес-логики. Use-case должен реализовывать интерфейс `UseCaseHandler` с методом `Execute(ctx, event) (*domain.BotResponse, error)`.
- Конкурентная обработка: `internal/concurrency` содержит `EventProcessor` и `WorkerPool` — события обрабатываются асинхронно воркерами.
- Выход (презентер): `internal/outbound/TelegramPresenter` (`internal/outbound/presenter.go`) преобразует `domain.BotResponse` в вызовы Telegram через `bot.ExtendedBotAPI`.
- Интеграции: база данных (Postgres, `database/pgxpool.go`), Redis (`database/redis.go`), внешний AI (DeepSeek, `deepseek/R1Connect.go`), и Whisper (speech-to-text) через `internal/services`.

3. Запуск и сборка
- Локально: `go run ./cmd` — основной исполняемый пакет в `cmd/main.go`.
- Сборка: `go build ./...` или `go build -o bin/mytelegrambotapi ./cmd`.
- В контейнерах: в репозитории есть `docker-compose.yml` (grafana/loki/promtail), но конкретного сервисного образа приложения нет — контейнеризация приложения не предоставлена по умолчанию.

4. Переменные окружения (важные, смотрите `config/config.go`)
- `TOKEN` — Telegram bot token (обязательный).
- `CONNECTION_STRING` — строка подключения к Postgres.
- `MAX_PGX_CONN`, `MAX_PGX_CONN_IDLE_TIME`, `MAX_PGX_CONN_LIFETIME`, `HEALTH_CHECK_PERIOD` — настройки pool'а.
- `R1_TOKEN`, `R1_PRO_TOKEN`, `AI_API_URL` — настройки DeepSeek/AI.
- `BOT_ENV` — `debug` для включения отладочного режима.
- `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB` — Redis (опционально; при ошибке подключение игнорируется и приложение продолжает работать).
- `WHISPER_SERVICE_URL` — URL сервиса распознавания голоса (whisper).
- `TEMP_DIR` — каталог для временных файлов (по умолчанию `/tmp`).

5. Важные проекты и конвенции
- Доменные объекты и события находятся в `internal/domain` — используйте фабричные функции `NewXXX` для создания событий/ответов.
- Use-case handler'ы регистрируются в `cmd/main.go` через `router.Register*` (см. `cmd/main.go`), поэтому изменения в регистрации — место конфигурации поведения.
- Логирование: везде используется `zap.SugaredLogger` (см. `logger.NewLogger`), передавайте `logger.Named("component")` для контекстной информации.
- Работа с Telegram API: вспомогательные методы в `bot/bot.go` (SendMessage, SendPhoto, DownloadFile и т.д.). Для отправки ответов используйте `internal/outbound/TelegramPresenter`.

6. Примеры паттернов кода (копировать/следовать)
- Преобразование Update → Event: `internal/inbound/gateway.go:ProcessUpdate`.
- Маршрутизация: `internal/routing/router.go:Route` — см. как проверяются `admin_` префиксы и fallback.
- Формирование ответа: `internal/domain/bot_response.go` содержит вспомогательные конструкторы `NewTextResponse`, `NewFileResponse`, `NewCallbackAnswer`.

7. Примечания по разработке
- Проверяйте наличие `.env` рядом с проектом: `config.LoadEnvCfg(".env")` — без него запуск выдаст ошибку.
- Redis не критичен: при ошибке подключения приложение логирует предупреждение и продолжает.
- Нет встроенных unit-тестов в репозитории (на момент создания инструкции). Для проверки стиля используйте `gofmt`, `go vet` и `golangci-lint` по необходимости.

8. Что важно для AI-агента при изменениях
- Не менять публичные контракты событий в `internal/domain` — от этого зависит маршрутизация и presenter.
- Регистрация use-cases должна делаться централизованно в `cmd/main.go`.
- При добавлении новых типов ответов — обновлять `internal/outbound/presenter.go`.

Если что-то не ясно или хотите, чтобы я включил дополнительные примеры (конкретные use-cases, SQL-запросы, or CI-команды), скажите — доработаю инструкцию.
