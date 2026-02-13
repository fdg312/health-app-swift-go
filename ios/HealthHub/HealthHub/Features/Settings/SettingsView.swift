import SwiftUI

struct SettingsView: View {
  @Environment(\.dismiss) private var dismiss
  @ObservedObject private var auth = AuthManager.shared

  let onboardingText: String?
  let preloadedResponse: SettingsResponse?
  let onSaved: ((SettingsDTO) -> Void)?

  @State private var settings: SettingsDTO
  @State private var isDefault: Bool
  @State private var isLoading = false
  @State private var isSaving = false

  @State private var alertMessage: String?
  @State private var showAlert = false
  @State private var didLoad = false
  @State private var syncPreferences = SyncPreferences()

  init(
    onboardingText: String? = nil, preloadedResponse: SettingsResponse? = nil,
    onSaved: ((SettingsDTO) -> Void)? = nil
  ) {
    self.onboardingText = onboardingText
    self.preloadedResponse = preloadedResponse
    self.onSaved = onSaved

    let initial = preloadedResponse?.settings ?? SettingsView.defaultSettings()
    _settings = State(initialValue: initial)
    _isDefault = State(initialValue: preloadedResponse?.isDefault ?? false)
  }

  var body: some View {
    NavigationStack {
      Form {
        if let onboardingText {
          Section {
            Text(onboardingText)
              .font(.subheadline)
          }
        }

        if isDefault {
          Section {
            Label("По умолчанию", systemImage: "exclamationmark.circle")
              .foregroundStyle(.orange)
          } footer: {
            Text("Эти значения взяты из серверных fallback defaults.")
          }
        }

        Section {
          Toggle("Фоновая синхронизация", isOn: backgroundSyncBinding)
          Toggle("Обновлять напоминания в фоне", isOn: backgroundRemindersBinding)
            .disabled(!backgroundSyncBinding.wrappedValue)
        } header: {
          Text("Фоновая синхронизация")
        } footer: {
          Text(
            "iOS может ограничивать фоновые обновления. На реальном устройстве работает стабильнее."
          )
        }

        Section("Часовой пояс") {
          TextField("Europe/Moscow", text: timeZoneBinding)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
        }

        Section("Тихие часы") {
          Toggle("Включить тихие часы", isOn: quietEnabledBinding)

          if quietEnabledBinding.wrappedValue {
            DatePicker("Начало", selection: quietStartBinding, displayedComponents: .hourAndMinute)
            DatePicker("Окончание", selection: quietEndBinding, displayedComponents: .hourAndMinute)
          }
        }

        Section("Лимиты") {
          Stepper(
            "Максимум уведомлений в день: \(settings.notificationsMaxPerDay)",
            value: $settings.notificationsMaxPerDay, in: 0...10)
        }

        Section("Пороги") {
          Stepper(
            "Сон минимум: \(settings.minSleepMinutes) мин", value: $settings.minSleepMinutes,
            in: 0...1200, step: 10)
          Stepper(
            "Шаги минимум: \(settings.minSteps)", value: $settings.minSteps, in: 0...50000,
            step: 500)
          Stepper(
            "Активная энергия минимум: \(settings.minActiveEnergyKcal) ккал",
            value: $settings.minActiveEnergyKcal, in: 0...5000, step: 10)
        }

        Section("Расписание") {
          DatePicker(
            "Утренний чек-ин", selection: morningTimeBinding, displayedComponents: .hourAndMinute)
          DatePicker(
            "Вечерний чек-ин", selection: eveningTimeBinding, displayedComponents: .hourAndMinute)
          DatePicker(
            "Добавки / витамины", selection: vitaminsTimeBinding,
            displayedComponents: .hourAndMinute)
        }
      }
      .disabled(isLoading || isSaving)
      .overlay {
        if isLoading || isSaving {
          ProgressView()
        }
      }
      .navigationTitle("Настройки")
      .navigationBarTitleDisplayMode(.inline)
      .toolbar {
        ToolbarItem(placement: .topBarLeading) {
          Button("Закрыть") {
            dismiss()
          }
        }
        ToolbarItem(placement: .topBarTrailing) {
          Button("Сохранить") {
            Task { await saveSettings() }
          }
          .disabled(isLoading || isSaving)
        }
      }
      .task {
        await loadIfNeeded()
      }
      .alert("Ошибка", isPresented: $showAlert) {
        Button("OK", role: .cancel) {}
      } message: {
        Text(alertMessage ?? "Попробуйте позже")
      }
    }
  }

  private var timeZoneBinding: Binding<String> {
    Binding(
      get: { settings.timeZone ?? "" },
      set: { newValue in
        let trimmed = newValue.trimmingCharacters(in: .whitespacesAndNewlines)
        settings.timeZone = trimmed.isEmpty ? nil : trimmed
      }
    )
  }

  private var backgroundSyncBinding: Binding<Bool> {
    Binding(
      get: { syncPreferences.backgroundSyncEnabled },
      set: { enabled in
        syncPreferences.backgroundSyncEnabled = enabled

        if enabled {
          if auth.isAuthenticated {
            Task {
              do {
                try await HealthKitManager.shared.enableBackgroundDelivery()
                HealthKitManager.shared.startObserverQueries {
                  Task { @MainActor in
                    BackgroundSyncManager.shared.handleHealthKitObserverUpdate()
                  }
                }
                BackgroundSyncManager.shared.scheduleAppRefresh(after: 5 * 60)
              } catch {
                await MainActor.run {
                  alertMessage =
                    "Не удалось включить фоновую синхронизацию: \(error.localizedDescription)"
                  showAlert = true
                }
              }
            }
          }
        } else {
          HealthKitManager.shared.stopObserverQueries()
          BackgroundSyncManager.shared.cancelScheduledRefresh()
        }
      }
    )
  }

  private var backgroundRemindersBinding: Binding<Bool> {
    Binding(
      get: { syncPreferences.backgroundRemindersEnabled },
      set: { enabled in
        syncPreferences.backgroundRemindersEnabled = enabled
      }
    )
  }

  private var quietEnabledBinding: Binding<Bool> {
    Binding(
      get: { settings.quietStartMinutes != nil && settings.quietEndMinutes != nil },
      set: { enabled in
        if enabled {
          if settings.quietStartMinutes == nil { settings.quietStartMinutes = 23 * 60 }
          if settings.quietEndMinutes == nil { settings.quietEndMinutes = 8 * 60 }
        } else {
          settings.quietStartMinutes = nil
          settings.quietEndMinutes = nil
        }
      }
    )
  }

  private var quietStartBinding: Binding<Date> {
    Binding(
      get: { minutesToDate(settings.quietStartMinutes ?? 23 * 60) },
      set: { settings.quietStartMinutes = dateToMinutes($0) }
    )
  }

  private var quietEndBinding: Binding<Date> {
    Binding(
      get: { minutesToDate(settings.quietEndMinutes ?? 8 * 60) },
      set: { settings.quietEndMinutes = dateToMinutes($0) }
    )
  }

  private var morningTimeBinding: Binding<Date> {
    Binding(
      get: { minutesToDate(settings.morningCheckinTimeMinutes) },
      set: { settings.morningCheckinTimeMinutes = dateToMinutes($0) }
    )
  }

  private var eveningTimeBinding: Binding<Date> {
    Binding(
      get: { minutesToDate(settings.eveningCheckinTimeMinutes) },
      set: { settings.eveningCheckinTimeMinutes = dateToMinutes($0) }
    )
  }

  private var vitaminsTimeBinding: Binding<Date> {
    Binding(
      get: { minutesToDate(settings.vitaminsTimeMinutes) },
      set: { settings.vitaminsTimeMinutes = dateToMinutes($0) }
    )
  }

  private func loadIfNeeded() async {
    if didLoad {
      return
    }
    didLoad = true

    if let preloadedResponse {
      settings = preloadedResponse.settings
      isDefault = preloadedResponse.isDefault
      return
    }

    isLoading = true
    defer { isLoading = false }

    do {
      let response = try await APIClient.shared.fetchSettings()
      settings = response.settings
      isDefault = response.isDefault
      syncLegacyLocalSettings(from: response.settings)
    } catch {
      if !handleError(error) {
        show(error, prefix: "Ошибка загрузки настроек")
      }
    }
  }

  private func saveSettings() async {
    isSaving = true
    defer { isSaving = false }

    do {
      let saved = try await APIClient.shared.updateSettings(settings)
      settings = saved
      isDefault = false
      syncLegacyLocalSettings(from: saved)
      onSaved?(saved)
      dismiss()
    } catch {
      if !handleError(error) {
        show(error, prefix: "Ошибка сохранения настроек")
      }
    }
  }

  private func handleError(_ error: Error) -> Bool {
    if let apiError = error as? APIError, apiError == .unauthorized {
      auth.handleUnauthorized()
      return true
    }
    if let apiError = error as? APIError, apiError == .rateLimited {
      alertMessage = "Ошибка настроек (\(apiError.uiCode))"
      showAlert = true
      return true
    }
    return false
  }

  private func show(_ error: Error, prefix: String) {
    alertMessage = "\(prefix) (\(errorCode(error)))"
    showAlert = true
  }

  private func errorCode(_ error: Error) -> String {
    if let apiError = error as? APIError {
      return apiError.uiCode
    }
    return "bad_response"
  }

  private func syncLegacyLocalSettings(from settings: SettingsDTO) {
    var local = NotificationSettings()
    local.maxLocalPerDay = settings.notificationsMaxPerDay
    local.setMorningCheckinTime(minutesToDate(settings.morningCheckinTimeMinutes))
    local.setEveningCheckinTime(minutesToDate(settings.eveningCheckinTimeMinutes))
    local.setActivityNudgeTime(minutesToDate(settings.vitaminsTimeMinutes))

    if let quietStart = settings.quietStartMinutes, let quietEnd = settings.quietEndMinutes {
      local.quietModeEnabled = true
      local.quietStartMinutes = quietStart
      local.quietEndMinutes = quietEnd
    } else {
      local.quietModeEnabled = false
    }
  }

  private func minutesToDate(_ minutes: Int) -> Date {
    let calendar = Calendar.current
    let now = Date()
    var components = calendar.dateComponents([.year, .month, .day], from: now)
    components.hour = max(0, min(23, minutes / 60))
    components.minute = max(0, min(59, minutes % 60))
    return calendar.date(from: components) ?? now
  }

  private func dateToMinutes(_ date: Date) -> Int {
    let calendar = Calendar.current
    let hour = calendar.component(.hour, from: date)
    let minute = calendar.component(.minute, from: date)
    return hour * 60 + minute
  }

  private static func defaultSettings() -> SettingsDTO {
    SettingsDTO(
      timeZone: TimeZone.current.identifier,
      quietStartMinutes: 23 * 60,
      quietEndMinutes: 8 * 60,
      notificationsMaxPerDay: 4,
      minSleepMinutes: 420,
      minSteps: 6000,
      minActiveEnergyKcal: 250,
      morningCheckinTimeMinutes: 540,
      eveningCheckinTimeMinutes: 1260,
      vitaminsTimeMinutes: 720
    )
  }
}

struct SettingsView_Previews: PreviewProvider {
  static var previews: some View {
    SettingsView()
  }
}
