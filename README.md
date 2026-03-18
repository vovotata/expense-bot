# Expense Bot

Telegram-бот для приёма и обработки заявок на оплату расходных материалов + Email-watcher для перехвата кодов подтверждения из почты.

## Возможности

- Интерактивный wizard для создания заявок через inline-кнопки
- Два сценария: стандартная заявка (крипта/карта) и пополнение в Антике
- Уведомления в служебный чат с кнопками управления статусом
- Email-watcher: мониторинг почтовых ящиков по IMAP IDLE
- Автоматический парсинг кодов подтверждения и мгновенная доставка в Telegram
- AES-256-GCM шифрование email-паролей
- Rate limiting, whitelist, structured logging

## Стек

- Go 1.22+
- PostgreSQL 16
- gotgbot v2 (Telegram Bot API)
- pgx/v5 + sqlc (база данных)
- go-imap/v2 (IMAP IDLE)
- goose (миграции)
- Docker + Docker Compose

## Быстрый старт

### 1. Настройка

```bash
cp .env.example .env
# Отредактируйте .env: BOT_TOKEN, ADMIN_CHAT_ID, DB_PASSWORD, EMAIL_ENCRYPTION_KEY
```

Для генерации ключа шифрования email:
```bash
openssl rand -hex 32
```

### 2. Запуск через Docker Compose

```bash
docker compose up -d --build
```

Это поднимет:
- PostgreSQL 16
- Миграции (goose)
- Telegram-бот
- Email-watcher

### 3. Запуск для разработки

```bash
# Поднять только БД
docker compose up -d postgres

# Применить миграции
make migrate DATABASE_URL="postgres://bot:password@localhost:5432/expense_bot?sslmode=disable"

# Запустить бота
make run

# Запустить email-watcher (в отдельном терминале)
make run-watcher
```

## Команды бота

| Команда | Описание |
|---------|----------|
| `/start` | Начать создание заявки |
| `/addmail` | Добавить email для мониторинга кодов |
| `/delmail` | Удалить email |
| `/mymails` | Список подключённых ящиков |
| `/codes` | Последние 10 перехваченных кодов |

## Разработка

```bash
# Сборка
make build
make build-watcher

# Тесты
make test

# Линтер
make lint

# Перегенерировать sqlc
make sqlc
```

## Структура проекта

```
cmd/bot/            — точка входа Telegram-бота
cmd/email-watcher/  — точка входа email-watcher
internal/
  bot/              — инициализация бота, хэндлеры, клавиатуры, middleware
  config/           — парсинг конфигурации из env
  domain/           — доменные модели и enum'ы
  emailwatch/       — IMAP IDLE watcher, парсер кодов, шифрование
  fsm/              — Finite State Machine для wizard'а
  notify/           — уведомления в служебный чат
  storage/          — интерфейс и PostgreSQL-реализация
migrations/         — SQL-миграции (goose)
sqlc/               — конфигурация и SQL-запросы для sqlc
```

## Переменные окружения

См. [.env.example](.env.example) для полного списка.
