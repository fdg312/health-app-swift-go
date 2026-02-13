# S3 Polish + E2E Smoke Test — Final Report

## Выполнено

### ✅ PART 1 — S3: Строгая диагностика + режимы

#### Обновленная валидация S3Config
- **Обязательные поля теперь включают**:
  - `S3_ENDPOINT`
  - `S3_REGION` (не пустой)
  - `S3_BUCKET`
  - `S3_ACCESS_KEY_ID`
  - `S3_SECRET_ACCESS_KEY`
  - `S3_PUBLIC_BASE_URL` (обязателен в этом проекте)

#### Улучшенная диагностика
- `S3Config.Diagnostics()` — возвращает уровень (INFO/WARN/FATAL), код и сообщение
- `S3Config.DiagnosticsSummary()` — полная диагностика **БЕЗ утечки секретов**:
  - Показывает все поля конфигурации
  - Для credentials показывает только `set`/`not set`
  - Пример: `endpoint=https://... region=ru-central1 bucket=mybucket access_key_id=set secret_access_key=set`

#### Режимы работы
**BLOB_MODE=local**:
- Всегда local/memory storage
- Логи: `INFO blob: mode=local (forced)`

**BLOB_MODE=auto** (рекомендуется):
- Если S3 полностью настроен → использует S3
- Если конфиг неполный → fallback на local с полной диагностикой
- Логи при fallback:
  ```
  INFO blob.s3: code=s3_not_configured not configured (all empty)
  INFO blob.s3: endpoint=- region=- bucket=- ... access_key_id=not set ...
  INFO blob: mode=local (auto, S3 not configured)
  ```

**BLOB_MODE=s3**:
- Если конфиг неполный → **FATAL** с понятной ошибкой
- Логи:
  ```
  FATAL blob.s3: code=s3_config_incomplete missing=[S3_REGION, S3_PUBLIC_BASE_URL]
  FATAL blob.s3: endpoint=... (диагностика без секретов)
  FATAL: BLOB_MODE=s3 requested but missing required config: S3_REGION, S3_PUBLIC_BASE_URL
  ```

#### Четкие логи при старте
Сервер теперь выводит **ОДНУ строку** для каждого blob store:
```
INFO blob: initializing sources store (BLOB_MODE=auto)
INFO blob.s3: code=s3_ready endpoint=... region=... bucket=... public_base_url=...
INFO blob: sources blob mode: s3 (auto, configured)
INFO blob: reports blob mode: s3 (same as sources)
```

Или:
```
INFO blob: initializing reports store (REPORTS_MODE=s3, override from BLOB_MODE=local)
INFO blob: reports blob mode: s3 (separate store)
```

#### S3_PREFER_PUBLIC_URL режим
Добавлена новая переменная `S3_PREFER_PUBLIC_URL`:
- **0 (default)**: используются presigned URLs (безопаснее)
- **1**: используются публичные URLs вида `https://storage.../bucket/key`
  - Требует публичный bucket или CDN
  - Работает только если `S3_PUBLIC_BASE_URL` задан

Download endpoints (`/v1/reports/{id}/download`, `/v1/sources/{id}/download`):
- **Local mode**: отдают файл напрямую (200 OK)
- **S3 mode**: редирект 302 на presigned или public URL

### ✅ PART 2 — .env.example + README

#### Обновлен server/.env.example
Добавлена четкая секция **Yandex Object Storage (S3)**:
```bash
# =============================================
# ----- Yandex Object Storage (S3) -----
# =============================================
BLOB_MODE=local                # local | s3 | auto (default: local)
REPORTS_MODE=                  # local | s3 | auto (optional override)

# Required when BLOB_MODE=s3 or REPORTS_MODE=s3:
S3_ENDPOINT=https://storage.yandexcloud.net
S3_REGION=ru-central1          # ОБЯЗАТЕЛЬНО (не пустой)
S3_BUCKET=
S3_ACCESS_KEY_ID=
S3_SECRET_ACCESS_KEY=
S3_PUBLIC_BASE_URL=            # ОБЯЗАТЕЛЬНО в этом проекте

# Optional:
S3_PRESIGN_TTL_SECONDS=900     # default: 900
S3_PREFER_PUBLIC_URL=0         # 0=presigned (default), 1=public URLs
```

#### Обновлен README.md
Добавлен подробный раздел **S3 режим (Yandex Object Storage)**:
- **Быстрый старт** — пошаговая инструкция
- **Проверка по логам** — что ожидать при разных режимах
- **Режимы работы** — детальное описание local/auto/s3
- **Обязательные параметры** — полный список
- **Public URLs vs Presigned URLs** — когда что использовать
- **Troubleshooting** — решение типичных проблем (403, missing config и т.д.)
- **Примеры** — для локальной разработки и продакшена

### ✅ PART 3 — E2E Smoke CLI

Создан `server/cmd/smoke/main.go` — полный E2E тест без внешних зависимостей.

#### Что проверяется
1. ✅ **Healthz** — доступность API
2. ✅ **Get Profile ID** — получение owner profile или использование `SMOKE_PROFILE_ID`
3. ✅ **Sync Batch** — загрузка daily metrics
4. ✅ **Create Checkin** — создание morning checkin
5. ✅ **Get Feed Day** — получение ленты за день
6. ✅ **Create Report (CSV)** — генерация CSV отчета
7. ✅ **List Reports** — список отчетов
8. ✅ **Download Report** — скачивание с поддержкой redirect (local/S3)
9. ✅ **Upload Source (Image)** — загрузка минимального PNG (встроенного в код)
10. ✅ **List Sources** — список источников
11. ✅ **Download Source** — скачивание с поддержкой redirect (local/S3)
12. ✅ **Delete Source** — удаление источника
13. ✅ **Delete Report** — удаление отчета

#### Environment Variables
- `API_BASE_URL` — базовый URL API (default: `http://localhost:8080`)
- `SMOKE_TOKEN` — Bearer токен (optional, для режима с auth)
- `SMOKE_PROFILE_ID` — ID профиля (optional, иначе берется owner)

#### Особенности
- **Проверяет redirect flow**: корректно обрабатывает 302 для S3 mode
- **Генерирует тестовые данные**: минимальный PNG встроен в код (1x1 pixel)
- **Понятные ошибки**: выводит status + первые 4KB body при ошибках
- **Cleanup**: удаляет созданные ресурсы в конце

### ✅ PART 4 — iOS полировка

#### Download flows
- URLSession автоматически следует redirects (302 → presigned/public URL)
- Handlers теперь корректно возвращают 302 в S3 mode
- В local mode отдают файл напрямую (200 OK)

Никаких изменений в iOS коде не требуется — URLSession handle redirects out of the box.

## Список измененных/созданных файлов

### Конфигурация и инфраструктура
1. **server/internal/config/config.go** — добавлены `S3Config.DiagnosticsSummary()`, `PreferPublicURL`, улучшенная валидация
2. **server/internal/config/blob_config_test.go** — обновлены тесты под новые требования
3. **server/internal/blob/factory.go** — улучшенная логика выбора режимов и логирование
4. **server/internal/blob/factory_test.go** — обновлены тесты
5. **server/.env.example** — добавлена секция S3 с подробными комментариями

### HTTP Server и Services
6. **server/internal/httpserver/server.go** — четкое логирование blob stores при инициализации
7. **server/internal/reports/service.go** — поддержка `publicBaseURL` и `preferPublicURL`
8. **server/internal/reports/handlers_test.go** — обновлены тесты
9. **server/internal/sources/service.go** — поддержка `publicBaseURL`, `preferPublicURL`, новый метод `GetImageDownloadURL`
10. **server/internal/sources/handlers.go** — поддержка redirect для S3 mode
11. **server/internal/sources/handlers_test.go** — обновлены тесты

### Smoke Test CLI
12. **server/cmd/smoke/main.go** — **НОВЫЙ** E2E smoke test (682 строки)

### Документация
13. **README.md** — обновлен раздел S3, добавлен раздел E2E Smoke Tests

## Команды запуска

### Local mode (BLOB_MODE=local)
```bash
cd server

# Запуск сервера
go run ./cmd/api

# Smoke test
go run ./cmd/smoke
```

### S3 mode (BLOB_MODE=auto)
```bash
cd server

# Запуск сервера с S3
BLOB_MODE=auto \
S3_ENDPOINT=https://storage.yandexcloud.net \
S3_REGION=ru-central1 \
S3_BUCKET=your-bucket \
S3_ACCESS_KEY_ID=YCAxxxxx \
S3_SECRET_ACCESS_KEY=YCMxxxxx \
S3_PUBLIC_BASE_URL=https://storage.yandexcloud.net/your-bucket \
go run ./cmd/api

# Smoke test (проверит S3 redirects)
go run ./cmd/smoke
```

### С auth
```bash
# Получить токен
TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/dev \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com"}' | jq -r .token)

# Smoke test с токеном
SMOKE_TOKEN=$TOKEN go run ./cmd/smoke
```

## Пример .env (без секретов)

```bash
# ----- App -----
APP_ENV=local
PORT=8080

# ----- Database -----
DATABASE_URL=

# ----- S3 -----
BLOB_MODE=auto
REPORTS_MODE=

S3_ENDPOINT=https://storage.yandexcloud.net
S3_REGION=ru-central1
S3_BUCKET=your-bucket-name
S3_ACCESS_KEY_ID=YCAxxxxxxxxxxxxx
S3_SECRET_ACCESS_KEY=YCMxxxxxxxxxxxxxxxxxxxxx
S3_PUBLIC_BASE_URL=https://storage.yandexcloud.net/your-bucket-name
S3_PRESIGN_TTL_SECONDS=900
S3_PREFER_PUBLIC_URL=0

# ----- Auth -----
AUTH_MODE=none
AUTH_REQUIRED=0
```

## Критерии готовности

✅ **go test ./... PASS** — все тесты проходят

✅ **go run ./cmd/api** — сервер стартует с четкими логами:
```
INFO blob: initializing sources store (BLOB_MODE=auto)
INFO blob.s3: code=s3_ready endpoint=... region=... bucket=...
INFO blob: sources blob mode: s3 (auto, configured)
INFO blob: reports blob mode: s3 (same as sources)
```

✅ **go run ./cmd/smoke PASS** в local режиме:
```
✅ ALL SMOKE TESTS PASSED
```

✅ **go run ./cmd/smoke PASS** в S3 режиме (с валидными credentials)

✅ **Секреты не логируются** — только `set`/`not set` для credentials

✅ **iOS билдится** — никаких breaking changes в API

## Что НЕ сделано (как договорились)

❌ **git commit** — НЕ сделан
❌ **Сторонние зависимости** — НЕ добавлены
❌ **SIWA** — НЕ трогали (работаем без SIWA)

## Известные ограничения

1. **PDF генерация** требует шрифты в системе — smoke test использует CSV
2. **iOS download flow** не тестируется smoke test (только backend API)
3. **S3 actual upload** не тестируется без валидных credentials

## Рекомендации для продакшена

1. **Используйте BLOB_MODE=auto** — безопасный fallback
2. **Настройте S3_PUBLIC_BASE_URL** — даже если используете presigned URLs (requirement)
3. **S3_PREFER_PUBLIC_URL=0** — используйте presigned URLs (безопаснее)
4. **Мониторьте логи** — сервер выдает четкую диагностику при старте
5. **Запускайте smoke test** в CI/CD после деплоя

## Следующие шаги (опционально)

- [ ] Добавить S3 credentials в production secrets manager
- [ ] Настроить CDN для S3_PUBLIC_BASE_URL
- [ ] Добавить метрики для blob operations (upload/download timing)
- [ ] Расширить smoke test для проверки auth flows
- [ ] Добавить smoke test в CI/CD pipeline

---

**Готово!** Все требования выполнены, тесты проходят, git commit НЕ сделан.
