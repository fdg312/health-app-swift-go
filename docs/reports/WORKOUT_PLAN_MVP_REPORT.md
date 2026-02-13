# Workout Plan MVP Implementation Report

**Дата:** 2024-12-20  
**Статус:** ✅ Backend Complete, iOS Partial (Core functionality ready)

## Обзор

Реализован полный end-to-end функционал планирования тренировок (Workout Plan MVP):
- ✅ Backend API (Go)
- ✅ Database migrations (PostgreSQL)
- ✅ Storage layer (Memory + Postgres)
- ✅ Business logic с валидацией
- ✅ Notifications (workout reminders)
- ✅ AI Proposals (kind=workout_plan)
- ✅ Tests (все проходят)
- ⚠️ iOS UI (частично - базовая функциональность)

## Backend Implementation

### 1. Database Migrations

**Файл:** `server/migrations/00013_workout_plans.sql`

Три таблицы:
- `workout_plans` - планы тренировок (один активный на профиль)
- `workout_plan_items` - элементы плана (упражнения)
- `workout_completions` - отметки выполнения

Ключевые ограничения:
- UNIQUE constraint на активный план (owner_user_id, profile_id)
- UNIQUE constraint на items (owner_user_id, profile_id, kind, time_minutes, days_mask)
- UNIQUE constraint на completions (owner_user_id, profile_id, date, plan_item_id)
- CHECK constraints на валидацию полей

### 2. Storage Layer

**Memory Storage:**
- `server/internal/storage/memory/workout_plans.go`
- `server/internal/storage/memory/workout_items.go`
- `server/internal/storage/memory/workout_completions.go`

**Postgres Storage:**
- `server/internal/storage/postgres/workout_plans.go`
- `server/internal/storage/postgres/workout_items.go`
- `server/internal/storage/postgres/workout_completions.go`

Все методы соблюдают ownership: фильтрация по `owner_user_id`, чужие ID возвращают 404.

### 3. Business Logic

**Пакет:** `server/internal/workouts/`

**Файлы:**
- `models.go` - DTOs, валидация, requests/responses
- `service.go` - бизнес-логика
- `handlers.go` - HTTP handlers
- `handlers_test.go` - тесты (6 test cases, все проходят)

**Валидация:**
- Максимум 30 items в плане
- Максимум 4 items на один день (по days_mask)
- time_minutes: 0-1439 (минуты дня)
- days_mask: 0-127 (битовая маска Mon-Sun)
- duration_min: 5-240 минут
- kind: run/walk/strength/morning/core/other
- intensity: low/medium/high
- details JSON: максимум 16KB

### 4. API Endpoints

```
GET  /v1/workouts/plan?profile_id=<uuid>
PUT  /v1/workouts/plan/replace
POST /v1/workouts/completions
GET  /v1/workouts/today?profile_id=<uuid>&date=YYYY-MM-DD
GET  /v1/workouts/completions?profile_id=<uuid>&from=YYYY-MM-DD&to=YYYY-MM-DD
```

### 5. Notifications

**Файл:** `server/internal/notifications/service.go`

Добавлен `maybeBuildWorkoutReminder()`:
- Генерирует напоминание "Тренировка сегодня"
- Только если есть planned items на сегодня
- Только если НЕТ completions (done/skipped)
- Уважает quiet hours и max_per_day
- Генерирует в окне: 30 минут до планового времени

**Тесты:** `server/internal/notifications/service_workouts_test.go` (3 test cases)

### 6. AI Proposals

**Поддержка kind=workout_plan:**

**Файлы:**
- `server/internal/proposals/service.go` - Apply метод
- `server/internal/proposals/models.go` - WorkoutPlanPayload
- `server/internal/ai/mock_provider.go` - генерация proposals

**Триггеры в chat:** тренир*/план*/бег*/силов*/выносл*

**Пример payload:**
```json
{
  "replace": true,
  "title": "План выносливости",
  "goal": "выносливость",
  "items": [
    {
      "kind": "run",
      "time_minutes": 420,
      "days_mask": 62,
      "duration_min": 30,
      "intensity": "medium",
      "note": "лёгкий темп",
      "details": {}
    }
  ]
}
```

## iOS Implementation

### 1. Models

**Файл:** `ios/HealthHub/HealthHub/Models/WorkoutsDTO.swift`

DTOs:
- `WorkoutPlanDTO`
- `WorkoutItemDTO` (с локализацией kind/intensity)
- `WorkoutCompletionDTO`
- `WorkoutSessionDTO`
- Requests/Responses

### 2. API Client

**Файл:** `ios/HealthHub/HealthHub/Core/Networking/APIClient.swift`

Методы:
```swift
fetchWorkoutPlan(profileId: UUID) async throws -> GetWorkoutPlanResponse
replaceWorkoutPlan(profileId: UUID, title: String, goal: String, items: [WorkoutItemUpsert]) async throws -> ReplaceWorkoutPlanResponse
fetchWorkoutToday(profileId: UUID, date: String) async throws -> WorkoutTodayResponse
upsertWorkoutCompletion(profileId: UUID, date: String, planItemId: UUID, status: String, note: String) async throws -> WorkoutCompletionDTO
listWorkoutCompletions(profileId: UUID, from: String, to: String) async throws -> [WorkoutCompletionDTO]
```

### 3. UI (Частично)

**Файл:** `ios/HealthHub/HealthHub/Features/Activity/WorkoutsPlanView.swift`

Реализовано:
- ✅ Просмотр плана тренировок
- ✅ Список planned items на сегодня
- ✅ Кнопки "Выполнено" / "Пропустить"
- ✅ Редактирование плана (sheet)
- ✅ Добавление/удаление items

**Не реализовано (требуется):**
- ⚠️ Интеграция в HomeView (карточка "Тренировка сегодня")
- ⚠️ Ссылка в ActivityView
- ⚠️ Apply/Reject для workout_plan proposals в ChatView
- ⚠️ Расширенное редактирование days_mask (picker дней недели)
- ⚠️ Редактирование details JSON

## Testing

### Backend Tests

```bash
cd server
go test ./... -timeout 30s
```

**Результат:** ✅ PASS (все пакеты)

**Workout Tests:**
- `TestWorkoutsReplacePlanAndGet` - создание и получение плана
- `TestWorkoutsTodayWithPlannedItems` - фильтрация по дню недели
- `TestWorkoutsCompletionAndTodayStatus` - completions и isDone
- `TestWorkoutsOwnershipIsolation` - изоляция между пользователями
- `TestWorkoutsValidationTooManyItemsPerDay` - валидация лимитов
- `TestWorkoutsValidationReplaceNotTrue` - валидация replace=true

**Notification Tests:**
- `TestWorkoutReminderGenerated` - генерация reminder
- `TestWorkoutReminderNotGeneratedWhenCompleted` - не генерируется если done
- `TestWorkoutReminderNotGeneratedTooEarly` - окно времени

## API Examples

### 1. Создать план тренировок

```bash
curl -X PUT http://localhost:8080/v1/workouts/plan/replace \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_id": "550e8400-e29b-41d4-a716-446655440000",
    "title": "План выносливости",
    "goal": "выносливость",
    "replace": true,
    "items": [
      {
        "kind": "run",
        "time_minutes": 420,
        "days_mask": 62,
        "duration_min": 30,
        "intensity": "medium",
        "note": "лёгкий темп",
        "details": {}
      },
      {
        "kind": "strength",
        "time_minutes": 1140,
        "days_mask": 20,
        "duration_min": 40,
        "intensity": "high",
        "note": "верх тела",
        "details": {}
      }
    ]
  }'
```

### 2. Получить план

```bash
curl http://localhost:8080/v1/workouts/plan?profile_id=550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer $TOKEN"
```

### 3. Сегодняшняя тренировка

```bash
curl "http://localhost:8080/v1/workouts/today?profile_id=550e8400-e29b-41d4-a716-446655440000&date=2024-12-20" \
  -H "Authorization: Bearer $TOKEN"
```

### 4. Отметить выполнено

```bash
curl -X POST http://localhost:8080/v1/workouts/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_id": "550e8400-e29b-41d4-a716-446655440000",
    "date": "2024-12-20",
    "plan_item_id": "660e8400-e29b-41d4-a716-446655440000",
    "status": "done",
    "note": "отлично!"
  }'
```

### 5. Chat → Proposal → Apply

```bash
# 1. Отправить сообщение в chat
curl -X POST http://localhost:8080/v1/chat/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_id": "550e8400-e29b-41d4-a716-446655440000",
    "content": "Составь мне план тренировок на неделю для выносливости"
  }'

# 2. Получить proposals
curl "http://localhost:8080/v1/ai/proposals?profile_id=550e8400-e29b-41d4-a716-446655440000&status=pending" \
  -H "Authorization: Bearer $TOKEN"

# 3. Apply proposal
curl -X POST "http://localhost:8080/v1/ai/proposals/{proposal_id}/apply" \
  -H "Authorization: Bearer $TOKEN"
```

### 6. Генерация notifications

```bash
curl -X POST http://localhost:8080/v1/inbox/generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "profile_id": "550e8400-e29b-41d4-a716-446655440000",
    "date": "2024-12-20",
    "time_zone": "Europe/Moscow",
    "now": "2024-12-20T06:30:00Z"
  }'
```

## Migration Commands

```bash
cd server

# Run migrations
go run cmd/migrate/main.go up

# Check migration status
psql $DATABASE_URL -c "SELECT version FROM goose_db_version ORDER BY id DESC LIMIT 1;"
```

## Known Limitations

1. **feedService integration:** Actual workouts из HealthKit не интегрированы в `today` response (поле `actual_workouts` всегда пустое). Для MVP достаточно planned + completions.

2. **iOS UI неполная:**
   - Нет карточки на HomeView
   - Нет интеграции в ActivityView
   - Нет apply/reject в ChatView
   - Нет расширенного редактора days_mask

3. **Details JSON:** В iOS нет UI для редактирования сложных структур в `details` (интервалы, упражнения). Сейчас можно передать только пустой объект.

4. **Локализация:** Все тексты хардкодены на русском языке.

## Files Changed/Created

### Backend (Go)
```
server/migrations/00013_workout_plans.sql                    [CREATED]
server/internal/storage/storage.go                           [MODIFIED]
server/internal/storage/memory/workout_plans.go              [CREATED]
server/internal/storage/memory/workout_items.go              [CREATED]
server/internal/storage/memory/workout_completions.go        [CREATED]
server/internal/storage/memory/memory.go                     [MODIFIED]
server/internal/storage/postgres/workout_plans.go            [CREATED]
server/internal/storage/postgres/workout_items.go            [CREATED]
server/internal/storage/postgres/workout_completions.go      [CREATED]
server/internal/storage/postgres/postgres.go                 [MODIFIED]
server/internal/workouts/models.go                           [CREATED]
server/internal/workouts/service.go                          [CREATED]
server/internal/workouts/handlers.go                         [CREATED]
server/internal/workouts/handlers_test.go                    [CREATED]
server/internal/notifications/service.go                     [MODIFIED]
server/internal/notifications/service_workouts_test.go       [CREATED]
server/internal/proposals/service.go                         [MODIFIED]
server/internal/proposals/models.go                          [MODIFIED]
server/internal/ai/mock_provider.go                          [MODIFIED]
server/internal/httpserver/server.go                         [MODIFIED]
```

### iOS (Swift)
```
ios/HealthHub/HealthHub/Models/WorkoutsDTO.swift             [CREATED]
ios/HealthHub/HealthHub/Core/Networking/APIClient.swift      [MODIFIED]
ios/HealthHub/HealthHub/Features/Activity/WorkoutsPlanView.swift [CREATED]
```

## Next Steps (TODO)

1. **iOS Integration:**
   - [ ] Добавить WorkoutTodayCard на HomeView
   - [ ] Добавить NavigationLink в ActivityView
   - [ ] Обработка workout_plan proposals в ChatView
   - [ ] Улучшенный редактор days_mask (picker Mon-Sun)

2. **Enhancements:**
   - [ ] Интеграция actual workouts из HealthKit в `today` response
   - [ ] Push notifications для workout reminders
   - [ ] Статистика выполнения (completion rate)
   - [ ] History view (календарь с отметками)

3. **OpenAPI Documentation:**
   - [ ] Обновить contracts/openapi.yaml с workout endpoints
   - [ ] Добавить примеры в README.md

## Verification Checklist

### Backend
- [x] Migrations apply успешно
- [x] go test ./... PASS
- [x] curl: create plan → 200 OK
- [x] curl: get plan → returns items
- [x] curl: today → planned items фильтруются по дню
- [x] curl: completion → today reflects completion
- [x] curl: notification generate → workout reminder
- [x] curl: chat → proposal → apply → plan created
- [x] Ownership isolation (userA не видит userB)
- [x] Validation errors (400 для invalid requests)

### iOS (Partial)
- [x] Models компилируются
- [x] API methods добавлены
- [x] WorkoutsPlanView базово работает
- [ ] Home card интеграция
- [ ] Activity navigation
- [ ] Chat proposals apply/reject

## Conclusion

**Backend:** ✅ Полностью готов к production  
**iOS:** ⚠️ Требуется завершение UI интеграции

Все основные требования MVP выполнены:
- Сущности (plans/items/completions) ✅
- API endpoints ✅
- Ownership & validation ✅
- Notifications ✅
- AI Proposals ✅
- Tests ✅

**Git commit НЕ сделан** (как требовалось).

---

**Author:** Health App Team  
**Date:** 2024-12-20
