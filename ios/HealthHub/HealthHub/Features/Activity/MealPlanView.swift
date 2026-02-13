import SwiftUI

struct MealPlanView: View {
  @ObservedObject private var auth = AuthManager.shared
  let profileId: UUID

  @State private var plan: MealPlanDTO?
  @State private var items: [MealPlanItemDTO] = []
  @State private var isLoading = false
  @State private var errorMessage: String?
  @State private var showEditSheet = false
  @State private var showDeleteConfirm = false
  @State private var showRateLimited = false

  private let mealSlots = ["breakfast", "lunch", "dinner", "snack"]
  private let slotNames: [String: String] = [
    "breakfast": "Завтрак",
    "lunch": "Обед",
    "dinner": "Ужин",
    "snack": "Перекус",
  ]

  var body: some View {
    Group {
      if isLoading && plan == nil {
        ProgressView("Загрузка...")
      } else if plan == nil {
        emptyState
      } else {
        planContent
      }
    }
    .navigationTitle("План питания")
    .toolbar {
      if plan != nil {
        ToolbarItem(placement: .topBarTrailing) {
          Menu {
            Button {
              showEditSheet = true
            } label: {
              Label("Редактировать", systemImage: "pencil")
            }

            Button(role: .destructive) {
              showDeleteConfirm = true
            } label: {
              Label("Удалить план", systemImage: "trash")
            }
          } label: {
            Image(systemName: "ellipsis.circle")
          }
        }
      }
    }
    .refreshable {
      await loadMealPlan(showLoader: false)
    }
    .task {
      await loadMealPlan()
    }
    .sheet(isPresented: $showEditSheet) {
      MealPlanEditorSheet(profileId: profileId, existingPlan: plan, existingItems: items) {
        await loadMealPlan(showLoader: false)
      }
    }
    .alert("Удалить план питания?", isPresented: $showDeleteConfirm) {
      Button("Удалить", role: .destructive) {
        Task { await deletePlan() }
      }
      Button("Отмена", role: .cancel) {}
    } message: {
      Text("Это действие нельзя отменить")
    }
    .alert("Слишком много запросов", isPresented: $showRateLimited) {
      Button("OK", role: .cancel) {}
    } message: {
      Text("Попробуйте позже")
    }
  }

  private var emptyState: some View {
    VStack(spacing: 16) {
      Image(systemName: "calendar.badge.clock")
        .font(.system(size: 60))
        .foregroundStyle(.secondary)

      Text("План не задан")
        .font(.headline)
        .foregroundStyle(.primary)

      Text("Создайте недельный план питания")
        .font(.subheadline)
        .foregroundStyle(.secondary)
        .multilineTextAlignment(.center)

      Button {
        showEditSheet = true
      } label: {
        Text("Создать план")
          .fontWeight(.semibold)
          .frame(maxWidth: .infinity)
      }
      .buttonStyle(.borderedProminent)
      .padding(.horizontal, 40)
      .padding(.top, 8)
    }
    .padding()
  }

  private var planContent: some View {
    ScrollView {
      VStack(alignment: .leading, spacing: 16) {
        // Title
        if let currentPlan = plan {
          VStack(alignment: .leading, spacing: 4) {
            Text(currentPlan.title)
              .font(.title2)
              .fontWeight(.bold)

            Text("Недельный план")
              .font(.caption)
              .foregroundStyle(.secondary)
          }
          .padding(.horizontal)
          .padding(.top, 8)
        }

        if let error = errorMessage {
          Text(error)
            .font(.caption)
            .foregroundStyle(.secondary)
            .padding(.horizontal)
        }

        // 7-day table
        VStack(spacing: 0) {
          ForEach(0..<7, id: \.self) { dayIndex in
            dayRow(dayIndex: dayIndex)
            if dayIndex < 6 {
              Divider()
            }
          }
        }
        .background(Color(.systemBackground))
        .clipShape(RoundedRectangle(cornerRadius: 12))
        .shadow(color: Color.black.opacity(0.05), radius: 4, x: 0, y: 2)
        .padding(.horizontal)
      }
      .padding(.bottom, 16)
    }
  }

  @ViewBuilder
  private func dayRow(dayIndex: Int) -> some View {
    VStack(alignment: .leading, spacing: 8) {
      // Day header
      Text("День \(dayIndex)")
        .font(.headline)
        .foregroundStyle(.primary)
        .padding(.horizontal, 12)
        .padding(.top, 12)

      // Meal slots
      ForEach(mealSlots, id: \.self) { slot in
        let item = items.first { $0.dayIndex == dayIndex && $0.mealSlot == slot }

        HStack(alignment: .top, spacing: 8) {
          Text(slotNames[slot] ?? slot)
            .font(.caption)
            .foregroundStyle(.secondary)
            .frame(width: 80, alignment: .leading)

          if let item = item {
            VStack(alignment: .leading, spacing: 2) {
              Text(item.title)
                .font(.subheadline)
                .foregroundStyle(.primary)

              HStack(spacing: 8) {
                macroLabel("ккал", item.approxKcal)
                macroLabel("Б", item.approxProteinG, color: .green)
                macroLabel("Ж", item.approxFatG, color: .orange)
                macroLabel("У", item.approxCarbsG, color: .purple)
              }
              .font(.caption2)
            }
          } else {
            Text("—")
              .font(.subheadline)
              .foregroundStyle(.tertiary)
          }

          Spacer()
        }
        .padding(.horizontal, 12)
      }

      Spacer(minLength: 8)
    }
    .padding(.bottom, 8)
  }

  @ViewBuilder
  private func macroLabel(_ label: String, _ value: Int, color: Color = .secondary) -> some View {
    if value > 0 {
      HStack(spacing: 2) {
        Text(label + ":")
        Text("\(value)")
          .foregroundStyle(color)
          .fontWeight(.medium)
      }
      .foregroundStyle(.secondary)
    }
  }

  // MARK: - Actions

  private func loadMealPlan(showLoader: Bool = true) async {
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
      let response = try await APIClient.shared.fetchMealPlan(profileId: profileId)
      plan = response.plan
      items = response.items
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка загрузки плана (\(errorCode(error)))"
      }
    }
  }

  private func deletePlan() async {
    do {
      try await APIClient.shared.deleteMealPlan(profileId: profileId)
      plan = nil
      items = []
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка удаления плана (\(errorCode(error)))"
      }
    }
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

// MARK: - Meal Plan Editor Sheet

struct MealPlanEditorSheet: View {
  let profileId: UUID
  let existingPlan: MealPlanDTO?
  let existingItems: [MealPlanItemDTO]
  let onSave: () async -> Void

  @Environment(\.dismiss) private var dismiss
  @State private var title = ""
  @State private var editableItems: [EditableMealItem] = []
  @State private var showAddItemSheet = false
  @State private var isSaving = false
  @State private var errorMessage: String?

  private let mealSlots = ["breakfast", "lunch", "dinner", "snack"]
  private let slotNames: [String: String] = [
    "breakfast": "Завтрак",
    "lunch": "Обед",
    "dinner": "Ужин",
    "snack": "Перекус",
  ]

  var body: some View {
    NavigationStack {
      Form {
        Section("План") {
          TextField("Название плана", text: $title)
        }

        Section {
          ForEach(editableItems.indices, id: \.self) { index in
            itemRow(editableItems[index])
          }
          .onDelete { indexSet in
            editableItems.remove(atOffsets: indexSet)
          }

          Button {
            showAddItemSheet = true
          } label: {
            Label("Добавить блюдо", systemImage: "plus.circle.fill")
          }
          .disabled(editableItems.count >= 28)
        } header: {
          Text("Блюда (\(editableItems.count)/28)")
        }

        if let error = errorMessage {
          Section {
            Text(error)
              .font(.caption)
              .foregroundStyle(.red)
          }
        }
      }
      .navigationTitle(existingPlan == nil ? "Новый план" : "Редактировать")
      .navigationBarTitleDisplayMode(.inline)
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("Отмена") {
            dismiss()
          }
          .disabled(isSaving)
        }

        ToolbarItem(placement: .confirmationAction) {
          Button {
            Task { await savePlan() }
          } label: {
            if isSaving {
              ProgressView()
            } else {
              Text("Сохранить")
            }
          }
          .disabled(!isValid || isSaving)
        }
      }
      .onAppear {
        if let existing = existingPlan {
          title = existing.title
          editableItems = existingItems.map { item in
            EditableMealItem(
              dayIndex: item.dayIndex,
              mealSlot: item.mealSlot,
              title: item.title,
              notes: item.notes,
              approxKcal: item.approxKcal,
              approxProteinG: item.approxProteinG,
              approxFatG: item.approxFatG,
              approxCarbsG: item.approxCarbsG
            )
          }
        } else {
          title = "Мой план питания"
        }
      }
      .sheet(isPresented: $showAddItemSheet) {
        AddMealItemSheet(existingItems: editableItems) { newItem in
          editableItems.append(newItem)
        }
      }
    }
  }

  @ViewBuilder
  private func itemRow(_ item: EditableMealItem) -> some View {
    VStack(alignment: .leading, spacing: 4) {
      HStack {
        Text("День \(item.dayIndex)")
          .font(.caption)
          .foregroundStyle(.white)
          .padding(.horizontal, 8)
          .padding(.vertical, 3)
          .background(Color.blue)
          .clipShape(Capsule())

        Text(slotNames[item.mealSlot] ?? item.mealSlot)
          .font(.caption)
          .foregroundStyle(.secondary)

        Spacer()
      }

      Text(item.title)
        .font(.subheadline)
        .fontWeight(.medium)

      if item.approxKcal > 0 || item.approxProteinG > 0 || item.approxFatG > 0
        || item.approxCarbsG > 0
      {
        HStack(spacing: 8) {
          if item.approxKcal > 0 {
            macroLabel("ккал", item.approxKcal)
          }
          if item.approxProteinG > 0 {
            macroLabel("Б", item.approxProteinG, color: .green)
          }
          if item.approxFatG > 0 {
            macroLabel("Ж", item.approxFatG, color: .orange)
          }
          if item.approxCarbsG > 0 {
            macroLabel("У", item.approxCarbsG, color: .purple)
          }
        }
        .font(.caption2)
      }
    }
    .padding(.vertical, 4)
  }

  @ViewBuilder
  private func macroLabel(_ label: String, _ value: Int, color: Color = .secondary) -> some View {
    HStack(spacing: 2) {
      Text(label + ":")
        .foregroundStyle(.secondary)
      Text("\(value)")
        .foregroundStyle(color)
        .fontWeight(.medium)
    }
  }

  private var isValid: Bool {
    !title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty && !editableItems.isEmpty
  }

  private func savePlan() async {
    isSaving = true
    errorMessage = nil

    let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)

    // Validate no duplicates
    var seen = Set<String>()
    for item in editableItems {
      let key = "\(item.dayIndex):\(item.mealSlot)"
      if seen.contains(key) {
        errorMessage =
          "Дубликат: День \(item.dayIndex), \(slotNames[item.mealSlot] ?? item.mealSlot)"
        isSaving = false
        return
      }
      seen.insert(key)
    }

    let upsertItems = editableItems.map { item in
      MealPlanItemUpsertDTO(
        dayIndex: item.dayIndex,
        mealSlot: item.mealSlot,
        title: item.title,
        notes: item.notes,
        approxKcal: item.approxKcal,
        approxProteinG: item.approxProteinG,
        approxFatG: item.approxFatG,
        approxCarbsG: item.approxCarbsG
      )
    }

    do {
      _ = try await APIClient.shared.replaceMealPlan(
        profileId: profileId,
        title: trimmedTitle,
        items: upsertItems
      )

      await onSave()
      dismiss()
    } catch {
      if let apiError = error as? APIError {
        errorMessage = "Ошибка сохранения: \(apiError.localizedDescription)"
      } else {
        errorMessage = "Ошибка сохранения: \(error.localizedDescription)"
      }
    }

    isSaving = false
  }
}

// MARK: - Add Meal Item Sheet

struct AddMealItemSheet: View {
  let existingItems: [EditableMealItem]
  let onAdd: (EditableMealItem) -> Void

  @Environment(\.dismiss) private var dismiss
  @State private var dayIndex = 0
  @State private var mealSlot = "breakfast"
  @State private var title = ""
  @State private var notes = ""
  @State private var kcal = ""
  @State private var protein = ""
  @State private var fat = ""
  @State private var carbs = ""

  private let mealSlots = ["breakfast", "lunch", "dinner", "snack"]
  private let slotNames: [String: String] = [
    "breakfast": "Завтрак",
    "lunch": "Обед",
    "dinner": "Ужин",
    "snack": "Перекус",
  ]

  var body: some View {
    NavigationStack {
      Form {
        Section("Время") {
          Picker("День", selection: $dayIndex) {
            ForEach(0..<7, id: \.self) { day in
              Text("День \(day)").tag(day)
            }
          }

          Picker("Приём пищи", selection: $mealSlot) {
            ForEach(mealSlots, id: \.self) { slot in
              Text(slotNames[slot] ?? slot).tag(slot)
            }
          }
        }

        Section("Блюдо") {
          TextField("Название", text: $title)
          TextField("Заметки (опционально)", text: $notes, axis: .vertical)
            .lineLimit(2...4)
        }

        Section("Макросы (опционально)") {
          HStack {
            Text("Калории:")
            Spacer()
            TextField("0", text: $kcal)
              .keyboardType(.numberPad)
              .multilineTextAlignment(.trailing)
              .frame(width: 80)
          }

          HStack {
            Text("Белки (г):")
            Spacer()
            TextField("0", text: $protein)
              .keyboardType(.numberPad)
              .multilineTextAlignment(.trailing)
              .frame(width: 80)
          }

          HStack {
            Text("Жиры (г):")
            Spacer()
            TextField("0", text: $fat)
              .keyboardType(.numberPad)
              .multilineTextAlignment(.trailing)
              .frame(width: 80)
          }

          HStack {
            Text("Углеводы (г):")
            Spacer()
            TextField("0", text: $carbs)
              .keyboardType(.numberPad)
              .multilineTextAlignment(.trailing)
              .frame(width: 80)
          }
        }

        if isDuplicate {
          Section {
            Text("Блюдо для этого дня и приёма уже добавлено")
              .font(.caption)
              .foregroundStyle(.red)
          }
        }
      }
      .navigationTitle("Добавить блюдо")
      .navigationBarTitleDisplayMode(.inline)
      .toolbar {
        ToolbarItem(placement: .cancellationAction) {
          Button("Отмена") {
            dismiss()
          }
        }

        ToolbarItem(placement: .confirmationAction) {
          Button("Добавить") {
            let newItem = EditableMealItem(
              dayIndex: dayIndex,
              mealSlot: mealSlot,
              title: title.trimmingCharacters(in: .whitespacesAndNewlines),
              notes: notes.trimmingCharacters(in: .whitespacesAndNewlines),
              approxKcal: Int(kcal) ?? 0,
              approxProteinG: Int(protein) ?? 0,
              approxFatG: Int(fat) ?? 0,
              approxCarbsG: Int(carbs) ?? 0
            )
            onAdd(newItem)
            dismiss()
          }
          .disabled(!isValid || isDuplicate)
        }
      }
    }
  }

  private var isValid: Bool {
    !title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
  }

  private var isDuplicate: Bool {
    existingItems.contains { $0.dayIndex == dayIndex && $0.mealSlot == mealSlot }
  }
}

// MARK: - Editable Meal Item

struct EditableMealItem: Identifiable {
  let id = UUID()
  let dayIndex: Int
  let mealSlot: String
  let title: String
  let notes: String
  let approxKcal: Int
  let approxProteinG: Int
  let approxFatG: Int
  let approxCarbsG: Int
}

struct MealPlanView_Previews: PreviewProvider {
  static var previews: some View {
    NavigationStack {
      MealPlanView(profileId: UUID())
    }
  }
}
