# Health Hub — Центр здоровья

Monorepo для MVP приложения "Центр здоровья": Go backend + iOS SwiftUI клиент.

## Структура репозитория

```
health-hub/
├── README.md
├── .gitignore
├── docs/                  # Архитектурная документация
├── contracts/             # OpenAPI спецификации
├── server/                # Go backend
│   ├── cmd/api/          # Точка входа API сервера
│   ├── internal/         # Внутренние пакеты
│   └── go.mod
└── ios/                   # iOS SwiftUI приложение
    └── HealthHub/        # Исходники приложения
```

## Требования

- **Go**: 1.23+ (или 1.22+)
- **Xcode**: 16+ для iOS 18
- **PostgreSQL**: опционально (Neon DATABASE_URL)

## Запуск Backend

### Установка зависимостей

```bash
cd server
go mod download
```

### Запуск сервера

```bash
cd server
go run ./cmd/api
```

По умолчанию сервер запускается на `localhost:8080`.

### Конфигурация через переменные окружения

```bash
# Порт (по умолчанию 8080)
PORT=9000 go run ./cmd/api

# Подключение к БД (опционально, пока не требуется)
DATABASE_URL="postgres://user:pass@host/dbname?sslmode=require" go run ./cmd/api
```

### Проверка работоспособности

```bash
curl http://localhost:8080/healthz
# Ответ: {"status":"ok"}
```

### Запуск тестов

```bash
cd server
go test ./...
```

## Запуск iOS приложения

### Создание Xcode проекта

1. Открой Xcode
2. Создай новый проект: **App** (iOS, SwiftUI)
3. Название: `HealthHub`
4. Bundle ID: `com.example.healthhub` (или свой)
5. Минимальная версия: **iOS 18.0**
6. Сохрани проект в папку `ios/`

### Замена исходников

После создания проекта замени содержимое:

- `ios/HealthHub/HealthHub/HealthHubApp.swift` → используй файл из репо
- `ios/HealthHub/HealthHub/ContentView.swift` → используй файл из репо
- Добавь папку `ios/HealthHub/HealthHub/Features/` с экранами

### Запуск

1. Открой `ios/HealthHub/HealthHub.xcodeproj` в Xcode
2. Выбери симулятор (iPhone 16 или новее)
3. Нажми **Run** (Cmd+R)

Приложение откроется с 4 вкладками (заглушками).

## Контракты API

См. `contracts/openapi.yaml` для спецификации эндпоинтов.

Основной эндпоинт:
- `GET /healthz` — проверка состояния сервера

Планируемые эндпоинты (черновики, не реализованы):
- `/v1/profiles/*`
- `/v1/sync/batch`
- `/v1/feed`
- `/v1/metrics/*`
- `/v1/checkins`
- `/v1/chat/messages`
- `/v1/ai/proposals`

## Следующие шаги

Текущий setup включает:
- ✅ Базовый HTTP сервер на Go
- ✅ Минимальное iOS приложение с 4 вкладками
- ✅ OpenAPI контракты (черновик)
- ✅ Unit тесты для healthz

Что НЕ реализовано (ожидаемо):
- ❌ Реальная бизнес-логика (профили, метрики, AI)
- ❌ Подключение к БД / миграции
- ❌ Интеграция HealthKit
- ❌ Аутентификация (SIWA)
- ❌ Deployment (Docker, K8s, CI/CD)
- ❌ Детализированные API схемы в OpenAPI

---

**Разработка**: Все изменения остаются локальными (не коммитятся автоматически).
