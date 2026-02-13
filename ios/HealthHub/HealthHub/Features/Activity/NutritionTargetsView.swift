//
//  NutritionTargetsView.swift
//  HealthHub
//
//  Created by HealthHub on 2024.
//

import Combine
import SwiftUI

struct NutritionTargetsView: View {
  @StateObject private var viewModel = NutritionTargetsViewModel()
  @State private var isEditing = false
  @State private var showError = false
  @State private var errorMessage = ""

  let profileId: UUID

  var body: some View {
    ScrollView {
      VStack(spacing: 20) {
        if viewModel.isLoading {
          ProgressView("Загрузка...")
            .padding()
        } else if let targets = viewModel.targets {
          // Header
          VStack(alignment: .leading, spacing: 8) {
            HStack {
              VStack(alignment: .leading) {
                Text("Цели по питанию")
                  .font(.title2)
                  .fontWeight(.bold)
                if viewModel.isDefault {
                  Text("Стандартные значения")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                }
              }
              Spacer()
              Button(isEditing ? "Отмена" : "Изменить") {
                if isEditing {
                  viewModel.loadTargets(profileId: profileId)
                }
                isEditing.toggle()
              }
              .buttonStyle(.bordered)
            }
          }
          .padding()
          .background(Color(.systemBackground))
          .cornerRadius(12)
          .shadow(radius: 2)

          // Targets List
          VStack(spacing: 16) {
            // Calories
            NutritionTargetRow(
              icon: "flame.fill",
              iconColor: .orange,
              label: "Калории",
              value: $viewModel.editedCalories,
              unit: "ккал",
              range: 800...6000,
              isEditing: isEditing
            )

            Divider()

            // Protein
            NutritionTargetRow(
              icon: "bolt.fill",
              iconColor: .red,
              label: "Белки",
              value: $viewModel.editedProtein,
              unit: "г",
              range: 0...400,
              isEditing: isEditing
            )

            Divider()

            // Fats
            NutritionTargetRow(
              icon: "drop.fill",
              iconColor: .yellow,
              label: "Жиры",
              value: $viewModel.editedFat,
              unit: "г",
              range: 0...400,
              isEditing: isEditing
            )

            Divider()

            // Carbs
            NutritionTargetRow(
              icon: "leaf.fill",
              iconColor: .green,
              label: "Углеводы",
              value: $viewModel.editedCarbs,
              unit: "г",
              range: 0...400,
              isEditing: isEditing
            )

            Divider()

            // Calcium
            NutritionTargetRow(
              icon: "heart.fill",
              iconColor: .blue,
              label: "Кальций",
              value: $viewModel.editedCalcium,
              unit: "мг",
              range: 0...5000,
              isEditing: isEditing
            )
          }
          .padding()
          .background(Color(.systemBackground))
          .cornerRadius(12)
          .shadow(radius: 2)

          if isEditing {
            Button(action: saveTargets) {
              if viewModel.isSaving {
                ProgressView()
                  .progressViewStyle(CircularProgressViewStyle(tint: .white))
                  .frame(maxWidth: .infinity)
              } else {
                Text("Сохранить")
                  .fontWeight(.semibold)
                  .frame(maxWidth: .infinity)
              }
            }
            .buttonStyle(.borderedProminent)
            .disabled(viewModel.isSaving || !viewModel.isValid)
            .padding(.horizontal)
          }

          // Info
          VStack(alignment: .leading, spacing: 8) {
            Label("Информация", systemImage: "info.circle")
              .font(.headline)
            Text(
              "Цели по питанию помогают отслеживать ежедневное потребление макронутриентов. Данные синхронизируются с HealthKit."
            )
            .font(.subheadline)
            .foregroundColor(.secondary)
          }
          .padding()
          .background(Color(.systemGray6))
          .cornerRadius(12)
        }
      }
      .padding()
    }
    .navigationTitle("Питание")
    .navigationBarTitleDisplayMode(.inline)
    .onAppear {
      viewModel.loadTargets(profileId: profileId)
    }
    .alert("Ошибка", isPresented: $showError) {
      Button("OK", role: .cancel) {}
    } message: {
      Text(errorMessage)
    }
    .onChange(of: viewModel.errorMessage) { newValue in
      if let error = newValue {
        errorMessage = error
        showError = true
      }
    }
  }

  private func saveTargets() {
    Task {
      let success = await viewModel.saveTargets(profileId: profileId)
      if success {
        isEditing = false
      }
    }
  }
}

struct NutritionTargetRow: View {
  let icon: String
  let iconColor: Color
  let label: String
  @Binding var value: Int
  let unit: String
  let range: ClosedRange<Int>
  let isEditing: Bool

  var body: some View {
    HStack(spacing: 16) {
      Image(systemName: icon)
        .foregroundColor(iconColor)
        .font(.title2)
        .frame(width: 30)

      VStack(alignment: .leading, spacing: 4) {
        Text(label)
          .font(.headline)
        if !isEditing {
          Text("\(value) \(unit)")
            .font(.title3)
            .fontWeight(.semibold)
            .foregroundColor(.primary)
        }
      }

      Spacer()

      if isEditing {
        HStack {
          TextField("", value: $value, format: .number)
            .keyboardType(.numberPad)
            .textFieldStyle(.roundedBorder)
            .frame(width: 80)
            .multilineTextAlignment(.trailing)
          Text(unit)
            .foregroundColor(.secondary)
        }
      }
    }
  }
}

@MainActor
class NutritionTargetsViewModel: ObservableObject {
  @Published var targets: NutritionTargetsDTO?
  @Published var isDefault = false
  @Published var isLoading = false
  @Published var isSaving = false
  @Published var errorMessage: String?

  @Published var editedCalories = 2200
  @Published var editedProtein = 120
  @Published var editedFat = 70
  @Published var editedCarbs = 250
  @Published var editedCalcium = 800

  var isValid: Bool {
    editedCalories >= 800 && editedCalories <= 6000 && editedProtein >= 0 && editedProtein <= 400
      && editedFat >= 0 && editedFat <= 400 && editedCarbs >= 0 && editedCarbs <= 400
      && editedCalcium >= 0 && editedCalcium <= 5000
  }

  func loadTargets(profileId: UUID) {
    isLoading = true
    errorMessage = nil

    Task {
      do {
        let response = try await APIClient.shared.fetchNutritionTargets(profileId: profileId)
        targets = response.targets
        isDefault = response.isDefault

        // Populate edit fields
        editedCalories = response.targets.caloriesKcal
        editedProtein = response.targets.proteinG
        editedFat = response.targets.fatG
        editedCarbs = response.targets.carbsG
        editedCalcium = response.targets.calciumMg

        isLoading = false
      } catch {
        isLoading = false
        errorMessage = "Не удалось загрузить цели: \(error.localizedDescription)"
      }
    }
  }

  func saveTargets(profileId: UUID) async -> Bool {
    guard isValid else {
      errorMessage = "Некорректные значения"
      return false
    }

    isSaving = true
    errorMessage = nil

    do {
      let updated = try await APIClient.shared.upsertNutritionTargets(
        profileId: profileId,
        caloriesKcal: editedCalories,
        proteinG: editedProtein,
        fatG: editedFat,
        carbsG: editedCarbs,
        calciumMg: editedCalcium
      )

      targets = updated
      isDefault = false
      isSaving = false
      return true
    } catch {
      isSaving = false
      errorMessage = "Не удалось сохранить цели: \(error.localizedDescription)"
      return false
    }
  }
}
