//
//  NutritionDTO.swift
//  HealthHub
//
//  Created by HealthHub on 2024.
//

import Foundation

// MARK: - Nutrition Targets

struct NutritionTargetsDTO: Codable, Identifiable {
  let profileId: UUID
  let caloriesKcal: Int
  let proteinG: Int
  let fatG: Int
  let carbsG: Int
  let calciumMg: Int
  let createdAt: Date
  let updatedAt: Date

  var id: UUID { profileId }

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case caloriesKcal = "calories_kcal"
    case proteinG = "protein_g"
    case fatG = "fat_g"
    case carbsG = "carbs_g"
    case calciumMg = "calcium_mg"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }
}

// MARK: - Nutrition Progress

struct NutritionProgressDTO: Codable {
  let actualCaloriesKcal: Int
  let actualProteinG: Int
  let actualFatG: Int
  let actualCarbsG: Int
  let actualCalciumMg: Int
  let caloriesPercent: Int
  let proteinPercent: Int
  let fatPercent: Int
  let carbsPercent: Int
  let calciumPercent: Int

  enum CodingKeys: String, CodingKey {
    case actualCaloriesKcal = "actual_calories_kcal"
    case actualProteinG = "actual_protein_g"
    case actualFatG = "actual_fat_g"
    case actualCarbsG = "actual_carbs_g"
    case actualCalciumMg = "actual_calcium_mg"
    case caloriesPercent = "calories_percent"
    case proteinPercent = "protein_percent"
    case fatPercent = "fat_percent"
    case carbsPercent = "carbs_percent"
    case calciumPercent = "calcium_percent"
  }
}

// MARK: - Requests

struct UpsertNutritionTargetsRequest: Codable {
  let profileId: UUID
  let caloriesKcal: Int
  let proteinG: Int
  let fatG: Int
  let carbsG: Int
  let calciumMg: Int

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case caloriesKcal = "calories_kcal"
    case proteinG = "protein_g"
    case fatG = "fat_g"
    case carbsG = "carbs_g"
    case calciumMg = "calcium_mg"
  }
}

// MARK: - Responses

struct GetNutritionTargetsResponse: Codable {
  let targets: NutritionTargetsDTO
  let isDefault: Bool

  enum CodingKeys: String, CodingKey {
    case targets
    case isDefault = "is_default"
  }
}
