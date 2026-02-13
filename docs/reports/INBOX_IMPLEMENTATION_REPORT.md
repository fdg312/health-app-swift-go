# Inbox/Notifications Implementation Report

## Overview

Реализована система server-side уведомлений (Inbox/Notifications) для Health Hub приложения. Уведомления генерируются на основе метрик и чек-инов, хранятся на сервере, отображаются в iOS приложении с бейджем unpread count.

## Changes Summary

### Backend (Go)

#### 1. Configuration (`server/.env.example`, `config/config.go`)
- Добавлены ENV переменные:
  - `NOTIFICATIONS_MAX_PER_DAY=4`
  - `DEFAULT_SLEEP_MIN_MINUTES=420`
  - `DEFAULT_STEPS_MIN=6000`
  - `DEFAULT_ACTIVE_ENERGY_MIN_KCAL=200`

#### 2. SQL Schema (`docs/sql/notifications.sql`)
- Таблица `notifications`:
  - `id` (UUID), `profile_id` (FK), `kind`, `title`, `body`, `source_date`, `severity`, `created_at`, `read_at`
  - UNIQUE constraint на `(profile_id, kind, source_date)` для предотвращения дубликатов
  - Индексы на `(profile_id, created_at)` и `(profile_id, read_at)`

#### 3. Storage Interface (`server/internal/storage/storage.go`)
- Добавлен `NotificationsStorage` interface:
  - `CreateNotification` (upsert по unique key)
  - `ListNotifications` (с фильтром onlyUnread)
  - `UnreadCount`
  - `MarkRead` / `MarkAllRead`
- Struct `Notification` с полями

#### 4. Storage Implementations
- **Memory** (`server/internal/storage/memory/notifications.go`):
  - In-memory хранение с индексами по profile_id
  - Unique keys map для upsert логики
- **Postgres** (`server/internal/storage/postgres/notifications.go`):
  - SQL queries с ON CONFLICT для upsert
  - Эффективные запросы с индексами
- Интеграция в `memory.go` и `postgres.go`

#### 5. Notifications Service (`server/internal/notifications/`)
- **models.go**: DTOs для всех endpoints
- **service.go**: Бизнес-логика генерации уведомлений:
  - **Правила** (без AI, объяснимые):
    - `low_sleep` (warn): < sleep_min_minutes
    - `low_activity` (info/warn): < steps_min ИЛИ < active_energy_min_kcal
    - `missing_morning_checkin` (info): date == today && time > 12:00 && нет morning checkin
    - `missing_evening_checkin` (info): date == today && time > 21:30 && нет evening checkin
  - **Приоритет**: warn > info
  - **Лимит**: max N уведомлений на source_date (configurable)
  - **Idempotent**: unique constraint предотвращает дубли
- **handlers.go**: HTTP handlers для 5 endpoints
- **handlers_test.go**: Unit tests (все проходят ✅)

#### 6. Routing (`server/internal/httpserver/server.go`)
- Зарегистрированы endpoints:
  - `GET /v1/inbox`
  - `GET /v1/inbox/unread-count`
  - `POST /v1/inbox/mark-read`
  - `POST /v1/inbox/mark-all-read`
  - `POST /v1/inbox/generate`

#### 7. OpenAPI (`contracts/openapi.yaml`)
- Version bump: `0.8.0` → `0.9.0`
- Добавлены schemas: `Notification`, `InboxListResponse`, `UnreadCountResponse`, `MarkReadRequest/Response`, `MarkAllReadRequest/Response`, `GenerateNotificationsRequest/Response`
- Endpoints с полным описанием параметров и responses

### iOS (Swift)

#### 1. Models (`ios/HealthHub/HealthHub/Models/NotificationDTO.swift`)
- `NotificationDTO` (Codable, Identifiable)
- Request/Response structs для всех API endpoints

#### 2. APIClient (`ios/HealthHub/HealthHub/Core/Networking/APIClient.swift`)
- Методы:
  - `fetchInbox(profileId, onlyUnread, limit, offset)`
  - `fetchUnreadCount(profileId)`
  - `markNotificationsRead(profileId, ids)`
  - `markAllNotificationsRead(profileId)`
  - `generateNotifications(profileId, date, timeZone, now, thresholds)`

#### 3. UI - FeedView (`ios/HealthHub/HealthHub/Features/Feed/FeedView.swift`)
- **Кнопка Inbox**: Bell icon в toolbar (topBarLeading) с красным бейджем unread count
- **State**: `@State showInbox`, `@State unreadCount`
- **Логика**:
  - `loadUnreadCount()` вызывается при загрузке и обновлении feed
  - `generateNotificationsForToday()` вызывается автоматически для today после loadFeedDay (best-effort)
- **Sheet**: `InboxSheet(profileId:onDismiss:)` открывается по нажатию

#### 4. UI - InboxSheet (`ios/HealthHub/HealthHub/Features/Feed/InboxSheet.swift`)
- **Список уведомлений**:
  - Отображение: title, body, source_date, created_at (relative time)
  - Визуальная индикация: непрочитанные (bold, синяя точка, severity color), прочитанные (dimmed)
- **Действия**:
  - Swipe action "Прочитано" (для одного уведомления)
  - Toolbar button "Отметить всё прочитанным" (если есть непрочитанные)
- **Empty states**: "Нет уведомлений", error handling
- **Refresh**: pull-to-refresh

### Documentation

#### README.md
- Добавлена `docs/sql/notifications.sql` в список SQL setup
- Curl примеры для всех Inbox endpoints (с jq и heredoc)
- Секция "iOS: Inbox (Уведомления)" с подробным описанием:
  - Как использовать (шаги)
  - Правила генерации
  - ENV конфигурация
  - Тестирование

## API Endpoints

### 1. GET /v1/inbox
**Query params**: `profile_id`, `only_unread` (bool), `limit`, `offset`  
**Response**: `{ "notifications": [...] }`

### 2. GET /v1/inbox/unread-count
**Query params**: `profile_id`  
**Response**: `{ "unread": 3 }`

### 3. POST /v1/inbox/mark-read
**Body**: `{ "profile_id": "...", "ids": ["uuid", ...] }`  
**Response**: `{ "marked": 2 }`

### 4. POST /v1/inbox/mark-all-read
**Body**: `{ "profile_id": "..." }`  
**Response**: `{ "marked": 10 }`

### 5. POST /v1/inbox/generate
**Body**: `{ "profile_id": "...", "date": "YYYY-MM-DD", "client_time_zone": "...", "now": "...", "thresholds": {...} }`  
**Response**: `{ "created": 1, "updated": 0, "skipped": 0 }`

## Testing

### Go Tests
```bash
cd server
go test ./... -v
```

**Results**:
- ✅ `internal/notifications`: All tests pass (7/7)
- ✅ `internal/checkins`: Pass
- ✅ `internal/feed`: Pass
- ✅ `internal/httpserver`: Pass
- ✅ `internal/metrics`: Pass
- ✅ `internal/profiles`: Pass
- ✅ `internal/sources`: Pass
- ⚠️ `internal/reports`: 1 test fail (PDF font missing - known issue, not related to this feature)

### Manual Testing (iOS)

1. **Синхронизация HealthKit**:
   - Открой "Показатели" → "Sync Today" (или "Sync Last 7 Days")
   - Данные синкаются на сервер

2. **Генерация уведомлений**:
   - Открой "Лента" (FeedView)
   - Генерация происходит автоматически для today
   - Или вручную через curl: `POST /v1/inbox/generate`

3. **Просмотр Inbox**:
   - В FeedView нажми bell icon (колокольчик)
   - Увидишь список уведомлений
   - Бейдж показывает количество непрочитанных

4. **Отметить прочитанным**:
   - Swipe влево по уведомлению → "Прочитано"
   - Или "Отметить всё прочитанным" в toolbar
   - Бейдж уменьшится

### Curl Examples

```bash
# Setup
PROFILE_ID=$(curl -s http://localhost:8080/v1/profiles | jq -r '.profiles[0].id')

# Generate notifications for today
jq -n --arg pid "$PROFILE_ID" '{
  profile_id:$pid,
  date:"2026-02-13",
  client_time_zone:"Europe/Moscow",
  now:"2026-02-13T22:00:00+03:00",
  thresholds:{sleep_min_minutes:420,steps_min:6000,active_energy_min_kcal:200}
}' | curl -X POST http://localhost:8080/v1/inbox/generate \
  -H 'Content-Type: application/json' --data-binary @- | jq .

# Get unread count
curl "http://localhost:8080/v1/inbox/unread-count?profile_id=$PROFILE_ID" | jq .

# List all notifications
curl "http://localhost:8080/v1/inbox?profile_id=$PROFILE_ID" | jq .

# Mark all as read
curl -X POST http://localhost:8080/v1/inbox/mark-all-read \
  -H 'Content-Type: application/json' \
  --data-binary @- <<JSON
{
  "profile_id": "$PROFILE_ID"
}
JSON
```

## Files Changed/Created

### Backend
- ✅ `server/.env.example` (modified)
- ✅ `server/internal/config/config.go` (modified)
- ✅ `docs/sql/notifications.sql` (created)
- ✅ `server/internal/storage/storage.go` (modified)
- ✅ `server/internal/storage/memory/notifications.go` (created)
- ✅ `server/internal/storage/memory/memory.go` (modified)
- ✅ `server/internal/storage/postgres/notifications.go` (created)
- ✅ `server/internal/storage/postgres/postgres.go` (modified)
- ✅ `server/internal/notifications/models.go` (created)
- ✅ `server/internal/notifications/service.go` (created)
- ✅ `server/internal/notifications/handlers.go` (created)
- ✅ `server/internal/notifications/handlers_test.go` (created)
- ✅ `server/internal/httpserver/server.go` (modified)
- ✅ `contracts/openapi.yaml` (modified, v0.9.0)

### iOS
- ✅ `ios/HealthHub/HealthHub/Models/NotificationDTO.swift` (created)
- ✅ `ios/HealthHub/HealthHub/Core/Networking/APIClient.swift` (modified)
- ✅ `ios/HealthHub/HealthHub/Features/Feed/FeedView.swift` (modified)
- ✅ `ios/HealthHub/HealthHub/Features/Feed/InboxSheet.swift` (created)

### Documentation
- ✅ `README.md` (modified)

## Key Features

1. **Server-side storage**: Уведомления хранятся на сервере, не в iOS локально
2. **Idempotent generation**: Unique constraint предотвращает дубликаты при повторной генерации
3. **Prioritization**: warn уведомления приоритетнее info
4. **Rate limiting**: Максимум N уведомлений на date (configurable)
5. **Time-aware**: Проверка времени дня для missing checkin notifications
6. **Backward compatible**: Старые клиенты без Inbox продолжат работать
7. **iOS badge**: Красный бейдж с unread count на bell icon
8. **Pull-to-refresh**: В InboxSheet для обновления списка
9. **Swipe actions**: Удобное управление одним жестом
10. **No AI**: Простые, объяснимые правила генерации

## Migration Steps (для продакшена)

1. Выполнить `docs/sql/notifications.sql` на Postgres
2. Добавить ENV переменные в .env (или использовать defaults)
3. Обновить код сервера и iOS приложения
4. Генерация уведомлений будет происходить автоматически при загрузке feed для today

## Future Improvements (not in this scope)

- Push notifications (APNS)
- Кастомные threshold'ы per user
- Больше типов уведомлений (goals, achievements, reminders)
- Notification preferences/settings
- Batch delete / bulk actions
- Notification history pagination (сейчас limit 50)

## Confirmation

- ✅ **NO git commit** (as requested)
- ✅ **All Go tests pass** (except 1 old PDF test - known issue)
- ✅ **Minimal changes** (no large refactoring)
- ✅ **iOS 18+** (using native components)
- ✅ **No third-party dependencies** (only stdlib + Apple frameworks)
- ✅ **No SIWA/JWT/AI/local push** (server-side only)
- ✅ **Curl examples with jq/heredoc** (no single quotes for variables)

---

**Implementation Status**: ✅ COMPLETE

All 8 parts completed successfully:
1. ENV + config ✅
2. SQL schema ✅
3. Storage interface + implementations ✅
4. Service + handlers + tests ✅
5. Routing + OpenAPI ✅
6. iOS models + APIClient + UI ✅
7. README + curl examples ✅
8. Run tests and verify ✅
