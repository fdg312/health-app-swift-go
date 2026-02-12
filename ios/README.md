# iOS App — HealthHub

## Создание проекта вручную

Поскольку Xcode проекты (.xcodeproj) создаются через GUI, выполни следующие шаги:

### 1. Создай новый проект в Xcode

1. Открой Xcode
2. File → New → Project
3. Выбери **iOS** → **App**
4. Настройки:
   - **Product Name**: `HealthHub`
   - **Team**: (твой Apple Developer аккаунт или оставь None для локальной разработки)
   - **Organization Identifier**: `com.example` (или свой)
   - **Bundle Identifier**: `com.example.HealthHub`
   - **Interface**: SwiftUI
   - **Language**: Swift
   - **Minimum Deployments**: iOS 18.0
5. Сохрани проект в папку `ios/HealthHub/`

### 2. Замени исходники

После создания проекта замени файлы:

```bash
# Удали дефолтный ContentView.swift и замени его нашим
# В Finder: ios/HealthHub/HealthHub/
```

В Xcode:
- **Замени** `HealthHubApp.swift` содержимым из `ios/HealthHub/HealthHub/HealthHubApp.swift`
- **Замени** `ContentView.swift` содержимым из `ios/HealthHub/HealthHub/ContentView.swift`
- **Добавь папку** `Features` в проект:
  - Right-click на `HealthHub` в Project Navigator → Add Files to "HealthHub"
  - Выбери папку `Features` и убедись, что выбрана опция **"Create groups"**

### 3. Проверь структуру проекта

Должна получиться такая структура в Xcode:

```
HealthHub
├── HealthHubApp.swift
├── ContentView.swift
├── Features
│   ├── Feed
│   │   └── FeedView.swift
│   ├── Metrics
│   │   └── MetricsView.swift
│   ├── Activity
│   │   └── ActivityView.swift
│   └── Chat
│       └── ChatView.swift
└── Assets.xcassets
```

### 4. Настройка минимальной версии iOS

1. В Xcode выбери проект в Project Navigator
2. Targets → HealthHub → General
3. **Minimum Deployments**: установи iOS 18.0

### 5. Запуск

1. Выбери симулятор: **iPhone 16** (или новее)
2. Нажми **Run** (Cmd+R)
3. Приложение откроется с 4 вкладками

## Структура приложения

- **ContentView.swift**: главный экран с TabView
- **Features/Feed**: лента активности (заглушка)
- **Features/Metrics**: показатели здоровья (заглушка)
- **Features/Activity**: активность (заглушка)
- **Features/Chat**: чат с AI (заглушка)

Все экраны — минимальные заглушки для проверки структуры.

## Следующие шаги

После успешного запуска можно добавлять:
- Модели данных
- Сетевой слой для API
- HealthKit интеграцию
- Реальные UI компоненты
