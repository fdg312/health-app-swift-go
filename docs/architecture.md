# Архитектура Health Hub MVP

## Обзор

Health Hub — монорепозиторий, содержащий:
- **Backend**: Go HTTP API
- **iOS клиент**: SwiftUI приложение (iOS 18+)
- **Контракты**: OpenAPI спецификации

## Компоненты

### Backend (Go)

- **Фреймворк**: stdlib `net/http` + `http.ServeMux`
- **БД**: PostgreSQL через Neon (опционально для setup)
- **Конфигурация**: переменные окружения (PORT, DATABASE_URL)
- **Структура**:
  - `cmd/api/main.go` — точка входа
  - `internal/config` — загрузка конфигурации
  - `internal/httpserver` — HTTP handlers и middleware

**Эндпоинты (планируемые)**:
- `/healthz` — проверка состояния
- `/v1/profiles` — управление профилями пользователей
- `/v1/sync/batch` — синхронизация данных HealthKit
- `/v1/feed` — лента активности
- `/v1/metrics/daily` — дневные метрики
- `/v1/metrics/hourly` — часовые метрики (не минутные)
- `/v1/checkins` — ручные чекины
- `/v1/chat/messages` — чат с AI
- `/v1/ai/proposals` — предложения AI

### iOS клиент (SwiftUI)

- **Минимальная версия**: iOS 18.0
- **Архитектура**: SwiftUI App с TabView
- **Вкладки**:
  1. **Лента** — активность и события
  2. **Показатели** — графики и статистика
  3. **Активность** — тренировки и шаги
  4. **Чат** — AI-ассистент

**Структура**:
- `HealthHubApp.swift` — entry point
- `ContentView.swift` — TabView
- `Features/` — модули по вкладкам

### Контракты

- **OpenAPI 3.1**: описание всех эндпоинтов
- **JSON Schema**: валидация request/response

## Принципы

1. **Минимализм**: только необходимый функционал для MVP
2. **Безопасность**: валидация входных данных, безопасное хранение токенов
3. **Тестируемость**: unit тесты для критических компонентов
4. **Масштабируемость**: структура готова к расширению

## Аутентификация (будущее)

- Sign in with Apple (SIWA)
- JWT токены для API
- Refresh tokens

## Данные HealthKit

- Синхронизация **по часам** (не по минутам)
- Batch API для оптимизации трафика
- Локальный кеш на iOS

## Deployment (не в текущем setup)

- Backend: Railway/Render/Fly.io
- БД: Neon PostgreSQL
- iOS: TestFlight → App Store

---

**Статус**: Initial setup. Только каркас, без полноценной реализации.
