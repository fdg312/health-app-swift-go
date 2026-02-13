//
//  HomeView.swift
//  HealthHub
//
//  Created by HealthHub Team
//

import SwiftUI

struct HomeView: View {
  @ObservedObject private var auth = AuthManager.shared
  @StateObject private var backgroundSync = BackgroundSyncManager.shared
  @EnvironmentObject private var navigation: AppNavigationState

  @State private var profiles: [ProfileDTO] = []
  @State private var feedDay: FeedDayResponse?
  @State private var workoutsToday: WorkoutTodayResponse?
  @State private var selectedDate = Date()

  @State private var isLoading = false
  @State private var isLoadingWorkouts = false
  @State private var isSavingMorning = false
  @State private var errorMessage: String?

  @State private var showAddWaterSheet = false
  @State private var showAddSupplementSheet = false
  @State private var showAccountSheet = false
  @State private var showSettingsSheet = false
  @State private var settingsResponse: SettingsResponse?
  @State private var settingsOnboardingText: String?
  @State private var didShowSettingsOnboarding = false
  @State private var showSessionExpiredAlert = false
  @State private var showRateLimitedAlert = false

  private var ownerProfile: ProfileDTO? {
    profiles.first(where: { $0.type == "owner" })
  }

  private var weekDays: [Date] {
    let calendar = Calendar(identifier: .iso8601)
    let weekStart = calendar.dateInterval(of: .weekOfYear, for: selectedDate)?.start ?? selectedDate
    return (0..<7).compactMap { calendar.date(byAdding: .day, value: $0, to: weekStart) }
  }

  private var isToday: Bool {
    Calendar.current.isDateInToday(selectedDate)
  }

  private var isEvening: Bool {
    let hour = Calendar.current.component(.hour, from: Date())
    return hour >= 18
  }

  var body: some View {
    NavigationStack {
      ScrollView {
        VStack(alignment: .leading, spacing: 20) {
          // Шапка с приветствием
          headerSection

          // Мини-календарь на неделю
          weekCalendarSection

          if isLoading {
            loadingView
          } else {
            // Блок "Сегодня" с ключевыми метриками
            todayStatsSection

            // Утренний чек-ин (только если сегодня и еще не заполнен)
            if isToday && feedDay?.checkins.morning == nil {
              morningCheckinSection
            }

            // Вечерний чек-ин (после 18:00 и если еще не заполнен)
            if isToday && isEvening && feedDay?.checkins.evening == nil {
              eveningCheckinSection
            }

            // План питания сегодня
            mealPlanSection

            // Тренировки сегодня
            workoutsSection

            // Quick Actions
            quickActionsSection
          }

          if let errorMessage {
            Text(errorMessage)
              .font(.caption)
              .foregroundStyle(.red)
              .padding(.horizontal, 4)
          }
        }
        .padding()
      }
      .navigationTitle("Главная")
      .toolbar {
        ToolbarItem(placement: .topBarLeading) {
          Button {
            showAccountSheet = true
          } label: {
            Image(systemName: "person.crop.circle")
          }
        }

        ToolbarItem(placement: .topBarTrailing) {
          Button {
            settingsOnboardingText = nil
            showSettingsSheet = true
          } label: {
            Image(systemName: "gearshape")
          }
        }
      }
      .refreshable {
        await loadData()
      }
      .task {
        await loadData()
      }
      .onChange(of: selectedDate) { _, _ in
        Task { await loadFeedDay() }
      }
      .sheet(isPresented: $showAddWaterSheet) {
        if let ownerProfile {
          AddWaterSheet(profileId: ownerProfile.id) {
            Task { await loadFeedDay() }
          }
        }
      }
      .sheet(isPresented: $showAddSupplementSheet) {
        if let ownerProfile {
          CreateSupplementSheet(profileId: ownerProfile.id) {
            Task { await loadFeedDay() }
          }
        }
      }
      .sheet(isPresented: $showAccountSheet) {
        AccountSheet()
      }
      .sheet(isPresented: $showSettingsSheet) {
        SettingsView(
          onboardingText: settingsOnboardingText,
          preloadedResponse: settingsResponse
        ) { saved in
          settingsResponse = SettingsResponse(settings: saved, isDefault: false)
          settingsOnboardingText = nil
        }
      }
      .alert("Слишком много запросов", isPresented: $showRateLimitedAlert) {
        Button("OK", role: .cancel) {}
      } message: {
        Text("Попробуйте позже")
      }
      .alert("Сессия истекла", isPresented: $showSessionExpiredAlert) {
        Button("OK") {
          auth.handleUnauthorized()
        }
      } message: {
        Text("Войдите заново")
      }
    }
  }

  // MARK: - Header Section

  private var headerSection: some View {
    HStack {
      VStack(alignment: .leading, spacing: 4) {
        Text("Привет!")
          .font(.system(size: 28, weight: .bold))
        Text(formattedLongDate(selectedDate))
          .font(.subheadline)
          .foregroundStyle(.secondary)
      }

      Spacer()

      Circle()
        .fill(Color(.systemGray5))
        .frame(width: 48, height: 48)
        .overlay(
          Image(systemName: "person.fill")
            .font(.system(size: 20))
            .foregroundStyle(.secondary)
        )
    }
  }

  // MARK: - Week Calendar Section

  private var weekCalendarSection: some View {
    HStack(spacing: 8) {
      ForEach(weekDays, id: \.self) { date in
        Button {
          selectedDate = date
        } label: {
          VStack(spacing: 6) {
            Text(shortWeekday(date))
              .font(.system(size: 11, weight: .medium))
              .textCase(.uppercase)
            Text(dayNumber(date))
              .font(.system(size: 16, weight: .semibold))
          }
          .frame(maxWidth: .infinity)
          .padding(.vertical, 12)
          .background(
            RoundedRectangle(cornerRadius: 12)
              .fill(isSelected(date) ? Color.accentColor : Color(.secondarySystemGroupedBackground))
          )
          .foregroundStyle(isSelected(date) ? .white : .primary)
        }
        .buttonStyle(.plain)
      }
    }
  }

  // MARK: - Loading View

  private var loadingView: some View {
    AppCard {
      HStack {
        Spacer()
        ProgressView("Загружаю...")
        Spacer()
      }
    }
  }

  // MARK: - Today Stats Section

  private var todayStatsSection: some View {
    VStack(alignment: .leading, spacing: 16) {
      SectionHeader("Сегодня")

      LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
        // Шаги
        StatTile(
          title: "Шаги",
          value: feedDay?.daily?.activity?.steps.map { "\($0)" } ?? "—",
          subtitle: "цель 10,000",
          icon: "figure.walk",
          iconColor: .green
        )

        // Сон
        StatTile(
          title: "Сон",
          value: formatSleep(),
          subtitle: feedDay?.daily?.sleep?.totalMinutes != nil ? "качество сна" : nil,
          icon: "bed.double.fill",
          iconColor: .indigo
        )

        // Питание
        StatTile(
          title: "Калории",
          value: formatCalories(),
          subtitle: formatCaloriesSubtitle(),
          icon: "fork.knife",
          iconColor: .orange
        )

        // Энергия
        StatTile(
          title: "Энергия",
          value: formatEnergy(),
          subtitle: feedDay?.checkins.morning != nil ? "утренний чек-ин" : nil,
          icon: "bolt.fill",
          iconColor: .yellow
        )
      }
    }
  }

  // MARK: - Morning Checkin Section

  private var morningCheckinSection: some View {
    AppCard {
      VStack(alignment: .leading, spacing: 12) {
        HStack {
          Image(systemName: "sun.max.fill")
            .font(.system(size: 20))
            .foregroundStyle(.orange)

          Text("Как прошло утро?")
            .font(.system(size: 17, weight: .semibold))
        }

        Text("Оцените ваше утреннее состояние")
          .font(.subheadline)
          .foregroundStyle(.secondary)

        HStack(spacing: 12) {
          ForEach(1...5, id: \.self) { score in
            Button {
              Task { await upsertMorning(score: score) }
            } label: {
              Image(systemName: "star.fill")
                .font(.system(size: 28))
                .foregroundStyle(score <= 3 ? .orange : .yellow)
            }
            .buttonStyle(.plain)
            .disabled(isSavingMorning || ownerProfile == nil)
          }

          if isSavingMorning {
            ProgressView()
              .controlSize(.small)
          }
        }
        .frame(maxWidth: .infinity, alignment: .center)
        .padding(.vertical, 8)
      }
    }
  }

  // MARK: - Evening Checkin Section

  private var eveningCheckinSection: some View {
    AppCard {
      VStack(alignment: .leading, spacing: 12) {
        HStack {
          Image(systemName: "moon.fill")
            .font(.system(size: 20))
            .foregroundStyle(.indigo)

          Text("Вечерний чек-ин")
            .font(.system(size: 17, weight: .semibold))
        }

        Text("Заполните дневник за сегодня")
          .font(.subheadline)
          .foregroundStyle(.secondary)

        PrimaryButton("Заполнить чек-ин", icon: "square.and.pencil") {
          navigation.openFeed(date: selectedDate, editType: .evening)
        }
      }
    }
  }

  // MARK: - Meal Plan Section

  private var mealPlanSection: some View {
    VStack(alignment: .leading, spacing: 16) {
      SectionHeader(
        "План питания",
        actionTitle: feedDay?.mealPlanTitle != nil ? "Все" : nil,
        action: feedDay?.mealPlanTitle != nil
          ? {
            navigation.selectedTab = .activity
          } : nil
      )

      AppCard {
        if let planTitle = feedDay?.mealPlanTitle {
          VStack(alignment: .leading, spacing: 12) {
            HStack {
              Image(systemName: "calendar.badge.clock")
                .foregroundStyle(.green)
              Text(planTitle)
                .font(.subheadline)
                .foregroundStyle(.secondary)
            }

            if let mealToday = feedDay?.mealToday, !mealToday.isEmpty {
              VStack(alignment: .leading, spacing: 10) {
                ForEach(mealToday.prefix(3)) { item in
                  HStack(alignment: .top, spacing: 12) {
                    TagChip(
                      slotName(item.mealSlot),
                      color: slotColor(item.mealSlot)
                    )

                    VStack(alignment: .leading, spacing: 2) {
                      Text(item.title)
                        .font(.subheadline)
                        .fontWeight(.medium)

                      if item.approxKcal > 0 {
                        Text("\(item.approxKcal) ккал")
                          .font(.caption)
                          .foregroundStyle(.secondary)
                      }
                    }

                    Spacer()
                  }
                }

                if mealToday.count > 3 {
                  Text("+ еще \(mealToday.count - 3)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                }
              }
            } else {
              Text("На сегодня нет записей плана")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .padding(.vertical, 8)
            }
          }
        } else {
          EmptyStateView(
            icon: "fork.knife.circle",
            title: "Нет плана питания",
            description: "Сгенерируйте персональный план через чат с AI ассистентом",
            actionTitle: "Открыть чат",
            action: {
              navigation.selectedTab = .chat
            }
          )
        }
      }
    }
  }

  // MARK: - Workouts Section

  private var workoutsSection: some View {
    VStack(alignment: .leading, spacing: 16) {
      SectionHeader(
        "Тренировки",
        actionTitle: workoutsToday?.planned.isEmpty == false ? "План" : nil,
        action: workoutsToday?.planned.isEmpty == false
          ? {
            navigation.selectedTab = .activity
          } : nil
      )

      AppCard {
        if isLoadingWorkouts {
          HStack {
            Spacer()
            ProgressView()
            Spacer()
          }
          .padding(.vertical, 20)
        } else if let workouts = workoutsToday, !workouts.planned.isEmpty {
          VStack(alignment: .leading, spacing: 12) {
            if workouts.isDone {
              HStack(spacing: 10) {
                Image(systemName: "checkmark.circle.fill")
                  .font(.system(size: 24))
                  .foregroundStyle(.green)
                Text("Все тренировки выполнены!")
                  .font(.subheadline)
                  .fontWeight(.medium)
              }
              .padding(.vertical, 8)
            } else {
              ForEach(workouts.planned.prefix(3)) { item in
                HStack(alignment: .center, spacing: 12) {
                  VStack(alignment: .leading, spacing: 4) {
                    Text(item.kindLocalized)
                      .font(.subheadline)
                      .fontWeight(.semibold)
                    Text("\(item.durationMin) мин • \(item.intensityLocalized)")
                      .font(.caption)
                      .foregroundStyle(.secondary)
                  }

                  Spacer()

                  if let completion = workouts.completions.first(where: {
                    $0.planItemId == item.id
                  }) {
                    Image(
                      systemName: completion.status == "done"
                        ? "checkmark.circle.fill" : "xmark.circle"
                    )
                    .font(.title3)
                    .foregroundStyle(completion.status == "done" ? .green : .orange)
                  } else {
                    Button {
                      Task { await markWorkoutDone(item) }
                    } label: {
                      Image(systemName: "circle")
                        .font(.title3)
                        .foregroundStyle(.secondary)
                    }
                    .buttonStyle(.plain)
                  }
                }
                .padding(.vertical, 6)
              }

              if workouts.planned.count > 3 {
                Text("+ еще \(workouts.planned.count - 3)")
                  .font(.caption)
                  .foregroundStyle(.secondary)
                  .padding(.top, 4)
              }
            }
          }
        } else {
          EmptyStateView(
            icon: "figure.run.circle",
            title: "Нет тренировок на сегодня",
            description: "Добавьте тренировки в свой план"
          )
        }
      }
    }
  }

  // MARK: - Quick Actions Section

  private var quickActionsSection: some View {
    VStack(alignment: .leading, spacing: 16) {
      SectionHeader("Быстрые действия")

      LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
        QuickActionButton(
          title: "Вода",
          icon: "drop.fill",
          color: .cyan
        ) {
          showAddWaterSheet = true
        }

        QuickActionButton(
          title: "Добавка",
          icon: "pills.fill",
          color: .purple
        ) {
          showAddSupplementSheet = true
        }

        QuickActionButton(
          title: "Источник",
          icon: "paperclip",
          color: .orange
        ) {
          navigation.openActivity()
        }

        QuickActionButton(
          title: "Дневник",
          icon: "book.fill",
          color: .green
        ) {
          navigation.selectedTab = .feed
        }
      }
    }
  }

  // MARK: - Formatters

  private func formatSleep() -> String {
    guard let totalMinutes = feedDay?.daily?.sleep?.totalMinutes else { return "—" }
    let hours = totalMinutes / 60
    let minutes = totalMinutes % 60
    return "\(hours)ч \(minutes)м"
  }

  private func formatCalories() -> String {
    guard let kcal = feedDay?.daily?.nutrition?.energyKcal, kcal > 0 else { return "—" }
    return "\(kcal)"
  }

  private func formatCaloriesSubtitle() -> String? {
    guard let nutrition = feedDay?.daily?.nutrition,
      let targets = feedDay?.nutritionTargets,
      nutrition.energyKcal > 0
    else { return nil }
    return "из \(targets.caloriesKcal) ккал"
  }

  private func formatEnergy() -> String {
    guard let score = feedDay?.checkins.morning?.score else { return "—" }
    return "\(score)/5"
  }

  private func slotName(_ slot: String) -> String {
    switch slot {
    case "breakfast": return "Завтрак"
    case "lunch": return "Обед"
    case "dinner": return "Ужин"
    case "snack": return "Перекус"
    default: return slot
    }
  }

  private func slotColor(_ slot: String) -> Color {
    switch slot {
    case "breakfast": return .orange
    case "lunch": return .green
    case "dinner": return .indigo
    case "snack": return .purple
    default: return .blue
    }
  }

  // MARK: - API Calls

  private func loadData() async {
    isLoading = true
    errorMessage = nil

    do {
      profiles = try await APIClient.shared.listProfiles()
      await loadFeedDay()
      await loadWorkoutsToday()

      if !didShowSettingsOnboarding {
        await loadSettings(showOnboarding: true)
        didShowSettingsOnboarding = true
      }
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка загрузки: \(error.localizedDescription)"
      }
    }

    isLoading = false
  }

  private func loadFeedDay() async {
    guard let profile = ownerProfile else { return }

    do {
      feedDay = try await APIClient.shared.fetchFeedDay(
        profileId: profile.id,
        date: selectedDate
      )
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка загрузки дня: \(error.localizedDescription)"
      }
    }
  }

  private func loadWorkoutsToday() async {
    guard let profile = ownerProfile else { return }

    isLoadingWorkouts = true
    let dateStr = formatDate(selectedDate)

    do {
      workoutsToday = try await APIClient.shared.fetchWorkoutToday(
        profileId: profile.id,
        date: dateStr
      )
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка загрузки тренировок: \(error.localizedDescription)"
      }
    }

    isLoadingWorkouts = false
  }

  private func markWorkoutDone(_ item: WorkoutItemDTO) async {
    guard let profile = ownerProfile else { return }

    let dateStr = formatDate(selectedDate)

    do {
      _ = try await APIClient.shared.upsertWorkoutCompletion(
        profileId: profile.id,
        date: dateStr,
        planItemId: item.id,
        status: "done"
      )
      await loadWorkoutsToday()
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка отметки тренировки: \(error.localizedDescription)"
      }
    }
  }

  private func upsertMorning(score: Int) async {
    guard let profile = ownerProfile else { return }

    isSavingMorning = true
    errorMessage = nil

    let dateStr = formatDate(selectedDate)
    let request = UpsertCheckinRequest(
      profileId: profile.id,
      date: dateStr,
      type: .morning,
      score: score,
      tags: nil,
      note: nil
    )

    do {
      _ = try await APIClient.shared.upsertCheckin(request)
      await loadFeedDay()
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка сохранения чек-ина: \(error.localizedDescription)"
      }
    }

    isSavingMorning = false
  }

  private func loadSettings(showOnboarding: Bool) async {
    do {
      let response = try await APIClient.shared.fetchSettings()
      settingsResponse = response

      if showOnboarding, response.isDefault {
        settingsOnboardingText =
          "Установите цели питания, чтобы отслеживать прогресс. Текущие значения по умолчанию."
        showSettingsSheet = true
      }
    } catch {
      // Игнорируем ошибки настроек
    }
  }

  private func manualSync() async {
    let success = await backgroundSync.performSync(reason: "manual")
    if success {
      await loadData()
    } else if let message = backgroundSync.lastErrorMessage, !message.isEmpty {
      errorMessage = "Синхронизация: \(message)"
    } else {
      errorMessage = "Синхронизация не выполнена"
    }
  }

  private func handleError(_ error: Error) -> Bool {
    if let apiError = error as? APIError {
      switch apiError {
      case .unauthorized:
        showSessionExpiredAlert = true
        return true
      case .rateLimited:
        showRateLimitedAlert = true
        return true
      default:
        break
      }
    }
    return false
  }

  // MARK: - Date Formatters

  private func formatDate(_ date: Date) -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    return formatter.string(from: date)
  }

  private func formattedLongDate(_ date: Date) -> String {
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "ru_RU")
    formatter.dateFormat = "d MMMM, EEEE"
    return formatter.string(from: date)
  }

  private func shortWeekday(_ date: Date) -> String {
    let formatter = DateFormatter()
    formatter.locale = Locale(identifier: "ru_RU")
    formatter.dateFormat = "EEE"
    return formatter.string(from: date)
  }

  private func dayNumber(_ date: Date) -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "d"
    return formatter.string(from: date)
  }

  private func isSelected(_ date: Date) -> Bool {
    Calendar.current.isDate(date, inSameDayAs: selectedDate)
  }
}

// MARK: - Quick Action Button

struct QuickActionButton: View {
  let title: String
  let icon: String
  let color: Color
  let action: () -> Void

  var body: some View {
    Button(action: action) {
      VStack(spacing: 12) {
        Image(systemName: icon)
          .font(.system(size: 28))
          .foregroundStyle(color)

        Text(title)
          .font(.system(size: 14, weight: .medium))
          .foregroundStyle(.primary)
      }
      .frame(maxWidth: .infinity)
      .padding(.vertical, 20)
      .background(
        RoundedRectangle(cornerRadius: 12)
          .fill(Color(.secondarySystemGroupedBackground))
      )
    }
    .buttonStyle(.plain)
  }
}

// MARK: - Account Sheet

struct AccountSheet: View {
  @Environment(\.dismiss) private var dismiss
  @ObservedObject private var auth = AuthManager.shared

  var body: some View {
    NavigationStack {
      VStack(spacing: 20) {
        Text("Аккаунт")
          .font(.headline)

        PrimaryButton("Выйти", icon: "rectangle.portrait.and.arrow.right") {
          auth.logout()
          dismiss()
        }
      }
      .padding()
      .navigationTitle("Аккаунт")
      .navigationBarTitleDisplayMode(.inline)
      .toolbar {
        ToolbarItem(placement: .topBarTrailing) {
          Button("Закрыть") {
            dismiss()
          }
        }
      }
    }
  }
}

// MARK: - Previews

#Preview {
  HomeView()
    .environmentObject(AppNavigationState())
}
