# iOS Production Setup Guide

## ✅ Настройка завершена!

iOS приложение теперь настроено на работу с production сервером:
**https://health-app-swift-go.onrender.com**

---

## 📝 Что было сделано

### 1. Обновлен `AppInfo.plist`

**Файл:** `ios/HealthHub/AppInfo.plist`

Добавлена конфигурация API URL:

```xml
<key>API_BASE_URL</key>
<string>https://health-app-swift-go.onrender.com</string>
```

### 2. Обновлен `AppConfig.swift`

**Файл:** `ios/HealthHub/HealthHub/Core/Config/AppConfig.swift`

- ✅ Добавлено логирование для отладки
- ✅ Изменен fallback URL с `localhost:8080` на production сервер
- ✅ Приоритет: Info.plist → Production URL

```swift
static var apiBaseURL: String {
    if let baseURL = Bundle.main.object(forInfoDictionaryKey: "API_BASE_URL") as? String,
       !baseURL.isEmpty {
        print("📡 API Base URL from Info.plist: \(baseURL)")
        return baseURL
    }
    
    let fallback = "https://health-app-swift-go.onrender.com"
    print("📡 API Base URL (fallback): \(fallback)")
    return fallback
}
```

---

## 🚀 Как запустить

### 1. Откройте проект в Xcode

```bash
cd ios/HealthHub
open HealthHub.xcodeproj
```

### 2. Clean Build (важно!)

```
Xcode → Product → Clean Build Folder (⇧⌘K)
```

### 3. Соберите проект

```
Xcode → Product → Build (⌘B)
```

### 4. Запустите на симуляторе или устройстве

```
Xcode → Product → Run (⌘R)
```

### 5. Проверьте логи

В консоли Xcode при запуске должно появиться:

```
📡 API Base URL from Info.plist: https://health-app-swift-go.onrender.com
```

---

## 🧪 Тестирование

### Проверка подключения

1. **Запустите приложение**
2. **Откройте экран входа/регистрации**
3. **Введите email** (например: `test@example.com`)
4. **Нажмите "Получить код"**

### Ожидаемое поведение:

✅ **Success:** 
- Показывается экран ввода кода
- На email приходит OTP код (если SMTP настроен на сервере)
- В логах: `Making request to: https://health-app-swift-go.onrender.com/v1/auth/email/request`

❌ **Error:**
- Если timeout → проверьте, что сервер работает
- Если 401 → проверьте AUTH настройки на сервере
- Если CORS error → обновите CORS_ALLOWED_ORIGINS на сервере

---

## 🔍 Отладка

### Проверка URL в коде

Добавьте в любой ViewController или Service:

```swift
import Foundation

print("Current API Base URL: \(AppConfig.apiBaseURL)")
```

### Проверка сетевых запросов

В вашем `APIClient` или `NetworkService` добавьте:

```swift
func makeRequest(to endpoint: String) {
    let fullURL = "\(AppConfig.apiBaseURL)\(endpoint)"
    print("🌐 Making request to: \(fullURL)")
    // ... ваш код
}
```

### Проверка сервера

```bash
# Health check
curl https://health-app-swift-go.onrender.com/healthz

# Должен вернуть:
# {"status":"ok"}
```

---

## 📱 Различия Simulator vs Device

### Simulator (Mac)
- Если в plist не указан URL, будет использоваться fallback
- `localhost` будет указывать на Mac (где может быть локальный сервер)
- Сетевые запросы идут через Mac

### Real Device (iPhone/iPad)
- **Обязательно** нужен внешний URL (не localhost)
- Должен использоваться `https://` для App Transport Security
- Нужен доступ к интернету

---

## 🎯 Режимы работы

### Development (локальный сервер)

Если нужно вернуться к локальному серверу:

1. **Вариант A: Через Info.plist**
   ```xml
   <key>API_BASE_URL</key>
   <string>http://localhost:8080</string>
   ```

2. **Вариант B: Комментировать ключ**
   ```xml
   <!-- <key>API_BASE_URL</key>
   <string>https://health-app-swift-go.onrender.com</string> -->
   ```

### Staging/Production

Можно создать разные конфигурации в Xcode:

1. **Xcode → Project → Configurations**
2. Создать: Debug, Staging, Production
3. Для каждой использовать свой Info.plist или Build Settings

---

## 🔒 App Transport Security (ATS)

### Текущая конфигурация: ✅ OK

Render использует HTTPS с валидным SSL сертификатом, поэтому дополнительные настройки ATS **не требуются**.

### Если используете локальный HTTP (только для Dev):

Добавьте в `AppInfo.plist`:

```xml
<key>NSAppTransportSecurity</key>
<dict>
    <key>NSAllowsLocalNetworking</key>
    <true/>
    <key>NSExceptionDomains</key>
    <dict>
        <key>localhost</key>
        <dict>
            <key>NSExceptionAllowsInsecureHTTPLoads</key>
            <true/>
        </dict>
    </dict>
</dict>
```

**⚠️ НЕ используйте это для production!**

---

## 📊 Проверка перед релизом

### Checklist

- [ ] `AppInfo.plist` содержит правильный `API_BASE_URL`
- [ ] Production сервер работает (health check)
- [ ] HTTPS используется (не HTTP)
- [ ] Приложение собирается без ошибок
- [ ] Вход через email/OTP работает
- [ ] Синхронизация данных работает
- [ ] Tested на реальном устройстве (не только симулятор)
- [ ] Логи показывают правильный URL
- [ ] Нет hardcoded localhost в коде

---

## 🚨 Распространенные проблемы

### Проблема: Приложение все еще использует localhost

**Решение:**
```bash
# В Xcode:
Product → Clean Build Folder (⇧⌘K)
# Удалите приложение с симулятора/устройства
# Пересоберите: Product → Build (⌘B)
# Запустите: Product → Run (⌘R)
```

### Проблема: "Failed to load resource" или Connection timeout

**Проверьте:**
1. Сервер работает: `curl https://health-app-swift-go.onrender.com/healthz`
2. Используется HTTPS (не HTTP)
3. Устройство подключено к интернету
4. Нет блокировки Firewall/VPN

### Проблема: "Unauthorized" на всех запросах

**Решение:**
1. Убедитесь, что `AUTH_REQUIRED=1` на сервере
2. Получите токен через email OTP
3. Токен передается в заголовке `Authorization: Bearer <token>`
4. Проверьте, что токен не истек (TTL по умолчанию: 30 дней)

### Проблема: CORS error в консоли

**Это не должно происходить в нативном iOS приложении!**

CORS относится только к веб-браузерам. Если видите CORS ошибку:
- Проверьте, что используете нативный URLSession (не WebView)
- Убедитесь, что не делаете запросы через WKWebView без настройки

---

## 🔄 Обновление API URL

### Если сервер переехал на новый URL:

1. Обновите `AppInfo.plist`:
   ```xml
   <key>API_BASE_URL</key>
   <string>https://new-server-url.com</string>
   ```

2. Clean Build + Rebuild
3. Протестируйте

### Если нужно поддержать несколько окружений:

Создайте отдельные конфигурации в Xcode или используйте схемы:

```swift
enum AppConfig {
    static var apiBaseURL: String {
        #if DEBUG
        return "http://localhost:8080"
        #elseif STAGING
        return "https://staging.example.com"
        #else
        return "https://health-app-swift-go.onrender.com"
        #endif
    }
}
```

---

## 📚 Связанная документация

- **Backend Setup**: `../docs/PRODUCTION_SERVER_SETUP.md`
- **API Documentation**: `../contracts/openapi.yaml`
- **Deployment Guide**: `../docs/DEPLOYMENT.md`
- **Environment Variables**: `../server/.env.example`

---

## ✅ Готово к работе!

iOS приложение настроено на production сервер:
- ✅ URL: `https://health-app-swift-go.onrender.com`
- ✅ Конфигурация в plist
- ✅ Fallback в коде
- ✅ Debug logging добавлен
- ✅ ATS совместимость

Теперь можно:
1. Собрать приложение
2. Запустить на устройстве
3. Войти через email OTP
4. Начать использовать все функции

**Приятной разработки!** 🚀
