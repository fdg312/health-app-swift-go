//
//  IntakesView.swift
//  HealthHub
//

import SwiftUI

struct IntakesView: View {
  let profileId: UUID
  @ObservedObject private var auth = AuthManager.shared

  @State private var supplements: [SupplementDTO] = []
  @State private var schedules: [ScheduleDTO] = []
  @State private var dailyIntakes: IntakesDailyResponse?
  @State private var isLoading = false
  @State private var errorMessage: String?

  @State private var showCreateSupplement = false
  @State private var showAddWaterSheet = false
  @State private var showAddScheduleSheet = false
  @State private var showRateLimited = false

  @State private var inFlightScheduleIDs = Set<UUID>()

  private var todayString: String {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    return formatter.string(from: Date())
  }

  private var supplementNameByID: [UUID: String] {
    Dictionary(uniqueKeysWithValues: supplements.map { ($0.id, $0.name) })
  }

  private var todaysSchedules: [ScheduleDTO] {
    schedules
      .filter { $0.isEnabled && isTodayIncluded(daysMask: $0.daysMask) }
      .sorted { $0.timeMinutes < $1.timeMinutes }
  }

  var body: some View {
    List {
      if isLoading && dailyIntakes == nil && supplements.isEmpty && schedules.isEmpty {
        Section {
          HStack {
            Spacer()
            ProgressView("Загрузка...")
            Spacer()
          }
        }
      } else {
        waterSection
        todayScheduleSection
        supplementsSection
        schedulesSection

        if let error = errorMessage {
          Section {
            Text(error)
              .font(.caption)
              .foregroundStyle(.secondary)
          }
        }
      }
    }
    .task {
      await loadData()
    }
    .refreshable {
      await loadData(showLoader: false)
    }
    .sheet(isPresented: $showCreateSupplement) {
      CreateSupplementSheet(
        profileId: profileId,
        onCreated: {
          Task { await loadData(showLoader: false) }
        })
    }
    .sheet(isPresented: $showAddWaterSheet) {
      AddWaterSheet(
        profileId: profileId,
        onAdded: {
          Task { await loadData(showLoader: false) }
        })
    }
    .sheet(isPresented: $showAddScheduleSheet) {
      AddScheduleSheet(
        profileId: profileId,
        supplements: supplements,
        existingCount: schedules.count,
        onCreated: {
          Task { await loadData(showLoader: false) }
        })
    }
    .alert("Слишком много запросов", isPresented: $showRateLimited) {
      Button("OK", role: .cancel) {}
    } message: {
      Text("Попробуйте позже")
    }
  }

  // MARK: - Sections

  private var waterSection: some View {
    Section("Вода") {
      HStack {
        Label("За сегодня", systemImage: "drop.fill")
          .foregroundStyle(.blue)
        Spacer()
        Text("\(dailyIntakes?.waterTotalMl ?? 0) мл")
          .font(.headline)
          .foregroundStyle(.blue)
      }

      HStack(spacing: 10) {
        Button("+250 мл") {
          Task { await addWater(250) }
        }
        .buttonStyle(.borderedProminent)

        Button("Другое...") {
          showAddWaterSheet = true
        }
        .buttonStyle(.bordered)
      }
    }
  }

  private var todayScheduleSection: some View {
    Section("Сегодня по расписанию") {
      if todaysSchedules.isEmpty {
        Text("Нет данных")
          .foregroundStyle(.secondary)
      } else {
        ForEach(todaysSchedules) { schedule in
          let status = dailyStatus(for: schedule.supplementId)
          HStack(alignment: .center, spacing: 12) {
            VStack(alignment: .leading, spacing: 4) {
              Text(supplementNameByID[schedule.supplementId] ?? "Добавка")
                .font(.subheadline)
              Text(timeLabel(schedule.timeMinutes))
                .font(.caption)
                .foregroundStyle(.secondary)
            }

            Spacer()

            if status == "taken" {
              Text("Принято")
                .font(.caption)
                .foregroundStyle(.green)
            } else if status == "skipped" {
              Text("Пропущено")
                .font(.caption)
                .foregroundStyle(.orange)
            } else {
              Button("Отметить принял") {
                Task { await markScheduleTaken(schedule) }
              }
              .buttonStyle(.bordered)
              .disabled(inFlightScheduleIDs.contains(schedule.id))
            }
          }
        }
      }
    }
  }

  private var supplementsSection: some View {
    Section {
      if supplements.isEmpty {
        Text("Нет данных")
          .foregroundStyle(.secondary)
      } else {
        ForEach(supplements) { supplement in
          supplementRow(supplement)
        }
      }
    } header: {
      HStack {
        Text("Добавки")
        Spacer()
        Button {
          showCreateSupplement = true
        } label: {
          Image(systemName: "plus.circle.fill")
            .foregroundStyle(.green)
        }
      }
    }
  }

  private var schedulesSection: some View {
    Section {
      if schedules.isEmpty {
        Text("Нет данных")
          .foregroundStyle(.secondary)
      } else {
        ForEach(schedules) { schedule in
          scheduleRow(schedule)
            .swipeActions(edge: .trailing, allowsFullSwipe: true) {
              Button(role: .destructive) {
                Task { await deleteSchedule(schedule) }
              } label: {
                Label("Удалить", systemImage: "trash")
              }
            }
        }
      }

      if schedules.count >= 20 {
        Text("Достигнут лимит: максимум 20 расписаний")
          .font(.caption)
          .foregroundStyle(.secondary)
      }
    } header: {
      HStack {
        Text("Расписание")
        Spacer()
        Button {
          showAddScheduleSheet = true
        } label: {
          Image(systemName: "plus.circle.fill")
            .foregroundStyle(.green)
        }
        .disabled(schedules.count >= 20 || supplements.isEmpty)
      }
    }
  }

  private func supplementRow(_ supplement: SupplementDTO) -> some View {
    let status = dailyStatus(for: supplement.id)

    return HStack {
      VStack(alignment: .leading, spacing: 4) {
        Text(supplement.name)
          .font(.body)
        if let notes = supplement.notes, !notes.isEmpty {
          Text(notes)
            .font(.caption)
            .foregroundStyle(.secondary)
        }

        if !supplement.components.isEmpty {
          Text(componentSummary(supplement.components))
            .font(.caption2)
            .foregroundStyle(.secondary)
        }
      }

      Spacer()

      Picker(
        "Статус",
        selection: Binding(
          get: { status },
          set: { newStatus in
            Task {
              await updateSupplementStatus(supplement.id, status: newStatus)
            }
          }
        )
      ) {
        Text("Нет").tag("none")
        Text("Принял").tag("taken")
        Text("Пропустил").tag("skipped")
      }
      .pickerStyle(.menu)
    }
    .padding(.vertical, 4)
  }

  private func scheduleRow(_ schedule: ScheduleDTO) -> some View {
    HStack(spacing: 12) {
      VStack(alignment: .leading, spacing: 4) {
        Text(supplementNameByID[schedule.supplementId] ?? "Добавка")
          .font(.body)
        Text("\(timeLabel(schedule.timeMinutes)) · \(daysLabel(schedule.daysMask))")
          .font(.caption)
          .foregroundStyle(.secondary)
      }

      Spacer()

      Toggle(
        "",
        isOn: Binding(
          get: { schedule.isEnabled },
          set: { newValue in
            Task { await updateScheduleEnabled(schedule, isEnabled: newValue) }
          }
        )
      )
      .labelsHidden()
      .disabled(inFlightScheduleIDs.contains(schedule.id))
    }
  }

  // MARK: - Actions

  private func loadData(showLoader: Bool = true) async {
    if showLoader {
      isLoading = true
    }
    errorMessage = nil
    defer {
      if showLoader {
        isLoading = false
      }
    }

    do {
      async let supplementsTask = APIClient.shared.listSupplements(profileId: profileId)
      async let dailyTask = APIClient.shared.fetchIntakesDaily(profileId: profileId, date: todayString)
      async let schedulesTask = APIClient.shared.listSupplementSchedules(profileId: profileId)

      supplements = try await supplementsTask
      dailyIntakes = try await dailyTask
      schedules = try await schedulesTask
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка загрузки приема (\(errorCode(error)))"
      }
    }
  }

  private func addWater(_ amountMl: Int) async {
    do {
      try await APIClient.shared.addWater(profileId: profileId, takenAt: Date(), amountMl: amountMl)

      do {
        try await HealthKitManager.shared.writeWater(amountMl: amountMl)
      } catch {
        print("Failed to write water to HealthKit: \(error)")
      }

      await loadData(showLoader: false)
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка добавления воды (\(errorCode(error)))"
      }
    }
  }

  private func updateSupplementStatus(_ supplementId: UUID, status: String) async {
    guard status != "none" else { return }

    do {
      try await APIClient.shared.upsertSupplementIntake(
        profileId: profileId,
        supplementId: supplementId,
        date: todayString,
        status: status
      )
      await loadData(showLoader: false)
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка обновления приема (\(errorCode(error)))"
      }
    }
  }

  private func markScheduleTaken(_ schedule: ScheduleDTO) async {
    if inFlightScheduleIDs.contains(schedule.id) { return }
    inFlightScheduleIDs.insert(schedule.id)
    defer { inFlightScheduleIDs.remove(schedule.id) }

    do {
      try await APIClient.shared.upsertSupplementIntake(
        profileId: profileId,
        supplementId: schedule.supplementId,
        date: todayString,
        status: "taken"
      )
      await loadData(showLoader: false)
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка отметки приема (\(errorCode(error)))"
      }
    }
  }

  private func updateScheduleEnabled(_ schedule: ScheduleDTO, isEnabled: Bool) async {
    if inFlightScheduleIDs.contains(schedule.id) { return }
    inFlightScheduleIDs.insert(schedule.id)
    defer { inFlightScheduleIDs.remove(schedule.id) }

    do {
      _ = try await APIClient.shared.upsertSupplementSchedule(
        UpsertScheduleRequest(
          profileId: profileId,
          supplementId: schedule.supplementId,
          timeMinutes: schedule.timeMinutes,
          daysMask: schedule.daysMask,
          isEnabled: isEnabled
        )
      )
      await loadData(showLoader: false)
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка обновления расписания (\(errorCode(error)))"
      }
    }
  }

  private func deleteSchedule(_ schedule: ScheduleDTO) async {
    if inFlightScheduleIDs.contains(schedule.id) { return }
    inFlightScheduleIDs.insert(schedule.id)
    defer { inFlightScheduleIDs.remove(schedule.id) }

    do {
      try await APIClient.shared.deleteSupplementSchedule(id: schedule.id)
      await loadData(showLoader: false)
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка удаления расписания (\(errorCode(error)))"
      }
    }
  }

  // MARK: - Helpers

  private func dailyStatus(for supplementId: UUID) -> String {
    dailyIntakes?.supplements.first(where: { $0.supplementId == supplementId })?.status ?? "none"
  }

  private func isTodayIncluded(daysMask: Int) -> Bool {
    let weekday = Calendar.current.component(.weekday, from: Date())
    let bit = weekdayToMaskBit(weekday)
    return (daysMask & (1 << bit)) != 0
  }

  private func weekdayToMaskBit(_ weekday: Int) -> Int {
    // Calendar weekday: 1=Sunday, 2=Monday ... 7=Saturday
    if weekday == 1 { return 6 }
    return weekday - 2
  }

  private func timeLabel(_ minutes: Int) -> String {
    let h = max(0, min(23, minutes / 60))
    let m = max(0, min(59, minutes % 60))
    return String(format: "%02d:%02d", h, m)
  }

  private func daysLabel(_ mask: Int) -> String {
    if mask == 127 { return "Ежедневно" }
    let labels = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"]
    var selected: [String] = []
    for i in 0..<7 where (mask & (1 << i)) != 0 {
      selected.append(labels[i])
    }
    return selected.isEmpty ? "—" : selected.joined(separator: ", ")
  }

  private func componentSummary(_ components: [SupplementComponentDTO]) -> String {
    components
      .map { "\($0.nutrientKey): \($0.amount.cleanString) \($0.unit)" }
      .joined(separator: ", ")
  }

  private func handleError(_ error: Error) -> Bool {
    if let apiError = error as? APIError, apiError == .unauthorized {
      auth.handleUnauthorized()
      return true
    }
    if let apiError = error as? APIError, apiError == .rateLimited {
      showRateLimited = true
      return true
    }
    return false
  }

  private func errorCode(_ error: Error) -> String {
    if let apiError = error as? APIError {
      return apiError.uiCode
    }
    return "bad_response"
  }
}

// MARK: - Add Schedule Sheet

struct AddScheduleSheet: View {
  let profileId: UUID
  let supplements: [SupplementDTO]
  let existingCount: Int
  let onCreated: () -> Void

  @Environment(\.dismiss) private var dismiss

  @State private var selectedSupplementID: UUID?
  @State private var selectedTime: Date
  @State private var selectedDays: Set<Int> = Set(0..<7)
  @State private var isEnabled = true
  @State private var isSaving = false
  @State private var errorText: String?

  init(profileId: UUID, supplements: [SupplementDTO], existingCount: Int, onCreated: @escaping () -> Void) {
    self.profileId = profileId
    self.supplements = supplements
    self.existingCount = existingCount
    self.onCreated = onCreated

    let calendar = Calendar.current
    let defaultDate = calendar.date(bySettingHour: 12, minute: 0, second: 0, of: Date()) ?? Date()
    _selectedTime = State(initialValue: defaultDate)
    _selectedSupplementID = State(initialValue: supplements.first?.id)
  }

  var body: some View {
    NavigationStack {
      Form {
        if let errorText {
          Section {
            Text(errorText)
              .font(.caption)
              .foregroundStyle(.red)
          }
        }

        Section("Добавка") {
          if supplements.isEmpty {
            Text("Сначала добавьте хотя бы одну добавку")
              .foregroundStyle(.secondary)
          } else {
            Picker("Добавка", selection: Binding(get: {
              selectedSupplementID ?? supplements.first?.id ?? UUID()
            }, set: { newValue in
              selectedSupplementID = newValue
            })) {
              ForEach(supplements) { supplement in
                Text(supplement.name).tag(supplement.id)
              }
            }
          }
        }

        Section("Время") {
          DatePicker("", selection: $selectedTime, displayedComponents: .hourAndMinute)
            .labelsHidden()
        }

        Section("Дни недели") {
          let labels = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"]
          ForEach(Array(labels.enumerated()), id: \.offset) { idx, label in
            Toggle(isOn: Binding(get: {
              selectedDays.contains(idx)
            }, set: { isOn in
              if isOn {
                selectedDays.insert(idx)
              } else {
                selectedDays.remove(idx)
              }
            })) {
              Text(label)
            }
          }
        }

        Section {
          Toggle("Включено", isOn: $isEnabled)
        }
      }
      .navigationTitle("Новое расписание")
      .navigationBarTitleDisplayMode(.inline)
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("Отмена") { dismiss() }
        }

        ToolbarItem(placement: .confirmationAction) {
          Button("Сохранить") {
            Task { await save() }
          }
          .disabled(isSaving || supplements.isEmpty)
        }
      }
    }
  }

  private func save() async {
    guard existingCount < 20 else {
      errorText = "Максимум 20 расписаний"
      return
    }
    guard let supplementID = selectedSupplementID else {
      errorText = "Выберите добавку"
      return
    }
    guard !selectedDays.isEmpty else {
      errorText = "Выберите хотя бы один день"
      return
    }

    let components = Calendar.current.dateComponents([.hour, .minute], from: selectedTime)
    let hour = components.hour ?? 0
    let minute = components.minute ?? 0
    let timeMinutes = max(0, min(1439, hour * 60 + minute))

    var mask = 0
    for dayBit in selectedDays {
      mask |= (1 << dayBit)
    }

    isSaving = true
    errorText = nil

    do {
      _ = try await APIClient.shared.upsertSupplementSchedule(
        UpsertScheduleRequest(
          profileId: profileId,
          supplementId: supplementID,
          timeMinutes: timeMinutes,
          daysMask: mask,
          isEnabled: isEnabled
        )
      )
      onCreated()
      dismiss()
    } catch {
      if let apiError = error as? APIError {
        errorText = "Ошибка сохранения (\(apiError.uiCode))"
      } else {
        errorText = "Ошибка сохранения (bad_response)"
      }
    }

    isSaving = false
  }
}

// MARK: - Create Supplement Sheet

struct CreateSupplementSheet: View {
  let profileId: UUID
  let onCreated: () -> Void

  @Environment(\.dismiss) private var dismiss
  @State private var name = ""
  @State private var notes = ""
  @State private var isSaving = false
  @State private var components: [ComponentDraft] = []
  @State private var errorText: String?

  var body: some View {
    NavigationStack {
      Form {
        if let errorText {
          Section {
            Text(errorText)
              .font(.caption)
              .foregroundStyle(.red)
          }
        }

        Section("Название") {
          TextField("Например: Витамин D3 или Омега-комплекс", text: $name)
        }

        Section("Компоненты (опционально)") {
          if components.isEmpty {
            Text("Можно оставить пустым или добавить витамины/минералы для синхронизации с HealthKit")
              .font(.caption)
              .foregroundStyle(.secondary)
          }

          ForEach($components) { $component in
            VStack(alignment: .leading, spacing: 8) {
              Picker("Витамин/компонент", selection: $component.preset) {
                ForEach(ComponentPreset.allCases) { preset in
                  Text(preset.title).tag(preset)
                }
              }

              HStack {
                TextField("Количество", value: $component.amount, format: .number)
                  .keyboardType(.decimalPad)

                TextField("Ед.", text: $component.unit)
                  .textInputAutocapitalization(.never)
                  .frame(width: 70)
              }
            }
          }
          .onDelete { indexSet in
            components.remove(atOffsets: indexSet)
          }

          Button {
            components.append(ComponentDraft())
          } label: {
            Label("Добавить компонент", systemImage: "plus")
          }
        }

        Section("Заметки (опционально)") {
          TextEditor(text: $notes)
            .frame(height: 100)
        }
      }
      .navigationTitle("Новая добавка")
      .navigationBarTitleDisplayMode(.inline)
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("Отмена") {
            dismiss()
          }
        }

        ToolbarItem(placement: .confirmationAction) {
          Button("Сохранить") {
            Task {
              await save()
            }
          }
          .disabled(name.isEmpty || isSaving)
        }
      }
    }
  }

  private func save() async {
    let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !trimmedName.isEmpty else {
      errorText = "Введите название"
      return
    }

    let requestComponents = components.compactMap { $0.toRequest() }

    isSaving = true
    errorText = nil

    do {
      _ = try await APIClient.shared.createSupplement(
        profileId: profileId,
        name: trimmedName,
        notes: notes.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : notes,
        components: requestComponents
      )
      onCreated()
      dismiss()
    } catch {
      if let apiError = error as? APIError {
        errorText = "Ошибка создания (\(apiError.uiCode))"
      } else {
        errorText = "Ошибка создания (bad_response)"
      }
    }

    isSaving = false
  }
}

private struct ComponentDraft: Identifiable {
  let id = UUID()
  var preset: ComponentPreset = .vitaminD
  var amount: Double = 1
  var unit: String = "mg"

  init() {
    self.preset = .vitaminD
    self.amount = 1
    self.unit = ComponentPreset.vitaminD.defaultUnit
  }

  func toRequest() -> ComponentInput? {
    let normalizedUnit = unit.trimmingCharacters(in: .whitespacesAndNewlines)
    guard amount > 0, !normalizedUnit.isEmpty else { return nil }
    return ComponentInput(
      nutrientKey: preset.nutrientKey,
      hkIdentifier: preset.hkIdentifier,
      amount: amount,
      unit: normalizedUnit
    )
  }
}

private enum ComponentPreset: String, CaseIterable, Identifiable {
  case vitaminD
  case vitaminC
  case vitaminB12
  case magnesium
  case omega3
  case zinc

  var id: String { rawValue }

  var title: String {
    switch self {
    case .vitaminD: return "Витамин D"
    case .vitaminC: return "Витамин C"
    case .vitaminB12: return "Витамин B12"
    case .magnesium: return "Магний"
    case .omega3: return "Омега‑3"
    case .zinc: return "Цинк"
    }
  }

  var nutrientKey: String {
    switch self {
    case .vitaminD: return "vitamin_d"
    case .vitaminC: return "vitamin_c"
    case .vitaminB12: return "vitamin_b12"
    case .magnesium: return "magnesium"
    case .omega3: return "omega_3"
    case .zinc: return "zinc"
    }
  }

  var hkIdentifier: String? {
    switch self {
    case .vitaminD: return "HKQuantityTypeIdentifierDietaryVitaminD"
    case .vitaminC: return "HKQuantityTypeIdentifierDietaryVitaminC"
    case .vitaminB12: return "HKQuantityTypeIdentifierDietaryVitaminB12"
    case .magnesium: return "HKQuantityTypeIdentifierDietaryMagnesium"
    case .omega3: return nil
    case .zinc: return "HKQuantityTypeIdentifierDietaryZinc"
    }
  }

  var defaultUnit: String {
    switch self {
    case .vitaminD: return "mcg"
    case .vitaminC: return "mg"
    case .vitaminB12: return "mcg"
    case .magnesium: return "mg"
    case .omega3: return "mg"
    case .zinc: return "mg"
    }
  }
}

private extension Double {
  var cleanString: String {
    if truncatingRemainder(dividingBy: 1) == 0 {
      return String(Int(self))
    }
    return String(format: "%.2f", self)
  }
}


// MARK: - Add Water Sheet

struct AddWaterSheet: View {
  let profileId: UUID
  let onAdded: () -> Void

  @Environment(\.dismiss) private var dismiss
  @State private var amountMl = 250
  @State private var isSaving = false

  var body: some View {
    NavigationStack {
      Form {
        Section("Количество (мл)") {
          Stepper(value: $amountMl, in: 50...2000, step: 50) {
            Text("\(amountMl) мл")
          }
        }
      }
      .navigationTitle("Добавить воду")
      .navigationBarTitleDisplayMode(.inline)
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("Отмена") {
            dismiss()
          }
        }

        ToolbarItem(placement: .confirmationAction) {
          Button("Добавить") {
            Task {
              await save()
            }
          }
          .disabled(isSaving)
        }
      }
    }
  }

  private func save() async {
    isSaving = true

    do {
      try await APIClient.shared.addWater(
        profileId: profileId,
        takenAt: Date(),
        amountMl: amountMl
      )
      onAdded()
      dismiss()
    } catch {
      print("Error adding water: \(error)")
    }

    isSaving = false
  }
}
