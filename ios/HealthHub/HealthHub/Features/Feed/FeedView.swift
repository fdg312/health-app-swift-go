import SwiftUI

struct FeedView: View {
  @ObservedObject private var auth = AuthManager.shared
  @EnvironmentObject private var navigation: AppNavigationState
  @State private var showSessionExpired = false
  @State private var showRateLimited = false
  @State private var profiles: [ProfileDTO] = []
  @State private var feedDay: FeedDayResponse?
  @State private var isLoading = false
  @State private var errorMessage: String?
  @State private var serverStatus: String = "Проверяю..."
  @State private var selectedDate = Date()

  // Editing state for morning checkin
  @State private var isEditingMorning = false
  @State private var draftMorningScore: Int = 3
  @State private var draftMorningNote: String = ""
  @State private var draftMorningTags: [String] = []
  @State private var newMorningTag: String = ""

  // Editing state for evening checkin
  @State private var isEditingEvening = false
  @State private var draftEveningScore: Int = 3
  @State private var draftEveningNote: String = ""
  @State private var draftEveningTags: [String] = []
  @State private var newEveningTag: String = ""

  // UI state
  @State private var showDeleteConfirmation = false
  @State private var deleteTarget: CheckinType?
  @State private var isSaving = false
  @State private var tagError: String?

  // Share state
  @State private var shareImage: UIImage?
  @State private var showShareSheet = false

  // Inbox state
  @State private var showInbox = false
  @State private var unreadCount: Int = 0
  @State private var showNotificationSettings = false
  @State private var pendingAutoEditType: CheckinType?

  private var ownerProfile: ProfileDTO? {
    profiles.first(where: { $0.type == "owner" })
  }

  var body: some View {
    NavigationStack {
      mainList
        .navigationTitle("Сводка дня")
        .toolbar { toolbarContent }
        .refreshable { await loadData() }
        .task { await loadData() }
        .sheet(isPresented: $showShareSheet) {
          if let image = shareImage {
            ShareSheet(activityItems: [image])
          }
        }
        .sheet(isPresented: $showInbox) {
          if let profile = ownerProfile {
            InboxSheet(
              profileId: profile.id,
              onDismiss: {
                Task { await loadUnreadCount() }
              })
          }
        }
        .sheet(isPresented: $showNotificationSettings) {
          SettingsView()
        }
        .onChange(of: navigation.feedNavigationRequest?.id) { _, _ in
          Task { await applyFeedNavigationRequestIfNeeded() }
        }
        .alert("Удалить чек-ин?", isPresented: $showDeleteConfirmation) {
          Button("Удалить", role: .destructive) {
            Task {
              if let target = deleteTarget {
                await deleteCheckin(type: target)
              }
            }
          }
          Button("Отмена", role: .cancel) {}
        } message: {
          Text("Это действие нельзя отменить")
        }
        .alert("Сессия истекла", isPresented: $showSessionExpired) {
          Button("OK") {
            auth.handleUnauthorized()
          }
        } message: {
          Text("Войдите заново")
        }
        .alert("Слишком много запросов", isPresented: $showRateLimited) {
          Button("OK", role: .cancel) {}
        } message: {
          Text("Попробуйте позже")
        }
    }
  }

  /// Centralized unauthorized check — shows alert and returns true if 401.
  private func handleError(_ error: Error) -> Bool {
    if let apiError = error as? APIError, apiError == .unauthorized {
      showSessionExpired = true
      return true
    }
    if let apiError = error as? APIError, apiError == .rateLimited {
      showRateLimited = true
      return true
    }
    return false
  }

  // MARK: - Main List

  @ViewBuilder
  private var mainList: some View {
    List {
      serverStatusSection
      datePickerSection

      if let feed = feedDay {
        dailyMetricsSection(feed)
        workoutsSection(feed)
        checkinsSection(feed)
        missingFieldsSection(feed)
        #if DEBUG
          debugSection
        #endif
      } else if isLoading {
        Section {
          HStack {
            Spacer()
            ProgressView("Загружаю...")
            Spacer()
          }
        }
      } else if let error = errorMessage {
        Section {
          Text("Ошибка: " + error)
            .foregroundStyle(.red)
            .font(.caption)
        }
      }
    }
  }

  // MARK: - Toolbar

  @ToolbarContentBuilder
  private var toolbarContent: some ToolbarContent {
    ToolbarItem(placement: .topBarLeading) {
      HStack(spacing: 12) {
        Button {
          showInbox = true
        } label: {
          ZStack(alignment: .topTrailing) {
            Image(systemName: "bell.fill")

            if unreadCount > 0 {
              Text("\(unreadCount)")
                .font(.caption2)
                .fontWeight(.bold)
                .foregroundStyle(.white)
                .padding(4)
                .background(Circle().fill(Color.red))
                .offset(x: 8, y: -8)
            }
          }
        }

        Button {
          showNotificationSettings = true
        } label: {
          Image(systemName: "gearshape")
        }
      }
    }

    ToolbarItem(placement: .topBarTrailing) {
      Button {
        Task {
          await generateShareImage()
        }
      } label: {
        Image(systemName: "square.and.arrow.up")
      }
      .disabled(feedDay == nil)
    }
  }

  // MARK: - Sections

  @ViewBuilder
  private var serverStatusSection: some View {
    Section("Статус сервера") {
      HStack {
        Image(
          systemName: serverStatus == "OK"
            ? "checkmark.circle.fill" : "exclamationmark.triangle.fill"
        )
        .foregroundStyle(serverStatus == "OK" ? .green : .orange)
        Text(serverStatus)
      }
    }
  }

  @ViewBuilder
  private var datePickerSection: some View {
    Section("Дата") {
      DatePicker("Выбрать дату", selection: $selectedDate, displayedComponents: .date)
        .datePickerStyle(.compact)
        .onChange(of: selectedDate) { _, _ in
          Task { await loadFeedDay() }
        }
    }
  }

  @ViewBuilder
  private func dailyMetricsSection(_ feed: FeedDayResponse) -> some View {
    if let daily = feed.daily {
      Section("Показатели дня") {
        dailyMetricsContent(daily)
      }
    } else if !feed.missingFields.contains("daily") {
      Section("Показатели дня") {
        Text("Нет данных")
          .foregroundStyle(.secondary)
          .font(.caption)
      }
    }
  }

  @ViewBuilder
  private func dailyMetricsContent(_ daily: DailyAggregate) -> some View {
    if let steps = daily.activity?.steps {
      MetricRow(icon: "figure.walk", label: "Шаги", value: "\(steps)")
    }

    if let weight = daily.body?.weightKgLast, weight > 0 {
      MetricRow(icon: "scalemass", label: "Вес", value: String(format: "%.1f кг", weight))
    }

    if let restingHR = daily.heart?.restingHrBpm, restingHR > 0 {
      MetricRow(icon: "heart.fill", label: "Пульс покоя", value: "\(restingHR) bpm")
    }

    if let totalMin = daily.sleep?.totalMinutes {
      let hours = totalMin / 60
      let minutes = totalMin % 60
      MetricRow(icon: "bed.double.fill", label: "Сон", value: "\(hours)ч \(minutes)м")
    }

    if let nutrition = daily.nutrition {
      let kcal = nutrition.energyKcal
      let protein = nutrition.proteinG
      let fat = nutrition.fatG
      let carbs = nutrition.carbsG

      if kcal > 0 || protein > 0 || fat > 0 || carbs > 0 {
        MetricRow(
          icon: "fork.knife", label: "Питание",
          value: "Ккал: \(kcal) • Б: \(protein)г Ж: \(fat)г У: \(carbs)г")
      }
    }

    if let activity = daily.activity {
      if let activeEnergy = activity.activeEnergyKcal, activeEnergy > 0 {
        MetricRow(icon: "flame.fill", label: "Активная энергия", value: "\(activeEnergy) ккал")
      }

      if let exerciseMin = activity.exerciseMin, exerciseMin > 0 {
        MetricRow(icon: "figure.run", label: "Упражнения", value: "\(exerciseMin) мин")
      }
    }
  }

  @ViewBuilder
  private func workoutsSection(_ feed: FeedDayResponse) -> some View {
    if let sessions = feed.sessions {
      if let workouts = sessions.workouts, !workouts.isEmpty {
        Section("Тренировки") {
          ForEach(workouts.indices, id: \.self) { idx in
            workoutRow(workouts[idx])
          }
        }
      }
    }
  }

  @ViewBuilder
  private func workoutRow(_ workout: WorkoutSession) -> some View {
    let duration = workout.end.timeIntervalSince(workout.start)
    let minutes = Int(duration / 60)
    let caloriesText = workout.caloriesKcal.map { " • \($0) ккал" } ?? ""

    HStack {
      Image(systemName: workoutIcon(workout.label))
        .foregroundStyle(.blue)
      VStack(alignment: .leading, spacing: 2) {
        Text(workoutLabel(workout.label))
          .font(.subheadline)
        Text("\(minutes) мин" + caloriesText)
          .font(.caption)
          .foregroundStyle(.secondary)
      }
    }
  }

  @ViewBuilder
  private func checkinsSection(_ feed: FeedDayResponse) -> some View {
    Section {
      VStack(alignment: .leading, spacing: 16) {
        morningCheckinContent(feed)
        Divider()
        eveningCheckinContent(feed)
      }
      .padding(.vertical, 8)
    } header: {
      Text("Чекины")
        .font(.headline)
    }
  }

  @ViewBuilder
  private func morningCheckinContent(_ feed: FeedDayResponse) -> some View {
    if isEditingMorning {
      CheckinEditorView(
        type: .morning,
        score: $draftMorningScore,
        note: $draftMorningNote,
        tags: $draftMorningTags,
        newTag: $newMorningTag,
        tagError: $tagError,
        isSaving: isSaving,
        existingCheckin: feed.checkins.morning,
        profileId: ownerProfile?.id,
        onSave: { await saveMorningCheckin() },
        onCancel: { cancelMorningEdit() },
        onDelete: { confirmDelete(.morning) }
      )
    } else {
      if let morning = feed.checkins.morning {
        // Компактный вид для утреннего чек-ина
        HStack(alignment: .center, spacing: 12) {
          Image(systemName: "sun.max.fill")
            .font(.title3)
            .foregroundStyle(.orange)

          VStack(alignment: .leading, spacing: 4) {
            Text("Утро")
              .font(.subheadline)
              .fontWeight(.medium)

            HStack(spacing: 4) {
              ForEach(1...5, id: \.self) { index in
                Image(systemName: index <= morning.score ? "star.fill" : "star")
                  .font(.caption)
                  .foregroundStyle(.yellow)
              }
            }
          }

          Spacer()

          Button {
            startEditingMorning(existing: morning)
          } label: {
            Text("Изменить")
              .font(.caption)
              .foregroundStyle(.secondary)
          }
        }
        .padding(12)
        .background(
          RoundedRectangle(cornerRadius: 10)
            .fill(Color(.secondarySystemGroupedBackground))
        )
      } else {
        // Минимальная кнопка добавления
        Button {
          startEditingMorning(existing: nil)
        } label: {
          HStack(spacing: 8) {
            Image(systemName: "sun.max")
              .foregroundStyle(.orange)
            Text("Утренний чек-ин")
              .font(.subheadline)
            Spacer()
            Image(systemName: "plus.circle")
              .foregroundStyle(.secondary)
          }
          .padding(12)
          .background(
            RoundedRectangle(cornerRadius: 10)
              .strokeBorder(Color(.separator), lineWidth: 1)
          )
        }
        .buttonStyle(.plain)
      }
    }
  }

  @ViewBuilder
  private func eveningCheckinContent(_ feed: FeedDayResponse) -> some View {
    if isEditingEvening {
      CheckinEditorView(
        type: .evening,
        score: $draftEveningScore,
        note: $draftEveningNote,
        tags: $draftEveningTags,
        newTag: $newEveningTag,
        tagError: $tagError,
        isSaving: isSaving,
        existingCheckin: feed.checkins.evening,
        profileId: ownerProfile?.id,
        onSave: { await saveEveningCheckin() },
        onCancel: { cancelEveningEdit() },
        onDelete: { confirmDelete(.evening) }
      )
    } else {
      if let evening = feed.checkins.evening {
        // Полная карточка для вечернего чек-ина
        VStack(alignment: .leading, spacing: 12) {
          HStack(alignment: .center, spacing: 12) {
            Image(systemName: "moon.fill")
              .font(.title2)
              .foregroundStyle(.indigo)

            VStack(alignment: .leading, spacing: 4) {
              Text("Вечер")
                .font(.headline)

              HStack(spacing: 4) {
                ForEach(1...5, id: \.self) { index in
                  Image(systemName: index <= evening.score ? "star.fill" : "star")
                    .font(.subheadline)
                    .foregroundStyle(.yellow)
                }
              }
            }

            Spacer()

            Button {
              startEditingEvening(existing: evening)
            } label: {
              Image(systemName: "square.and.pencil")
                .foregroundStyle(.secondary)
            }
          }

          if let note = evening.note, !note.isEmpty {
            Text(note)
              .font(.subheadline)
              .foregroundStyle(.secondary)
              .lineLimit(3)
          }

          if let tags = evening.tags, !tags.isEmpty {
            FlowLayout(spacing: 6) {
              ForEach(tags, id: \.self) { tag in
                TagChip(tag, color: .indigo)
              }
            }
          }
        }
        .padding(16)
        .background(
          RoundedRectangle(cornerRadius: 12)
            .fill(Color(.secondarySystemGroupedBackground))
        )
      } else {
        // Акцентная кнопка после 18:00
        if shouldHighlightEveningCTA {
          Button {
            startEditingEvening(existing: nil)
          } label: {
            HStack(spacing: 12) {
              Image(systemName: "moon.stars.fill")
                .font(.title3)

              VStack(alignment: .leading, spacing: 4) {
                Text("Вечерний чек-ин")
                  .font(.subheadline)
                  .fontWeight(.semibold)
                Text("Заполните дневник за сегодня")
                  .font(.caption)
              }

              Spacer()

              Image(systemName: "chevron.right")
            }
            .foregroundStyle(.white)
            .padding(16)
            .background(
              RoundedRectangle(cornerRadius: 12)
                .fill(Color.indigo)
            )
          }
          .buttonStyle(.plain)
        } else {
          Button {
            startEditingEvening(existing: nil)
          } label: {
            HStack(spacing: 8) {
              Image(systemName: "moon")
                .foregroundStyle(.indigo)
              Text("Вечерний чек-ин")
                .font(.subheadline)
              Spacer()
              Image(systemName: "plus.circle")
                .foregroundStyle(.secondary)
            }
            .padding(12)
            .background(
              RoundedRectangle(cornerRadius: 10)
                .strokeBorder(Color(.separator), lineWidth: 1)
            )
          }
          .buttonStyle(.plain)
        }
      }
    }
  }

  private var shouldHighlightEveningCTA: Bool {
    let calendar = Calendar.current
    let isToday = calendar.isDateInToday(selectedDate)
    let hour = calendar.component(.hour, from: Date())
    return isToday && hour >= 18
  }

  @ViewBuilder
  private func missingFieldsSection(_ feed: FeedDayResponse) -> some View {
    if !feed.missingFields.isEmpty {
      Section("Недостающие данные") {
        ForEach(feed.missingFields, id: \.self) { field in
          HStack {
            Image(systemName: "exclamationmark.circle")
              .foregroundStyle(.orange)
            Text(missingFieldLabel(field))
              .font(.caption)
          }
        }
      }
    }
  }

  #if DEBUG
    @ViewBuilder
    private var debugSection: some View {
      Section("Debug Info") {
        if let ownerProfile = profiles.first(where: { $0.type == "owner" }) {
          Text("Profile: \(ownerProfile.id.uuidString)")
            .font(.caption2)
            .foregroundStyle(.secondary)
        }
        Text("Date: \(formatDate(selectedDate))")
          .font(.caption2)
          .foregroundStyle(.secondary)
      }
    }
  #endif

  // MARK: - Data Loading

  private func loadData() async {
    isLoading = true
    errorMessage = nil

    do {
      // Проверяем healthz
      let isHealthy = try await APIClient.shared.healthCheck()
      serverStatus = isHealthy ? "OK" : "Недоступен"

      // Загружаем профили
      profiles = try await APIClient.shared.listProfiles()

      // Загружаем feed для owner профиля
      await loadFeedDay()

      // Загружаем unread count
      await loadUnreadCount()

      // Generate notifications for today (best-effort)
      await generateNotificationsForToday()
    } catch {
      if !handleError(error) {
        errorMessage = error.localizedDescription
        serverStatus = "Ошибка"
      }
    }

    isLoading = false
  }

  private func applyFeedNavigationRequestIfNeeded() async {
    guard let request = navigation.feedNavigationRequest else { return }
    selectedDate = request.date
    if let editType = request.editType {
      pendingAutoEditType = CheckinType(rawValue: editType.rawValue)
    } else {
      pendingAutoEditType = nil
    }
    navigation.feedNavigationRequest = nil

    if profiles.isEmpty {
      do {
        profiles = try await APIClient.shared.listProfiles()
      } catch {
        if !handleError(error) {
          errorMessage = "Не удалось загрузить профили: \(error.localizedDescription)"
        }
        return
      }
    }

    await loadFeedDay()
  }

  private func loadUnreadCount() async {
    guard let profile = ownerProfile else { return }

    do {
      unreadCount = try await APIClient.shared.fetchUnreadCount(profileId: profile.id)

      // Update app icon badge
      LocalNotificationScheduler.shared.updateBadge(unreadCount: unreadCount)
    } catch {
      // Silently fail - non-critical
      unreadCount = 0
    }
  }

  private func generateNotificationsForToday() async {
    guard let profile = ownerProfile else { return }

    // Reschedule local notifications after server generation
    defer {
      Task {
        await LocalNotificationScheduler.shared.rescheduleForToday(profileId: profile.id)
      }
    }

    // Only generate for today
    let calendar = Calendar.current
    if !calendar.isDateInToday(selectedDate) {
      return
    }

    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    let dateString = formatter.string(from: selectedDate)

    let thresholds = GenerateThresholds(
      sleepMinMinutes: 420,  // 7 hours
      stepsMin: 6000,
      activeEnergyMinKcal: 200
    )

    do {
      _ = try await APIClient.shared.generateNotifications(
        profileId: profile.id,
        date: dateString,
        timeZone: TimeZone.current.identifier,
        now: Date(),
        thresholds: thresholds
      )
      // Reload unread count after generation
      await loadUnreadCount()
    } catch {
      // Silently fail - generation is best-effort
    }
  }

  private func loadFeedDay() async {
    guard let ownerProfile = profiles.first(where: { $0.type == "owner" }) else {
      errorMessage = "Owner profile not found"
      return
    }

    do {
      feedDay = try await APIClient.shared.fetchFeedDay(
        profileId: ownerProfile.id,
        date: selectedDate
      )
      errorMessage = nil
      applyPendingAutoEditIfNeeded()
      await loadUnreadCount()
    } catch {
      if !handleError(error) {
        errorMessage = "Не удалось загрузить сводку: \(error.localizedDescription)"
      }
    }
  }

  private func applyPendingAutoEditIfNeeded() {
    guard let target = pendingAutoEditType, let feed = feedDay else { return }
    pendingAutoEditType = nil

    switch target {
    case .morning:
      startEditingMorning(existing: feed.checkins.morning)
    case .evening:
      startEditingEvening(existing: feed.checkins.evening)
    }
  }

  // MARK: - Morning Checkin Actions

  private func startEditingMorning(existing: CheckinSummary?) {
    if let existing = existing {
      draftMorningScore = existing.score
      draftMorningNote = existing.note ?? ""
      draftMorningTags = existing.tags ?? []
    } else {
      draftMorningScore = 3
      draftMorningNote = ""
      draftMorningTags = []
    }
    newMorningTag = ""
    tagError = nil
    isEditingMorning = true
  }

  private func cancelMorningEdit() {
    isEditingMorning = false
    draftMorningScore = 3
    draftMorningNote = ""
    draftMorningTags = []
    newMorningTag = ""
    tagError = nil
  }

  private func saveMorningCheckin() async {
    guard let ownerProfile = profiles.first(where: { $0.type == "owner" }) else { return }

    isSaving = true
    errorMessage = nil

    do {
      let request = UpsertCheckinRequest(
        profileId: ownerProfile.id,
        date: formatDate(selectedDate),
        type: .morning,
        score: draftMorningScore,
        tags: draftMorningTags.isEmpty ? nil : draftMorningTags,
        note: draftMorningNote.isEmpty ? nil : draftMorningNote
      )

      _ = try await APIClient.shared.upsertCheckin(request)

      // Reload feed to get updated data
      await loadFeedDay()

      // Close editor
      isEditingMorning = false
      draftMorningScore = 3
      draftMorningNote = ""
      draftMorningTags = []
      newMorningTag = ""
    } catch {
      if !handleError(error) {
        errorMessage = "Не удалось сохранить чек-ин: \(error.localizedDescription)"
      }
    }

    isSaving = false
  }

  // MARK: - Evening Checkin Actions

  private func startEditingEvening(existing: CheckinSummary?) {
    if let existing = existing {
      draftEveningScore = existing.score
      draftEveningNote = existing.note ?? ""
      draftEveningTags = existing.tags ?? []
    } else {
      draftEveningScore = 3
      draftEveningNote = ""
      draftEveningTags = []
    }
    newEveningTag = ""
    tagError = nil
    isEditingEvening = true
  }

  private func cancelEveningEdit() {
    isEditingEvening = false
    draftEveningScore = 3
    draftEveningNote = ""
    draftEveningTags = []
    newEveningTag = ""
    tagError = nil
  }

  private func saveEveningCheckin() async {
    guard let ownerProfile = profiles.first(where: { $0.type == "owner" }) else { return }

    isSaving = true
    errorMessage = nil

    do {
      let request = UpsertCheckinRequest(
        profileId: ownerProfile.id,
        date: formatDate(selectedDate),
        type: .evening,
        score: draftEveningScore,
        tags: draftEveningTags.isEmpty ? nil : draftEveningTags,
        note: draftEveningNote.isEmpty ? nil : draftEveningNote
      )

      _ = try await APIClient.shared.upsertCheckin(request)

      // Reload feed to get updated data
      await loadFeedDay()

      // Close editor
      isEditingEvening = false
      draftEveningScore = 3
      draftEveningNote = ""
      draftEveningTags = []
      newEveningTag = ""
    } catch {
      if !handleError(error) {
        errorMessage = "Не удалось сохранить чек-ин: \(error.localizedDescription)"
      }
    }

    isSaving = false
  }

  // MARK: - Delete Checkin

  private func confirmDelete(_ type: CheckinType) {
    deleteTarget = type
    showDeleteConfirmation = true
  }

  private func deleteCheckin(type: CheckinType) async {
    guard let feed = feedDay else { return }

    let checkinId: UUID?
    switch type {
    case .morning:
      checkinId = feed.checkins.morning?.id
    case .evening:
      checkinId = feed.checkins.evening?.id
    }

    guard let id = checkinId else { return }

    isSaving = true
    errorMessage = nil

    do {
      try await APIClient.shared.deleteCheckin(id: id)

      // Reload feed
      await loadFeedDay()

      // Close editor if it was open
      if type == .morning {
        isEditingMorning = false
      } else {
        isEditingEvening = false
      }
    } catch {
      if !handleError(error) {
        errorMessage = "Не удалось удалить чек-ин: \(error.localizedDescription)"
      }
    }

    isSaving = false
  }

  // MARK: - Helpers

  private func formatDate(_ date: Date) -> String {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withFullDate]
    return formatter.string(from: date)
  }

  private func missingFieldLabel(_ field: String) -> String {
    switch field {
    case "daily": return "Дневные метрики"
    case "morning_checkin": return "Утренний чекин"
    case "evening_checkin": return "Вечерний чекин"
    case "weight": return "Вес"
    case "resting_hr": return "Пульс покоя"
    default: return field
    }
  }

  // MARK: - Share Image Generation

  @MainActor
  private func generateShareImage() async {
    guard let feed = feedDay else { return }

    // Prepare data for share card
    let metrics: DaySummaryShareCard.MetricsData?
    if let daily = feed.daily {
      metrics = DaySummaryShareCard.MetricsData(
        steps: daily.activity?.steps,
        weight: daily.body?.weightKgLast,
        restingHR: daily.heart?.restingHrBpm,
        sleepMinutes: daily.sleep?.totalMinutes,
        activeEnergyKcal: daily.activity?.activeEnergyKcal
      )
    } else {
      metrics = nil
    }

    let morning: DaySummaryShareCard.CheckinData?
    if let m = feed.checkins.morning {
      morning = DaySummaryShareCard.CheckinData(
        score: m.score,
        tags: m.tags ?? [],
        note: m.note
      )
    } else {
      morning = nil
    }

    let evening: DaySummaryShareCard.CheckinData?
    if let e = feed.checkins.evening {
      evening = DaySummaryShareCard.CheckinData(
        score: e.score,
        tags: e.tags ?? [],
        note: e.note
      )
    } else {
      evening = nil
    }

    // Create share card view
    let shareCard = DaySummaryShareCard(
      date: feed.date,
      metrics: metrics,
      morning: morning,
      evening: evening
    )

    // Render to UIImage
    let renderer = ImageRenderer(content: shareCard)
    renderer.scale = 3.0  // Retina quality

    if let uiImage = renderer.uiImage {
      shareImage = uiImage
      showShareSheet = true
    }
  }

  // MARK: - Workout Helpers

  private func workoutIcon(_ label: String) -> String {
    switch label {
    case "run": return "figure.run"
    case "walk": return "figure.walk"
    case "strength": return "dumbbell.fill"
    case "core": return "figure.core.training"
    case "cycle": return "bicycle"
    case "swim": return "figure.pool.swim"
    case "yoga": return "figure.yoga"
    case "hike": return "figure.hiking"
    default: return "figure.mixed.cardio"
    }
  }

  private func workoutLabel(_ label: String) -> String {
    switch label {
    case "run": return "Бег"
    case "walk": return "Ходьба"
    case "strength": return "Силовая"
    case "core": return "Корпус"
    case "cycle": return "Велосипед"
    case "swim": return "Плавание"
    case "yoga": return "Йога"
    case "hike": return "Поход"
    default: return "Тренировка"
    }
  }
}

// MARK: - Checkin Editor View

private struct CheckinEditorView: View {
  let type: CheckinType
  @Binding var score: Int
  @Binding var note: String
  @Binding var tags: [String]
  @Binding var newTag: String
  @Binding var tagError: String?
  let isSaving: Bool
  let existingCheckin: CheckinSummary?
  let profileId: UUID?
  let onSave: () async -> Void
  let onCancel: () -> Void
  let onDelete: () -> Void

  private let maxTags = 10
  private let maxNoteLength = 500

  var body: some View {
    VStack(alignment: .leading, spacing: 12) {
      // Header
      Text(type == .morning ? "Утренний чек-ин" : "Вечерний чек-ин")
        .font(.headline)

      // Score selector
      VStack(alignment: .leading, spacing: 8) {
        Text("Как вы себя чувствуете?")
          .font(.subheadline)
          .foregroundStyle(.secondary)

        HStack(spacing: 12) {
          ForEach(1...5, id: \.self) { value in
            Button {
              score = value
            } label: {
              VStack(spacing: 4) {
                Image(systemName: score == value ? "star.fill" : "star")
                  .font(.title2)
                  .foregroundStyle(scoreColor(for: value))

                Text("\(value)")
                  .font(.caption2)
                  .foregroundStyle(.secondary)
              }
              .frame(maxWidth: .infinity)
              .padding(.vertical, 8)
              .background(
                RoundedRectangle(cornerRadius: 8)
                  .fill(score == value ? Color.blue.opacity(0.1) : Color.clear)
              )
              .overlay(
                RoundedRectangle(cornerRadius: 8)
                  .stroke(score == value ? Color.blue : Color.gray.opacity(0.3), lineWidth: 1)
              )
            }
            .buttonStyle(.plain)
          }
        }

        HStack {
          Text("1 = плохо")
            .font(.caption2)
            .foregroundStyle(.red)
          Spacer()
          Text("5 = отлично")
            .font(.caption2)
            .foregroundStyle(.blue)
        }
      }

      // Tags
      VStack(alignment: .leading, spacing: 8) {
        Text("Теги")
          .font(.subheadline)
          .foregroundStyle(.secondary)

        // Existing tags
        if !tags.isEmpty {
          FlowLayout(spacing: 6) {
            ForEach(tags, id: \.self) { tag in
              FeedTagChip(text: tag) {
                tags.removeAll { $0 == tag }
              }
            }
          }
        }

        // Add tag field
        if tags.count < maxTags {
          HStack {
            TextField("Добавить тег", text: $newTag)
              .textFieldStyle(.roundedBorder)
              .autocapitalization(.none)
              .disabled(isSaving)

            Button("Добавить") {
              addTag()
            }
            .disabled(newTag.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || isSaving)
            .buttonStyle(.bordered)
          }
        } else {
          Text("Максимум \(maxTags) тегов")
            .font(.caption)
            .foregroundStyle(.orange)
        }

        if let error = tagError {
          Text(error)
            .font(.caption)
            .foregroundStyle(.red)
        }
      }

      // Note
      VStack(alignment: .leading, spacing: 8) {
        HStack {
          Text("Заметка")
            .font(.subheadline)
            .foregroundStyle(.secondary)
          Spacer()
          Text("\(note.count)/\(maxNoteLength)")
            .font(.caption2)
            .foregroundStyle(note.count > maxNoteLength ? .red : .secondary)
        }

        TextEditor(text: $note)
          .frame(height: 80)
          .overlay(
            RoundedRectangle(cornerRadius: 8)
              .stroke(Color.gray.opacity(0.3), lineWidth: 1)
          )
          .disabled(isSaving)
          .onChange(of: note) { _, newValue in
            if newValue.count > maxNoteLength {
              note = String(newValue.prefix(maxNoteLength))
            }
          }
      }

      // Attachments section
      if let profileId = profileId {
        CheckinAttachmentsView(
          profileId: profileId,
          checkinId: existingCheckin?.id,
          maxCount: 4
        )
      }

      // Action buttons
      HStack(spacing: 12) {
        if existingCheckin != nil {
          Button("Удалить", role: .destructive) {
            onDelete()
          }
          .disabled(isSaving)
        }

        Spacer()

        Button("Отмена") {
          onCancel()
        }
        .disabled(isSaving)

        Button {
          Task {
            await onSave()
          }
        } label: {
          if isSaving {
            ProgressView()
              .controlSize(.small)
          } else {
            Text("Сохранить")
          }
        }
        .buttonStyle(.borderedProminent)
        .disabled(isSaving)
      }
    }
    .padding(.vertical, 8)
  }

  private func addTag() {
    let trimmed = newTag.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmed.isEmpty else { return }

    if tags.count >= maxTags {
      tagError = "Максимум \(maxTags) тегов"
      return
    }

    if tags.contains(trimmed) {
      tagError = "Тег уже добавлен"
      return
    }

    tags.append(trimmed)
    newTag = ""
    tagError = nil
  }

  private func scoreColor(for value: Int) -> Color {
    switch value {
    case 1...2: return .red
    case 3: return .orange
    case 4: return .green
    case 5: return .blue
    default: return .gray
    }
  }
}

// MARK: - Tag Chip

private struct FeedTagChip: View {
  let text: String
  let onDelete: () -> Void

  var body: some View {
    HStack(spacing: 4) {
      Text(text)
        .font(.caption)
      Button {
        onDelete()
      } label: {
        Image(systemName: "xmark.circle.fill")
          .font(.caption)
          .foregroundStyle(.secondary)
      }
      .buttonStyle(.plain)
    }
    .padding(.horizontal, 8)
    .padding(.vertical, 4)
    .background(Color.orange.opacity(0.2))
    .cornerRadius(12)
  }
}

// MARK: - Flow Layout (simple implementation)

private struct FlowLayout: Layout {
  var spacing: CGFloat = 8

  func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
    let result = FlowResult(
      in: proposal.replacingUnspecifiedDimensions().width,
      subviews: subviews,
      spacing: spacing
    )
    return result.size
  }

  func placeSubviews(
    in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()
  ) {
    let result = FlowResult(
      in: bounds.width,
      subviews: subviews,
      spacing: spacing
    )
    for (index, subview) in subviews.enumerated() {
      subview.place(
        at: CGPoint(
          x: bounds.minX + result.frames[index].minX,
          y: bounds.minY + result.frames[index].minY),
        proposal: .unspecified)
    }
  }

  struct FlowResult {
    var frames: [CGRect] = []
    var size: CGSize = .zero

    init(in maxWidth: CGFloat, subviews: Subviews, spacing: CGFloat) {
      var currentX: CGFloat = 0
      var currentY: CGFloat = 0
      var lineHeight: CGFloat = 0

      for subview in subviews {
        let size = subview.sizeThatFits(.unspecified)

        if currentX + size.width > maxWidth && currentX > 0 {
          currentX = 0
          currentY += lineHeight + spacing
          lineHeight = 0
        }

        frames.append(CGRect(x: currentX, y: currentY, width: size.width, height: size.height))
        lineHeight = max(lineHeight, size.height)
        currentX += size.width + spacing
      }

      self.size = CGSize(width: maxWidth, height: currentY + lineHeight)
    }
  }
}

// MARK: - Supporting Views

struct MetricRow: View {
  let icon: String
  let label: String
  let value: String

  var body: some View {
    HStack {
      Image(systemName: icon)
        .foregroundStyle(.blue)
        .frame(width: 24)
      Text(label)
      Spacer()
      Text(value)
        .foregroundStyle(.secondary)
    }
  }
}

struct CheckinCard: View {
  let type: String
  let checkin: CheckinSummary

  var body: some View {
    VStack(alignment: .leading, spacing: 4) {
      HStack {
        Text(type)
          .font(.caption)
          .foregroundStyle(.secondary)
        Spacer()
        ScoreView(score: checkin.score)
      }

      if let tags = checkin.tags, !tags.isEmpty {
        FlowLayout(spacing: 6) {
          ForEach(tags, id: \.self) { tag in
            Text(tag)
              .font(.caption2)
              .padding(.horizontal, 6)
              .padding(.vertical, 2)
              .background(Color.orange.opacity(0.2))
              .cornerRadius(4)
          }
        }
      }

      if let note = checkin.note, !note.isEmpty {
        Text(note)
          .font(.caption)
          .foregroundStyle(.secondary)
          .lineLimit(2)
      }
    }
    .padding(.vertical, 4)
  }
}

struct ScoreView: View {
  let score: Int

  var body: some View {
    HStack(spacing: 2) {
      ForEach(1...5, id: \.self) { index in
        Image(systemName: index <= score ? "star.fill" : "star")
          .foregroundStyle(scoreColor)
          .font(.caption)
      }
    }
  }

  private var scoreColor: Color {
    switch score {
    case 1...2: return .red
    case 3: return .orange
    case 4: return .green
    case 5: return .blue
    default: return .gray
    }
  }
}

// MARK: - Share Sheet Wrapper

struct ShareSheet: UIViewControllerRepresentable {
  let activityItems: [Any]

  func makeUIViewController(context: Context) -> UIActivityViewController {
    let controller = UIActivityViewController(
      activityItems: activityItems,
      applicationActivities: nil
    )
    return controller
  }

  func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}

struct FeedView_Previews: PreviewProvider {
  static var previews: some View {
    FeedView()
      .environmentObject(AppNavigationState())
  }
}
