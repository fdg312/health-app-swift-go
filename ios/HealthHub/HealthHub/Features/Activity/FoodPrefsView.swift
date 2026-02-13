import SwiftUI

struct FoodPrefsView: View {
  @ObservedObject private var auth = AuthManager.shared
  let profileId: UUID

  @State private var foodPrefs: [FoodPrefDTO] = []
  @State private var isLoading = false
  @State private var errorMessage: String?
  @State private var searchText = ""
  @State private var showAddSheet = false
  @State private var showDeleteConfirm = false
  @State private var prefToDelete: FoodPrefDTO?
  @State private var showRateLimited = false

  var body: some View {
    Group {
      if isLoading && foodPrefs.isEmpty {
        ProgressView("Загрузка...")
      } else {
        VStack(spacing: 0) {
          if let error = errorMessage {
            Text(error)
              .font(.caption)
              .foregroundStyle(.secondary)
              .frame(maxWidth: .infinity, alignment: .leading)
              .padding(.horizontal)
              .padding(.vertical, 8)
          }

          List {
            if foodPrefs.isEmpty {
              Text("Нет данных")
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .center)
                .listRowBackground(Color.clear)
            } else {
              ForEach(foodPrefs) { pref in
                foodPrefRow(pref)
                  .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                    Button(role: .destructive) {
                      prefToDelete = pref
                      showDeleteConfirm = true
                    } label: {
                      Label("Удалить", systemImage: "trash")
                    }
                  }
              }
            }
          }
          .listStyle(.plain)
        }
      }
    }
    .navigationTitle("Обычные продукты")
    .searchable(text: $searchText, prompt: "Поиск продуктов")
    .onChange(of: searchText) { _, _ in
      Task {
        try? await Task.sleep(nanoseconds: 300_000_000)  // 300ms debounce
        await loadFoodPrefs(showLoader: false)
      }
    }
    .toolbar {
      ToolbarItem(placement: .topBarTrailing) {
        Button {
          showAddSheet = true
        } label: {
          Image(systemName: "plus")
        }
      }
    }
    .refreshable {
      await loadFoodPrefs(showLoader: false)
    }
    .task {
      await loadFoodPrefs()
    }
    .sheet(isPresented: $showAddSheet) {
      AddFoodPrefSheet(profileId: profileId) {
        await loadFoodPrefs(showLoader: false)
      }
    }
    .alert("Удалить продукт?", isPresented: $showDeleteConfirm) {
      Button("Удалить", role: .destructive) {
        if let pref = prefToDelete {
          Task { await deleteFoodPref(pref) }
        }
      }
      Button("Отмена", role: .cancel) {}
    }
    .alert("Слишком много запросов", isPresented: $showRateLimited) {
      Button("OK", role: .cancel) {}
    } message: {
      Text("Попробуйте позже")
    }
  }

  @ViewBuilder
  private func foodPrefRow(_ pref: FoodPrefDTO) -> some View {
    VStack(alignment: .leading, spacing: 6) {
      // Name
      Text(pref.name)
        .font(.subheadline)
        .fontWeight(.medium)
        .foregroundStyle(.primary)

      // Tags
      if !pref.tags.isEmpty {
        ScrollView(.horizontal, showsIndicators: false) {
          HStack(spacing: 6) {
            ForEach(pref.tags, id: \.self) { tag in
              Text(tag)
                .font(.caption2)
                .padding(.horizontal, 8)
                .padding(.vertical, 3)
                .background(Color.blue.opacity(0.15))
                .foregroundStyle(.blue)
                .clipShape(Capsule())
            }
          }
        }
      }

      // Macros
      HStack(spacing: 12) {
        macroView(label: "ккал", value: pref.kcalPer100g)
        macroView(label: "Б", value: pref.proteinGPer100g, color: .green)
        macroView(label: "Ж", value: pref.fatGPer100g, color: .orange)
        macroView(label: "У", value: pref.carbsGPer100g, color: .purple)
      }
      .font(.caption2)
      .foregroundStyle(.secondary)
    }
    .padding(.vertical, 4)
  }

  @ViewBuilder
  private func macroView(label: String, value: Int, color: Color = .secondary) -> some View {
    HStack(spacing: 2) {
      Text(label + ":")
        .foregroundStyle(.secondary)
      Text("\(value)")
        .foregroundStyle(color)
        .fontWeight(.medium)
    }
  }

  // MARK: - Actions

  private func loadFoodPrefs(showLoader: Bool = true) async {
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
      let response = try await APIClient.shared.listFoodPrefs(
        profileId: profileId,
        query: searchText.isEmpty ? nil : searchText,
        limit: 200,
        offset: 0
      )
      foodPrefs = response.items
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка загрузки продуктов (\(errorCode(error)))"
      }
    }
  }

  private func deleteFoodPref(_ pref: FoodPrefDTO) async {
    guard let id = UUID(uuidString: pref.id) else {
      errorMessage = "Неверный ID продукта"
      return
    }

    do {
      try await APIClient.shared.deleteFoodPref(id: id)
      await loadFoodPrefs(showLoader: false)
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка удаления продукта (\(errorCode(error)))"
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

// MARK: - Add Food Pref Sheet

struct AddFoodPrefSheet: View {
  let profileId: UUID
  let onSave: () async -> Void

  @Environment(\.dismiss) private var dismiss
  @State private var name = ""
  @State private var tagsText = ""
  @State private var kcal = ""
  @State private var protein = ""
  @State private var fat = ""
  @State private var carbs = ""
  @State private var isSaving = false
  @State private var errorMessage: String?

  var body: some View {
    NavigationStack {
      Form {
        Section("Продукт") {
          TextField("Название", text: $name)
          TextField("Теги (через запятую)", text: $tagsText)
            .textInputAutocapitalization(.never)
        }

        Section("Макросы (на 100г, опционально)") {
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

        if let error = errorMessage {
          Section {
            Text(error)
              .font(.caption)
              .foregroundStyle(.red)
          }
        }
      }
      .navigationTitle("Новый продукт")
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
            Task { await saveFoodPref() }
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
    }
  }

  private var isValid: Bool {
    !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
  }

  private func saveFoodPref() async {
    isSaving = true
    errorMessage = nil

    let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
    let tags =
      tagsText
      .split(separator: ",")
      .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
      .filter { !$0.isEmpty }

    let kcalValue = Int(kcal) ?? 0
    let proteinValue = Int(protein) ?? 0
    let fatValue = Int(fat) ?? 0
    let carbsValue = Int(carbs) ?? 0

    do {
      _ = try await APIClient.shared.upsertFoodPref(
        profileId: profileId,
        name: trimmedName,
        tags: tags,
        kcalPer100g: kcalValue,
        proteinPer100g: proteinValue,
        fatPer100g: fatValue,
        carbsPer100g: carbsValue
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

struct FoodPrefsView_Previews: PreviewProvider {
  static var previews: some View {
    NavigationStack {
      FoodPrefsView(profileId: UUID())
    }
  }
}
