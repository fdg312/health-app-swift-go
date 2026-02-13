import Foundation

struct MealPlanDTO: Codable, Identifiable {
  let id: String
  let profileId: String
  let title: String
  let isActive: Bool
  let fromDate: Date?
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id, title
    case profileId = "profile_id"
    case isActive = "is_active"
    case fromDate = "from_date"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }
}

struct MealPlanItemDTO: Codable, Identifiable {
  let id: String
  let profileId: String
  let planId: String
  let dayIndex: Int
  let mealSlot: String
  let title: String
  let notes: String
  let approxKcal: Int
  let approxProteinG: Int
  let approxFatG: Int
  let approxCarbsG: Int
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id, title, notes
    case profileId = "profile_id"
    case planId = "plan_id"
    case dayIndex = "day_index"
    case mealSlot = "meal_slot"
    case approxKcal = "approx_kcal"
    case approxProteinG = "approx_protein_g"
    case approxFatG = "approx_fat_g"
    case approxCarbsG = "approx_carbs_g"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }
}

struct GetMealPlanResponse: Codable {
  let plan: MealPlanDTO?
  let items: [MealPlanItemDTO]
}

struct MealPlanItemUpsertDTO: Codable {
  let dayIndex: Int
  let mealSlot: String
  let title: String
  let notes: String
  let approxKcal: Int
  let approxProteinG: Int
  let approxFatG: Int
  let approxCarbsG: Int

  enum CodingKeys: String, CodingKey {
    case title, notes
    case dayIndex = "day_index"
    case mealSlot = "meal_slot"
    case approxKcal = "approx_kcal"
    case approxProteinG = "approx_protein_g"
    case approxFatG = "approx_fat_g"
    case approxCarbsG = "approx_carbs_g"
  }
}

struct ReplaceMealPlanRequest: Codable {
  let profileId: String
  let title: String
  let items: [MealPlanItemUpsertDTO]

  enum CodingKeys: String, CodingKey {
    case title, items
    case profileId = "profile_id"
  }
}

struct GetTodayResponse: Codable {
  let date: String
  let items: [MealPlanItemDTO]
}
