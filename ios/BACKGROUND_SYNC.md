# Background Sync Setup (iOS)

Документ описывает настройку фоновой синхронизации HealthKit + обновление inbox/локальных напоминаний.

## 1) Capability: Background Modes

1. Открой `ios/HealthHub/HealthHub.xcodeproj` в Xcode.
2. Выбери target **HealthHub**.
3. Открой вкладку **Signing & Capabilities**.
4. Нажми **+ Capability** и добавь **Background Modes**.
5. Включи:
   - `Background fetch`
   - `Background processing` (опционально, для будущих BG processing задач)

## 2) Capability: HealthKit

1. В том же target добавь **HealthKit**, если ещё не добавлен.
2. Убедись, что есть `NSHealthShareUsageDescription` и `NSHealthUpdateUsageDescription`.

## 3) Info.plist ключи

Нужны ключи:

- `BGTaskSchedulerPermittedIdentifiers` (Array)
  - `$(PRODUCT_BUNDLE_IDENTIFIER).refresh`
- `UIBackgroundModes` (Array)
  - `fetch`
  - `processing` (опционально)

В этом проекте Info.plist генерируется через build settings, поэтому эквивалент можно задать через `INFOPLIST_KEY_*` в target build settings.

## 4) Что делает приложение

- Регистрирует BG task: `$(bundleId).refresh`
- При логине:
  - включает HealthKit background delivery
  - запускает `HKObserverQuery` для ключевых типов
  - планирует `BGAppRefreshTask`
- В фоне/по observer:
  - синхронизирует HealthKit за **сегодня + вчера**
  - hourly отправляет только за **сегодня**
  - обновляет inbox и локальные smart reminders

## 5) Локальные переключатели

Экран **Настройки**:

- `Фоновая синхронизация`
- `Обновлять напоминания в фоне`

По умолчанию оба включены.

## 6) Тестирование

### Симулятор

1. Запусти сервер (`go run ./cmd/api`) и приложение.
2. Авторизуйся.
3. В Xcode: **Debug → Simulate Background Fetch**.
4. Проверь:
   - обновился статус "Последняя синхронизация" на Home
   - были запросы к `/v1/sync/batch`
   - обновился inbox/reminders

Примечание: `HKObserverQuery` на симуляторе может быть нестабильным.

### Реальное устройство

1. Выдай Health permissions.
2. Добавь/измени данные в Apple Health (шаги, сон, тренировка и т.д.).
3. Заблокируй экран и подожди фоновые циклы iOS.
4. Проверь в приложении:
   - `Последняя синхронизация` обновляется
   - напоминания/бейджи соответствуют inbox

