# Health Hub — Documentation Index

> Карта проекта и ссылки на ключевые документы.

## Быстрый старт

| Документ | Описание |
|----------|----------|
| [README.md](../README.md) | Главный README: запуск backend/iOS, API примеры, конфигурация |
| [QUICK_SETUP.md](./QUICK_SETUP.md) | **🚀 Быстрый старт за 5 минут** — минимальная конфигурация для разработки |
| [ENV_TROUBLESHOOTING.md](./ENV_TROUBLESHOOTING.md) | **🔧 Решение проблем** с переменными окружения и настройкой |
| [DEPLOYMENT.md](./DEPLOYMENT.md) | Деплой на Render + Neon + Yandex S3 (пошагово) |
| [architecture.md](./architecture.md) | Архитектура проекта (компоненты, принципы) |

## Backend (Go)

### Конфигурация и инфраструктура

- **Environment Setup** — [`.env.example`](../server/.env.example) содержит все доступные переменные окружения с описаниями и примерами. См. также [QUICK_SETUP.md](./QUICK_SETUP.md) для быстрого старта.
- **Environment Variables** — автоматическая загрузка через `godotenv/autoload`. При проблемах см. [ENV_TROUBLESHOOTING.md](./ENV_TROUBLESHOOTING.md).
- **Database Migrations** — via [goose](https://github.com/pressly/goose), файлы в `server/migrations/`. Команды: `make migrate-up`, `go run ./cmd/migrate up|status|down`. См. [README § Database migrations](../README.md).
- **S3 (Yandex Object Storage)** — режимы `BLOB_MODE` и `REPORTS_MODE` (`local`/`auto`/`s3`). См. [README § S3 режим](../README.md).
- **E2E Smoke Tests** — `go run ./cmd/smoke` или `make smoke`. См. [README § E2E Smoke Tests](../README.md).

### Аутентификация

- **Auth Overview** — `AUTH_MODE` (`none`/`dev`/`siwa`), `AUTH_REQUIRED`, `EMAIL_AUTH_ENABLED`. См. [README § Authentication](../README.md).
- **Email OTP** — `EMAIL_SENDER_MODE=local` (код в консоли) / `smtp` (настоящая отправка). См. [README § Email OTP auth](../README.md).
- **SIWA** — Sign in with Apple (опционально, можно включить позже). См. [reports/SIWA_AUTH_REPORT.md](./reports/SIWA_AUTH_REPORT.md).

### API Contracts

- [contracts/openapi.yaml](../contracts/openapi.yaml) — основная OpenAPI спецификация
- [contracts/openapi-feed-checkins.yaml](../contracts/openapi-feed-checkins.yaml) — feed & checkins фрагмент
- [workout-endpoints-openapi-fragment.yaml](./workout-endpoints-openapi-fragment.yaml) — workout endpoints фрагмент

## iOS (SwiftUI)

- [ios/README.md](../ios/README.md) — сборка и запуск iOS-приложения
- [ios/BACKGROUND_SYNC.md](../ios/BACKGROUND_SYNC.md) — фоновая синхронизация HealthKit + Inbox
- [ios/HEALTHKIT_SETUP.md](../ios/HEALTHKIT_SETUP.md) — настройка HealthKit entitlements
- **AppConfig** — `ios/HealthHub/HealthHub/Core/Config/AppConfig.swift`: API base URL читается из `Info.plist` (`API_BASE_URL`), fallback на `http://localhost:8080`

## Фичи — Implementation Reports

Все отчёты о реализации лежат в [`docs/reports/`](./reports/):

| Отчёт | Фича |
|-------|-------|
| [FINAL_REPORT.md](./reports/FINAL_REPORT.md) | Итоговый отчёт проекта |
| [INTEGRATION_SUMMARY.md](./reports/INTEGRATION_SUMMARY.md) | Итоги интеграции всех компонентов |
| [S3_POLISH_E2E_REPORT.md](./reports/S3_POLISH_E2E_REPORT.md) | S3 mode, diagnostics, E2E polish |
| [REPORTS_IMPLEMENTATION_REPORT.md](./reports/REPORTS_IMPLEMENTATION_REPORT.md) | Export Reports (CSV/PDF) |
| [INBOX_IMPLEMENTATION_REPORT.md](./reports/INBOX_IMPLEMENTATION_REPORT.md) | Inbox / Notifications |
| [SMART_REMINDERS_IMPLEMENTATION_REPORT.md](./reports/SMART_REMINDERS_IMPLEMENTATION_REPORT.md) | Smart Local Reminders |
| [INTAKES_IMPLEMENTATION_REPORT.md](./reports/INTAKES_IMPLEMENTATION_REPORT.md) | Water & Supplements Intakes |
| [MEAL_PLAN_MVP_REPORT.md](./reports/MEAL_PLAN_MVP_REPORT.md) | Meal Plan MVP (backend) |
| [FINAL_MEAL_PLAN_IOS_REPORT.md](./reports/FINAL_MEAL_PLAN_IOS_REPORT.md) | Meal Plan iOS integration |
| [NUTRITION_TARGETS_MVP_REPORT.md](./reports/NUTRITION_TARGETS_MVP_REPORT.md) | Nutrition Targets |
| [WORKOUT_PLAN_MVP_REPORT.md](./reports/WORKOUT_PLAN_MVP_REPORT.md) | Workout Plans MVP (backend) |
| [WORKOUT_PLANS_INTEGRATION_REPORT.md](./reports/WORKOUT_PLANS_INTEGRATION_REPORT.md) | Workout Plans iOS integration |
| [SIWA_AUTH_REPORT.md](./reports/SIWA_AUTH_REPORT.md) | Sign in with Apple |
| [iOS_CHARTS_SHARE_CONFIG_REPORT.md](./reports/iOS_CHARTS_SHARE_CONFIG_REPORT.md) | iOS Charts & Share Config |
| [iOS_CHECKIN_EDITOR_REPORT.md](./reports/iOS_CHECKIN_EDITOR_REPORT.md) | iOS Checkin Editor |
| [REVIEW_CHECKLIST.md](./reports/REVIEW_CHECKLIST.md) | Code review checklist |

## SQL Reference

- [docs/sql/](./sql/) — справочные SQL-запросы

## Структура каталогов

```
health-app-swift-go/
├── README.md
├── Makefile
├── render.yaml
├── contracts/          # OpenAPI specs
├── docs/
│   ├── INDEX.md        # ← вы здесь
│   ├── DEPLOYMENT.md   # Render + Neon + S3 guide
│   ├── architecture.md
│   ├── sql/
│   └── reports/        # implementation reports
├── server/
│   ├── Dockerfile
│   ├── cmd/
│   │   ├── api/        # main HTTP server
│   │   ├── migrate/    # DB migration tool
│   │   └── smoke/      # E2E smoke tests
│   ├── internal/       # business logic packages
│   └── migrations/     # goose SQL migrations
├── ios/
│   └── HealthHub/      # SwiftUI iOS app
└── scripts/            # helper shell scripts
```
