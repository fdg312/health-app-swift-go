//
//  WorkoutsPlanView.swift
//  HealthHub
//
//  Created by HealthHub on 2024.
//

import Combine
import SwiftUI

struct WorkoutsPlanView: View {
  @StateObject private var viewModel = WorkoutsPlanViewModel()
  @State private var showEditSheet = false
  @State private var showError = false
  @State private var errorMessage = ""

  let profileId: UUID

  var body: some View {
    ScrollView {
      VStack(spacing: 20) {
        if viewModel.isLoading {
          ProgressView("Загрузка...")
            .padding()
        } else if let plan = viewModel.plan {
          // Plan Header
          VStack(alignment: .leading, spacing: 8) {
            HStack {
              VStack(alignment: .leading) {
                Text(plan.title)
                  .font(.title2)
                  .fontWeight(.bold)
                if !plan.goal.isEmpty {
                  Text(plan.goal.capitalized)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                }
              }
              Spacer()
              Button("Редактировать") {
                showEditSheet = true
              }
              .buttonStyle(.bordered)
            }
          }
          .padding()
          .background(Color(.systemBackground))
          .cornerRadius(12)
          .shadow(radius: 2)

          // Today's Workouts
          if !viewModel.todayPlanned.isEmpty {
            VStack(alignment: .leading, spacing: 12) {
              Text("Сегодня")
                .font(.headline)
                .padding(.horizontal)

              ForEach(viewModel.todayPlanned) { item in
                WorkoutItemCard(
                  item: item,
                  completion: viewModel.todayCompletions.first(where: {
                    $0.planItemId == item.id
                  }),
                  onComplete: {
                    Task {
                      await viewModel.markComplete(item: item, profileId: profileId)
                    }
                  },
                  onSkip: {
                    Task {
                      await viewModel.markSkipped(item: item, profileId: profileId)
                    }
                  }
                )
              }
            }
          }

          // All Plan Items
          VStack(alignment: .leading, spacing: 12) {
            Text("План тренировок")
              .font(.headline)
              .padding(.horizontal)

            if viewModel.items.isEmpty {
              Text("План пуст")
                .foregroundColor(.secondary)
                .padding()
            } else {
              ForEach(viewModel.items) { item in
                WorkoutPlanItemCard(item: item)
              }
            }
          }
        } else {
          VStack(spacing: 16) {
            Image(systemName: "figure.run")
              .font(.system(size: 60))
              .foregroundColor(.secondary)
            Text("План тренировок не создан")
              .font(.headline)
            Button("Создать план") {
              showEditSheet = true
            }
            .buttonStyle(.borderedProminent)
          }
          .padding()
        }
      }
      .padding()
    }
    .navigationTitle("Тренировки")
    .task {
      await viewModel.loadPlan(profileId: profileId)
      await viewModel.loadToday(profileId: profileId)
    }
    .sheet(isPresented: $showEditSheet) {
      EditWorkoutPlanView(
        profileId: profileId,
        existingPlan: viewModel.plan,
        existingItems: viewModel.items
      ) {
        Task {
          await viewModel.loadPlan(profileId: profileId)
          await viewModel.loadToday(profileId: profileId)
        }
      }
    }
    .alert("Ошибка", isPresented: $showError) {
      Button("OK", role: .cancel) {}
    } message: {
      Text(errorMessage)
    }
  }
}

struct WorkoutItemCard: View {
  let item: WorkoutItemDTO
  let completion: WorkoutCompletionDTO?
  let onComplete: () -> Void
  let onSkip: () -> Void

  var body: some View {
    VStack(alignment: .leading, spacing: 8) {
      HStack {
        VStack(alignment: .leading, spacing: 4) {
          HStack {
            Image(systemName: iconForKind(item.kind))
              .foregroundColor(.blue)
            Text(item.kindLocalized)
              .font(.headline)
          }
          Text("\(item.timeFormatted) • \(item.durationMin) мин • \(item.intensityLocalized)")
            .font(.caption)
            .foregroundColor(.secondary)
          if !item.note.isEmpty {
            Text(item.note)
              .font(.caption)
              .foregroundColor(.secondary)
          }
        }
        Spacer()

        if let comp = completion {
          Image(systemName: comp.status == "done" ? "checkmark.circle.fill" : "xmark.circle")
            .foregroundColor(comp.status == "done" ? .green : .orange)
            .font(.title2)
        } else {
          HStack(spacing: 8) {
            Button(action: onComplete) {
              Image(systemName: "checkmark.circle")
                .font(.title3)
            }
            Button(action: onSkip) {
              Image(systemName: "xmark.circle")
                .font(.title3)
            }
          }
        }
      }
    }
    .padding()
    .background(Color(.systemBackground))
    .cornerRadius(12)
    .shadow(radius: 2)
    .padding(.horizontal)
  }

  private func iconForKind(_ kind: String) -> String {
    switch kind {
    case "run": return "figure.run"
    case "walk": return "figure.walk"
    case "strength": return "dumbbell.fill"
    case "morning": return "sunrise.fill"
    case "core": return "figure.core.training"
    default: return "figure.mixed.cardio"
    }
  }
}

struct WorkoutPlanItemCard: View {
  let item: WorkoutItemDTO

  var body: some View {
    VStack(alignment: .leading, spacing: 8) {
      HStack {
        Image(systemName: iconForKind(item.kind))
          .foregroundColor(.blue)
        Text(item.kindLocalized)
          .font(.headline)
        Spacer()
        Text(item.intensityLocalized)
          .font(.caption)
          .padding(.horizontal, 8)
          .padding(.vertical, 4)
          .background(Color.blue.opacity(0.2))
          .cornerRadius(8)
      }
      Text("\(item.timeFormatted) • \(item.durationMin) мин")
        .font(.subheadline)
        .foregroundColor(.secondary)
      Text(item.daysFormatted)
        .font(.caption)
        .foregroundColor(.secondary)
      if !item.note.isEmpty {
        Text(item.note)
          .font(.caption)
          .foregroundColor(.secondary)
      }
    }
    .padding()
    .background(Color(.systemBackground))
    .cornerRadius(12)
    .shadow(radius: 2)
    .padding(.horizontal)
  }

  private func iconForKind(_ kind: String) -> String {
    switch kind {
    case "run": return "figure.run"
    case "walk": return "figure.walk"
    case "strength": return "dumbbell.fill"
    case "morning": return "sunrise.fill"
    case "core": return "figure.core.training"
    default: return "figure.mixed.cardio"
    }
  }
}

@MainActor
class WorkoutsPlanViewModel: ObservableObject {
  @Published var plan: WorkoutPlanDTO?
  @Published var items: [WorkoutItemDTO] = []
  @Published var todayPlanned: [WorkoutItemDTO] = []
  @Published var todayCompletions: [WorkoutCompletionDTO] = []
  @Published var isLoading = false

  func loadPlan(profileId: UUID) async {
    isLoading = true
    defer { isLoading = false }

    do {
      let response = try await APIClient.shared.fetchWorkoutPlan(profileId: profileId)
      plan = response.plan
      items = response.items
    } catch {
      print("Error loading workout plan: \(error)")
    }
  }

  func loadToday(profileId: UUID) async {
    let dateFormatter = DateFormatter()
    dateFormatter.dateFormat = "yyyy-MM-dd"
    let today = dateFormatter.string(from: Date())

    do {
      let response = try await APIClient.shared.fetchWorkoutToday(
        profileId: profileId, date: today)
      todayPlanned = response.planned
      todayCompletions = response.completions
    } catch {
      print("Error loading today's workouts: \(error)")
    }
  }

  func markComplete(item: WorkoutItemDTO, profileId: UUID) async {
    let dateFormatter = DateFormatter()
    dateFormatter.dateFormat = "yyyy-MM-dd"
    let today = dateFormatter.string(from: Date())

    do {
      _ = try await APIClient.shared.upsertWorkoutCompletion(
        profileId: profileId,
        date: today,
        planItemId: item.id,
        status: "done"
      )
      await loadToday(profileId: profileId)
    } catch {
      print("Error marking complete: \(error)")
    }
  }

  func markSkipped(item: WorkoutItemDTO, profileId: UUID) async {
    let dateFormatter = DateFormatter()
    dateFormatter.dateFormat = "yyyy-MM-dd"
    let today = dateFormatter.string(from: Date())

    do {
      _ = try await APIClient.shared.upsertWorkoutCompletion(
        profileId: profileId,
        date: today,
        planItemId: item.id,
        status: "skipped"
      )
      await loadToday(profileId: profileId)
    } catch {
      print("Error marking skipped: \(error)")
    }
  }
}

struct EditWorkoutPlanView: View {
  @Environment(\.dismiss) var dismiss
  @State private var title: String
  @State private var goal: String
  @State private var items: [WorkoutItemEditModel] = []
  @State private var isLoading = false
  @State private var showError = false
  @State private var errorMessage = ""

  let profileId: UUID
  let onSave: () -> Void

  init(
    profileId: UUID, existingPlan: WorkoutPlanDTO?, existingItems: [WorkoutItemDTO],
    onSave: @escaping () -> Void
  ) {
    self.profileId = profileId
    self.onSave = onSave
    _title = State(initialValue: existingPlan?.title ?? "")
    _goal = State(initialValue: existingPlan?.goal ?? "")
    _items = State(
      initialValue: existingItems.map { WorkoutItemEditModel(from: $0) })
  }

  var body: some View {
    NavigationStack {
      Form {
        Section("Основное") {
          TextField("Название плана", text: $title)
          TextField("Цель", text: $goal)
        }

        Section("Тренировки") {
          ForEach($items) { $item in
            VStack(alignment: .leading, spacing: 8) {
              Picker("Тип", selection: $item.kind) {
                Text("Бег").tag("run")
                Text("Прогулка").tag("walk")
                Text("Силовая").tag("strength")
                Text("Утренняя зарядка").tag("morning")
                Text("Кор").tag("core")
                Text("Другое").tag("other")
              }

              HStack {
                Text("Время:")
                Spacer()
                TextField("Минуты", value: $item.timeMinutes, format: .number)
                  .multilineTextAlignment(.trailing)
                  .keyboardType(.numberPad)
                  .frame(width: 80)
              }

              HStack {
                Text("Длительность:")
                Spacer()
                TextField("Мин", value: $item.durationMin, format: .number)
                  .multilineTextAlignment(.trailing)
                  .keyboardType(.numberPad)
                  .frame(width: 80)
              }

              Picker("Интенсивность", selection: $item.intensity) {
                Text("Низкая").tag("low")
                Text("Средняя").tag("medium")
                Text("Высокая").tag("high")
              }

              TextField("Заметка", text: $item.note)
            }
          }
          .onDelete { indices in
            items.remove(atOffsets: indices)
          }

          Button("Добавить тренировку") {
            items.append(WorkoutItemEditModel())
          }
        }
      }
      .navigationTitle("Редактировать план")
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
          .disabled(title.isEmpty || items.isEmpty || isLoading)
        }
      }
      .alert("Ошибка", isPresented: $showError) {
        Button("OK", role: .cancel) {}
      } message: {
        Text(errorMessage)
      }
    }
  }

  private func save() async {
    isLoading = true
    defer { isLoading = false }

    do {
      let upsertItems = items.map { item in
        WorkoutItemUpsert(
          kind: item.kind,
          timeMinutes: item.timeMinutes,
          daysMask: item.daysMask,
          durationMin: item.durationMin,
          intensity: item.intensity,
          note: item.note,
          details: nil
        )
      }

      _ = try await APIClient.shared.replaceWorkoutPlan(
        profileId: profileId,
        title: title,
        goal: goal,
        items: upsertItems
      )

      onSave()
      dismiss()
    } catch {
      errorMessage = "Не удалось сохранить план: \(error.localizedDescription)"
      showError = true
    }
  }
}

struct WorkoutItemEditModel: Identifiable {
  var id = UUID()
  var kind: String = "run"
  var timeMinutes: Int = 420
  var daysMask: Int = 127
  var durationMin: Int = 30
  var intensity: String = "medium"
  var note: String = ""

  init() {}

  init(from dto: WorkoutItemDTO) {
    self.kind = dto.kind
    self.timeMinutes = dto.timeMinutes
    self.daysMask = dto.daysMask
    self.durationMin = dto.durationMin
    self.intensity = dto.intensity
    self.note = dto.note
  }
}
