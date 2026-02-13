# Deployment Guide — Render + Neon + Yandex S3

> Пошаговое руководство по деплою Health Hub backend на [Render](https://render.com) с базой данных [Neon PostgreSQL](https://neon.tech) и файловым хранилищем [Yandex Object Storage (S3)](https://cloud.yandex.ru/services/storage).

---

## Содержание

1. [Обзор архитектуры](#обзор-архитектуры)
2. [Neon PostgreSQL](#neon-postgresql)
3. [Yandex Object Storage (S3)](#yandex-object-storage-s3)
4. [SMTP для Email OTP](#smtp-для-email-otp)
5. [Деплой на Render](#деплой-на-render)
6. [Проверка после деплоя](#проверка-после-деплоя)
7. [Troubleshooting](#troubleshooting)

---

## Обзор архитектуры

```
┌─────────────┐         ┌──────────────────┐
│  iOS App    │──HTTPS──▶│  Render (Docker) │
│  (SwiftUI)  │         │  health-hub-api   │
└─────────────┘         └────┬────┬────┬────┘
                             │    │    │
                   ┌─────────┘    │    └──────────┐
                   ▼              ▼                ▼
           ┌─────────────┐ ┌──────────┐  ┌────────────────┐
           │ Neon PG     │ │ SMTP     │  │ Yandex S3      │
           │ (pooled +   │ │ (Email   │  │ (sources,      │
           │  direct)    │ │  OTP)    │  │  reports)      │
           └─────────────┘ └──────────┘  └────────────────┘
```

**Ключевые переменные окружения:**

| Переменная | Назначение |
|---|---|
| `DATABASE_URL_POOLED` | Runtime-запросы (через pgbouncer) |
| `DATABASE_URL_DIRECT` | DDL / миграции (прямое подключение) |
| `BLOB_MODE` / `REPORTS_MODE` | `local` / `auto` / `s3` |
| `EMAIL_SENDER_MODE` | `local` (OTP в консоль) / `smtp` (реальная отправка) |
| `AUTH_MODE` | `none` / `dev` / `siwa` |
| `AUTH_REQUIRED` | `0` / `1` |

---

## Neon PostgreSQL

### Создание базы

1. Зайди на [console.neon.tech](https://console.neon.tech).
2. **Create Project** → выбери Region (например `eu-central-1`).
3. Дождись создания проекта — Neon автоматически создаст ветку `main` и базу `neondb`.

### Pooled vs Direct — зачем два URL?

Neon предоставляет **два** connection string:

| Тип | Где найти | Порт | Для чего |
|---|---|---|---|
| **Pooled** | Dashboard → Connection Details → Pooled | обычно `5432` (через pgbouncer) | Runtime-запросы (`SELECT`, `INSERT`, `UPDATE`) |
| **Direct** | Dashboard → Connection Details → Direct | обычно `5432` (прямой endpoint) | DDL-миграции (`CREATE TABLE`, `ALTER TABLE`, `DROP`) |

#### Почему pooled нельзя для миграций?

Neon (как и Supabase) использует **PgBouncer** в режиме `transaction`. В этом режиме:

- ❌ `SET` команды не сохраняются между запросами
- ❌ Advisory locks (`pg_advisory_lock`) не работают — goose полагается на них
- ❌ `CREATE INDEX CONCURRENTLY` невозможен
- ❌ Multi-statement транзакции могут ломаться

**Итого:** `DATABASE_URL_DIRECT` обязателен для `cmd/migrate` и для `RUN_MIGRATIONS_ON_STARTUP=1`.

#### Как найти Direct URL в Neon UI

1. Dashboard → твой проект → **Connection Details**
2. Убедись, что в дропдауне выбран **Direct** (не Pooled)
3. Скопируй connection string — он выглядит как:
   ```
   postgresql://user:password@ep-xxx-yyy-123456.eu-central-1.aws.neon.tech/neondb?sslmode=require
   ```
4. Для Pooled — переключи на **Pooled** и скопируй второй URL.

> **Совет:** В Neon, endpoint для pooled и direct может быть один и тот же хост, но с разным параметром `-pooler` в имени. Проверь, что ты используешь правильный.

### Переменные для Render

```
DATABASE_URL_POOLED=postgresql://user:pass@ep-xxx-pooler.region.aws.neon.tech/neondb?sslmode=require
DATABASE_URL_DIRECT=postgresql://user:pass@ep-xxx.region.aws.neon.tech/neondb?sslmode=require
```

---

## Yandex Object Storage (S3)

### Создание бакета

1. [console.cloud.yandex.ru](https://console.cloud.yandex.ru) → Object Storage → **Create bucket**.
2. Имя бакета: например `health-hub-prod`.
3. **Доступ:**
   - Если хочешь public URLs без presign: сделай **Public read** для объектов.
   - Если хочешь presigned-only: оставь **Private**.
4. Регион: `ru-central1`.

### Создание сервисного аккаунта + ключей

1. IAM → Service accounts → Create.
2. Роль: `storage.editor` (или `storage.uploader` + `storage.viewer`).
3. Создай **Static access key** (Access Key ID + Secret Access Key).

### URL-схема

Yandex S3 использует **path-style** URLs:

```
https://storage.yandexcloud.net/<bucket>/<key>
```

Пример:
```
S3_ENDPOINT=https://storage.yandexcloud.net
S3_REGION=ru-central1
S3_BUCKET=health-hub-prod
S3_PUBLIC_BASE_URL=https://storage.yandexcloud.net/health-hub-prod
```

### Public vs Presigned URLs

| Параметр | Когда |
|---|---|
| `S3_PREFER_PUBLIC_URL=1` | Бакет **public read** → download возвращает прямую ссылку |
| `S3_PREFER_PUBLIC_URL=0` (default) | Download возвращает presigned URL (работает с private бакетами) |

`S3_PRESIGN_TTL_SECONDS` — время жизни presigned URL (по умолчанию 900 = 15 минут).

### Переменные для Render

```
BLOB_MODE=s3
REPORTS_MODE=s3
S3_ENDPOINT=https://storage.yandexcloud.net
S3_REGION=ru-central1
S3_BUCKET=health-hub-prod
S3_ACCESS_KEY_ID=YCA...
S3_SECRET_ACCESS_KEY=YCs...
S3_PUBLIC_BASE_URL=https://storage.yandexcloud.net/health-hub-prod
S3_PRESIGN_TTL_SECONDS=900
S3_PREFER_PUBLIC_URL=0
```

---

## SMTP для Email OTP

В production Email OTP коды отправляются через настоящий SMTP-сервер.

### Провайдеры

Подойдёт любой SMTP-провайдер:
- **Yandex 360** (smtp.yandex.ru:587)
- **Mailgun**, **SendGrid**, **Amazon SES**
- Любой SMTP с поддержкой STARTTLS

### Переменные

```
EMAIL_SENDER_MODE=smtp
SMTP_HOST=smtp.yandex.ru
SMTP_PORT=587
SMTP_USERNAME=your-email@yourdomain.com
SMTP_PASSWORD=app-specific-password
SMTP_FROM=HealthHub <no-reply@yourdomain.com>
SMTP_USE_TLS=1
```

### Локальная разработка

В `local` mode (`EMAIL_SENDER_MODE=local`) OTP-код просто печатается в консоль сервера:

```
mailer.local: to=user@example.com subject="Your verification code" body="Your code: 123456"
```

Это задокументировано и безопасно для dev — никакой реальной отправки не происходит.

---

## Деплой на Render

### Вариант A: Blueprint (render.yaml)

1. Пушнешь код в GitHub/GitLab.
2. В Render Dashboard → **New** → **Blueprint**.
3. Выбери репозиторий → Render найдёт `render.yaml` автоматически.
4. Заполни все `sync: false` переменные (секреты):
   - `DATABASE_URL_POOLED`
   - `DATABASE_URL_DIRECT`
   - `S3_BUCKET`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `S3_PUBLIC_BASE_URL`
   - `SMTP_HOST`, `SMTP_USERNAME`, `SMTP_PASSWORD`
5. Deploy!

### Вариант B: Manual

1. **New** → **Web Service** → Docker.
2. Настройки:
   - **Root Directory:** `server`
   - **Dockerfile Path:** `./Dockerfile`
   - **Docker Context:** `.`
3. **Environment:**

   | Variable | Value |
   |---|---|
   | `APP_ENV` | `production` |
   | `PORT` | `8080` |
   | `AUTH_MODE` | `dev` |
   | `AUTH_REQUIRED` | `1` |
   | `EMAIL_AUTH_ENABLED` | `1` |
   | `JWT_SECRET` | *(generate random 32+ char string)* |
   | `OTP_SECRET` | *(generate random 32+ char string)* |
   | `RUN_MIGRATIONS_ON_STARTUP` | `1` |
   | `DATABASE_URL_POOLED` | *(from Neon)* |
   | `DATABASE_URL_DIRECT` | *(from Neon)* |
   | `BLOB_MODE` | `s3` |
   | `REPORTS_MODE` | `s3` |
   | `EMAIL_SENDER_MODE` | `smtp` |
   | + все `S3_*` и `SMTP_*` | *(see above)* |

4. **Health Check Path:** `/healthz`
5. **Start Command:** оставь пустым (Dockerfile `ENTRYPOINT` уже задан).

### Почему AUTH_MODE=dev, а не siwa?

`AUTH_MODE=dev` включает JWT-валидацию и позволяет использовать:
- `/v1/auth/email/request` + `/v1/auth/email/verify` — Email OTP (основной flow)
- `/v1/auth/dev` — dev-токен (удобно для тестирования, отключи на проде через `AUTH_REQUIRED`)

SIWA (Sign in with Apple) можно добавить позже, переключив на `AUTH_MODE=siwa` и настроив `APPLE_BUNDLE_ID`.

### Миграции на startup

При `RUN_MIGRATIONS_ON_STARTUP=1` сервер выполняет `goose up` перед стартом HTTP.

**Безопасно ли это на Render?**

- ✅ Render запускает **один инстанс** (plan=starter/free) — race condition нет.
- ✅ Goose использует advisory locks — даже при нескольких инстансах миграции атомарны.
- ⚠️ Если миграция зависает — health check не пройдёт, Render перезапустит инстанс.

**Альтернатива (ручные миграции):**

Если не хочешь миграции на startup:

1. Убери `RUN_MIGRATIONS_ON_STARTUP` (или `=0`).
2. Перед деплоем локально выполни:
   ```sh
   DATABASE_URL_DIRECT=postgresql://... go run ./cmd/migrate up
   ```

---

## Проверка после деплоя

### 1. Health check

```sh
curl https://your-app.onrender.com/healthz
# {"status":"ok"}
```

### 2. Smoke tests

```sh
# Получи JWT токен через Email OTP:
# 1) POST /v1/auth/email/request  {"email":"your@email.com"}
# 2) Найди код в почте
# 3) POST /v1/auth/email/verify   {"email":"your@email.com","code":"123456"}
# 4) Запомни token из ответа

export API_BASE_URL=https://your-app.onrender.com
export SMOKE_TOKEN=eyJhbGciOiJI...

cd server
go run ./cmd/smoke
```

Или через Makefile:

```sh
API_BASE_URL=https://your-app.onrender.com SMOKE_TOKEN=eyJ... make smoke
```

### 3. Что проверяется smoke-тестами

- `/healthz` — сервер жив
- Profiles — создание/получение
- Sync batch — запись метрик
- Checkins — создание чекина
- Feed — получение ленты
- Reports — создание CSV, скачивание, удаление
- Sources — загрузка изображения, скачивание, удаление

### 4. Стартовые логи

При правильной конфигурации в логах Render ты увидишь:

```
========== Health Hub API ==========
  env              = production
  port             = 8080
---- database ----
  runtime_url      = set (via DATABASE_URL_POOLED)
  pooled           = set
  direct           = set
  migrations_on_startup = true
  migrations_via   = DATABASE_URL_DIRECT
---- auth ----
  auth_mode        = dev
  auth_required    = true
  email_auth       = true
  jwt_secret       = set (custom)
  otp_secret       = set
---- blob ----
  blob_mode        = s3
  reports_mode     = s3 (effective=s3)
  s3: endpoint=https://storage.yandexcloud.net region=ru-central1 bucket=health-hub-prod ...
---- mailer ----
  email_sender     = smtp
  smtp_host        = smtp.yandex.ru
  smtp_port        = 587
  smtp_from        = HealthHub <no-reply@yourdomain.com>
  smtp_username    = set
  smtp_password    = set
  smtp_use_tls     = true
---- ai ----
  ai_mode          = mock
====================================
```

Если что-то сконфигурировано неправильно — сервер упадёт с `FATAL` и подскажет, чего именно не хватает.

---

## Troubleshooting

### Сервер падает с FATAL при старте

Проверь логи. Типичные причины:

| Ошибка | Причина | Решение |
|---|---|---|
| `FATAL blob: ... missing: S3_ENDPOINT, S3_BUCKET` | `BLOB_MODE=s3`, но S3 не настроен | Добавь все `S3_*` переменные |
| `FATAL mailer: ... missing: SMTP_HOST` | `EMAIL_SENDER_MODE=smtp`, но SMTP не настроен | Добавь `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM` |
| `FATAL auth: JWT_SECRET must not be 'change_me'` | Забыл задать `JWT_SECRET` | Сгенерируй случайную строку |
| `FATAL startup migrations: DATABASE_URL_DIRECT is required` | Миграции включены, но нет direct URL | Задай `DATABASE_URL_DIRECT` |
| `FATAL db: no DATABASE_URL configured` | Production без базы данных | Задай `DATABASE_URL_POOLED` или `DATABASE_URL` |

### Миграции зависают

- Проверь, что `DATABASE_URL_DIRECT` указывает на **direct** endpoint Neon (не pooled).
- Попробуй локально: `DATABASE_URL_DIRECT=... go run ./cmd/migrate status`

### S3 upload возвращает 500

- Проверь `S3_ACCESS_KEY_ID` и `S3_SECRET_ACCESS_KEY`.
- Проверь, что бакет существует и сервисный аккаунт имеет права `storage.editor`.
- Проверь `S3_ENDPOINT` — для Yandex это `https://storage.yandexcloud.net`.

### Email OTP не приходит

- Проверь логи — ошибка SMTP будет в логе.
- Проверь `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`.
- Убедись, что `SMTP_USE_TLS=1` для порта 587.
- Попробуй `EMAIL_SENDER_MODE=local` для диагностики — код появится в логах.

### iOS не подключается к Render API

- В `ios/HealthHub/AppInfo.plist` добавь ключ `API_BASE_URL` со значением `https://your-app.onrender.com`.
- Или измени значение по умолчанию в `AppConfig.swift`.
- Убедись, что ATS (App Transport Security) не блокирует HTTPS-запросы (для `.onrender.com` — не блокирует, это валидный HTTPS).
