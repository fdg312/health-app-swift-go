import SafariServices
import SwiftUI

struct ActivityView: View {
  @ObservedObject private var auth = AuthManager.shared
  @State private var showRateLimited = false
  @State private var profiles: [ProfileDTO] = []
  @State private var sources: [SourceDTO] = []
  @State private var isLoading = false
  @State private var errorMessage: String?
  @State private var searchText = ""
  @State private var selectedTab: ActivityTab = .sources
  @State private var selectedFilter: SourceFilter = .all
  @State private var showCreateSheet = false
  @State private var createType: CreateSourceType?
  @State private var previewSource: SourceDTO?
  @State private var showPreview = false
  @State private var safariURL: URL?
  @State private var showSafari = false
  @State private var sourceToDelete: SourceDTO?
  @State private var showDeleteConfirm = false
  @State private var selectedNutritionTab: NutritionTab = .foodPrefs

  enum ActivityTab: String, CaseIterable {
    case sources = "Источники"
    case intakes = "Прием"
    case workouts = "Тренировки"
    case nutrition = "Питание"
  }

  enum SourceFilter: String, CaseIterable {
    case all = "Все"
    case image = "Фото"
    case link = "Ссылки"
    case note = "Заметки"

    var apiKind: String? {
      switch self {
      case .all: return nil
      case .image: return "image"
      case .link: return "link"
      case .note: return "note"
      }
    }
  }

  enum CreateSourceType {
    case link, note
  }

  private var ownerProfile: ProfileDTO? {
    profiles.first(where: { $0.type == "owner" })
  }

  private var filteredSources: [SourceDTO] {
    guard let kindFilter = selectedFilter.apiKind else {
      return sources
    }
    return sources.filter { $0.kind == kindFilter }
  }

  var body: some View {
    NavigationStack {
      VStack(spacing: 0) {
        // Top Tab Selector
        Picker("Раздел", selection: $selectedTab) {
          ForEach(ActivityTab.allCases, id: \.self) { tab in
            Text(tab.rawValue).tag(tab)
          }
        }
        .pickerStyle(.segmented)
        .padding()

        // Content based on selected tab
        if selectedTab == .sources {
          sourcesView
        } else if selectedTab == .intakes {
          if let owner = ownerProfile {
            IntakesView(profileId: owner.id)
          } else {
            Text("Профиль не найден")
              .foregroundStyle(.secondary)
          }
        } else if selectedTab == .workouts {
          if let owner = ownerProfile {
            WorkoutsPlanView(profileId: owner.id)
          } else {
            Text("Профиль не найден")
              .foregroundStyle(.secondary)
          }
        } else if selectedTab == .nutrition {
          if let owner = ownerProfile {
            nutritionView(owner: owner)
          } else {
            Text("Профиль не найден")
              .foregroundStyle(.secondary)
          }
        }
      }
      .navigationTitle("Активность")
      .task {
        do {
          profiles = try await APIClient.shared.listProfiles()
        } catch {
          // Profiles will also be loaded by sourcesView's loadData()
        }
      }
    }
  }

  // MARK: - Sources View

  private var sourcesView: some View {
    VStack(spacing: 0) {
      if isLoading && sources.isEmpty {
        Spacer()
        ProgressView("Загрузка...")
        Spacer()
      } else if let error = errorMessage, sources.isEmpty {
        Spacer()
        EmptyStateView(
          icon: "exclamationmark.triangle",
          title: "Ошибка загрузки",
          description: error
        )
        Spacer()
      } else {
        // Filter
        Picker("Фильтр", selection: $selectedFilter) {
          ForEach(SourceFilter.allCases, id: \.self) { filter in
            Text(filter.rawValue).tag(filter)
          }
        }
        .pickerStyle(.segmented)
        .padding()

        if let error = errorMessage {
          Text(error)
            .font(.caption)
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal)
        }

        // List
        List {
          if filteredSources.isEmpty {
            Section {
              EmptyStateView(
                icon: "tray",
                title: "Нет источников",
                description: "Добавьте ссылки, заметки или фотографии"
              )
              .listRowBackground(Color.clear)
              .listRowInsets(EdgeInsets())
            }
          } else {
            ForEach(filteredSources) { source in
              sourceRow(source)
                .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                  Button(role: .destructive) {
                    sourceToDelete = source
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
    .navigationTitle("Источники")
    .searchable(text: $searchText, prompt: "Поиск по источникам")
    .onChange(of: searchText) { _, _ in
      Task {
        try? await Task.sleep(nanoseconds: 300_000_000)  // 300ms debounce
        await loadSources(showLoader: false)
      }
    }
    .onChange(of: selectedFilter) { _, _ in
      // Client-side filter, no reload needed
    }
    .toolbar {
      ToolbarItem(placement: .topBarTrailing) {
        Menu {
          Button {
            createType = .link
            showCreateSheet = true
          } label: {
            Label("Ссылка", systemImage: "link")
          }

          Button {
            createType = .note
            showCreateSheet = true
          } label: {
            Label("Заметка", systemImage: "note.text")
          }
        } label: {
          Image(systemName: "plus")
        }
      }
    }
    .refreshable {
      await refreshSources()
    }
    .task {
      await loadData()
    }
    .sheet(isPresented: $showCreateSheet) {
      if let type = createType {
        CreateSourceSheet(
          type: type,
          profileId: ownerProfile?.id,
          onSave: { await loadSources(showLoader: false) }
        )
      }
    }
    .sheet(isPresented: $showPreview) {
      if let source = previewSource {
        if source.kind == "note" {
          NotePreviewSheet(source: source)
        } else {
          PhotoPreviewSheet(source: source)
        }
      }
    }
    .sheet(isPresented: $showSafari) {
      if let url = safariURL {
        SafariView(url: url)
      }
    }
    .alert("Удалить источник?", isPresented: $showDeleteConfirm) {
      Button("Удалить", role: .destructive) {
        if let source = sourceToDelete {
          Task { await deleteSource(source) }
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

  @ViewBuilder
  func sourceRow(_ source: SourceDTO) -> some View {
    Button {
      handleSourceTap(source)
    } label: {
      HStack(spacing: 12) {
        Circle()
          .fill(sourceColor(source.kind).opacity(0.2))
          .frame(width: 44, height: 44)
          .overlay(
            Image(systemName: sourceIcon(source.kind))
              .font(.system(size: 18))
              .foregroundStyle(sourceColor(source.kind))
          )

        VStack(alignment: .leading, spacing: 4) {
          Text(sourceTitle(source))
            .font(.subheadline)
            .fontWeight(.medium)
            .foregroundStyle(.primary)
            .lineLimit(2)

          if let secondary = sourceSecondary(source) {
            Text(secondary)
              .font(.caption)
              .foregroundStyle(.secondary)
              .lineLimit(1)
          }

          Text(formatDate(source.createdAt))
            .font(.caption2)
            .foregroundStyle(.tertiary)
        }

        Spacer()

        if source.kind == "link" {
          Image(systemName: "arrow.up.right.circle.fill")
            .font(.system(size: 20))
            .foregroundStyle(.secondary)
        }
      }
      .padding(.vertical, 8)
    }
    .buttonStyle(.plain)
  }

  // MARK: - Actions

  private func loadData() async {
    do {
      profiles = try await APIClient.shared.listProfiles()
      await loadSources()
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка загрузки профилей (\(errorCode(error)))"
      }
    }
  }

  private func refreshSources() async {
    if ownerProfile == nil {
      await loadData()
      return
    }
    await loadSources(showLoader: false)
  }

  private func loadSources(showLoader: Bool = true) async {
    guard let profile = ownerProfile else { return }

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
      sources = try await APIClient.shared.listSources(
        profileId: profile.id,
        query: searchText.isEmpty ? nil : searchText,
        checkinId: nil,
        limit: 100,
        offset: 0
      )
    } catch {
      if !handleError(error) {
        errorMessage = "Ошибка загрузки источников (\(errorCode(error)))"
      }
    }
  }

  private func handleSourceTap(_ source: SourceDTO) {
    switch source.kind {
    case "image":
      previewSource = source
      showPreview = true
    case "link":
      if let urlString = source.url, let url = URL(string: urlString) {
        safariURL = url
        showSafari = true
      }
    case "note":
      previewSource = source
      showPreview = true
    default:
      break
    }
  }

  private func deleteSource(_ source: SourceDTO) async {
    do {
      try await APIClient.shared.deleteSource(sourceId: source.id)
      await loadSources(showLoader: false)
    } catch {
      errorMessage = "Ошибка удаления источника (\(errorCode(error)))"
    }
  }

  // MARK: - Helpers

  private func sourceIcon(_ kind: String) -> String {
    switch kind {
    case "image": return "photo"
    case "link": return "link"
    case "note": return "note.text"
    default: return "doc"
    }
  }

  private func sourceColor(_ kind: String) -> Color {
    switch kind {
    case "image": return .blue
    case "link": return .purple
    case "note": return .orange
    default: return .gray
    }
  }

  private func sourceTitle(_ source: SourceDTO) -> String {
    if let title = source.title, !title.isEmpty {
      return title
    }

    switch source.kind {
    case "image": return "Изображение"
    case "link": return source.url ?? "Ссылка"
    case "note":
      if let text = source.text, !text.isEmpty {
        return String(text.prefix(50))
      }
      return "Заметка"
    default: return "Источник"
    }
  }

  private func sourceSecondary(_ source: SourceDTO) -> String? {
    switch source.kind {
    case "image":
      if let size = source.sizeBytes, size > 0 {
        return formatBytes(size)
      }
      return nil
    case "link":
      return source.url
    case "note":
      if let text = source.text, !text.isEmpty {
        return text.replacingOccurrences(of: "\n", with: " ")
      }
      return nil
    default:
      return nil
    }
  }

  private func formatDate(_ date: Date) -> String {
    let formatter = RelativeDateTimeFormatter()
    formatter.unitsStyle = .short
    formatter.locale = Locale(identifier: "ru_RU")
    return formatter.localizedString(for: date, relativeTo: Date())
  }

  private func formatBytes(_ bytes: Int64) -> String {
    if bytes < 1024 {
      return "\(bytes) Б"
    } else if bytes < 1024 * 1024 {
      return String(format: "%.1f КБ", Double(bytes) / 1024.0)
    } else {
      return String(format: "%.1f МБ", Double(bytes) / (1024.0 * 1024.0))
    }
  }

  private func errorCode(_ error: Error) -> String {
    if let apiError = error as? APIError {
      return apiError.uiCode
    }
    return "bad_response"
  }

  // MARK: - Nutrition View

  @ViewBuilder
  private func nutritionView(owner: ProfileDTO) -> some View {
    VStack(spacing: 0) {
      // Sub-tabs for Food Prefs and Meal Plan
      Picker("Раздел", selection: $selectedNutritionTab) {
        Text("Продукты").tag(NutritionTab.foodPrefs)
        Text("План").tag(NutritionTab.mealPlan)
      }
      .pickerStyle(.segmented)
      .padding()

      if selectedNutritionTab == .foodPrefs {
        FoodPrefsView(profileId: owner.id)
      } else {
        MealPlanView(profileId: owner.id)
      }
    }
  }
}

// MARK: - Nutrition Tab

extension ActivityView {
  enum NutritionTab {
    case foodPrefs
    case mealPlan
  }
}

// MARK: - Create Source Sheet

struct CreateSourceSheet: View {
  let type: ActivityView.CreateSourceType
  let profileId: UUID?
  let onSave: () async -> Void

  @Environment(\.dismiss) private var dismiss
  @State private var title = ""
  @State private var url = ""
  @State private var text = ""
  @State private var isSaving = false
  @State private var errorMessage: String?

  var body: some View {
    NavigationStack {
      Form {
        if type == .link {
          Section("Ссылка") {
            TextField("Название (опционально)", text: $title)
            TextField("URL", text: $url)
              .keyboardType(.URL)
              .textInputAutocapitalization(.never)
              .autocorrectionDisabled()
          }
        } else {
          Section("Заметка") {
            TextField("Название (опционально)", text: $title)
            TextEditor(text: $text)
              .frame(minHeight: 120)
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
      .navigationTitle(type == .link ? "Новая ссылка" : "Новая заметка")
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
            Task { await saveSource() }
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
    guard profileId != nil else { return false }

    if type == .link {
      return !url.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    } else {
      return !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }
  }

  private func saveSource() async {
    guard let profileId = profileId else {
      errorMessage = "Профиль не найден"
      return
    }

    isSaving = true
    errorMessage = nil

    do {
      let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
      let titleValue = trimmedTitle.isEmpty ? nil : trimmedTitle

      if type == .link {
        _ = try await APIClient.shared.createSourceLink(
          profileId: profileId,
          checkinId: nil,
          title: titleValue,
          url: url.trimmingCharacters(in: .whitespacesAndNewlines)
        )
      } else {
        _ = try await APIClient.shared.createSourceNote(
          profileId: profileId,
          checkinId: nil,
          title: titleValue,
          text: text.trimmingCharacters(in: .whitespacesAndNewlines)
        )
      }

      await onSave()
      dismiss()
    } catch {
      errorMessage = "Ошибка сохранения: \(error.localizedDescription)"
    }

    isSaving = false
  }
}

// MARK: - Safari View

struct SafariView: UIViewControllerRepresentable {
  let url: URL

  func makeUIViewController(context: Context) -> SFSafariViewController {
    return SFSafariViewController(url: url)
  }

  func updateUIViewController(_ uiViewController: SFSafariViewController, context: Context) {}
}

struct ActivityView_Previews: PreviewProvider {
  static var previews: some View {
    ActivityView()
  }
}
