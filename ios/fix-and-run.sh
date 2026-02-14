#!/bin/bash

echo "🚀 Health Hub - Исправление и запуск"
echo "===================================="
echo ""

# 1. Закрыть все
echo "1️⃣ Закрываем Xcode и симуляторы..."
killall Simulator 2>/dev/null
killall Xcode 2>/dev/null
sleep 1

# 2. Очистить кэш
echo "2️⃣ Очищаем кэш Xcode..."
rm -rf ~/Library/Developer/Xcode/DerivedData/HealthHub-*

# 3. Сбросить симуляторы
echo "3️⃣ Сбрасываем симуляторы..."
xcrun simctl shutdown all 2>/dev/null
xcrun simctl erase all 2>/dev/null

echo ""
echo "✅ Готово!"
echo ""
echo "Теперь:"
echo "  1. Откройте Xcode: open HealthHub/HealthHub.xcodeproj"
echo "  2. Выберите любой iPhone симулятор"
echo "  3. Нажмите ▶ (Play)"
echo ""
