import Foundation

// MARK: - Feed Models

struct FeedDayResponse: Codable {
  let date: String  // YYYY-MM-DD
  let profileId: UUID
  let daily: DailyAggregate?
  let sessions: Sessions?
  let checkins: DayCheckins
  let nutritionTargets: NutritionTargetsDTO?
  let nutritionProgress: NutritionProgressDTO?
  let missingFields: [String]
  let mealToday: [MealPlanItemDTO]?
  let mealPlanTitle: String?
  let foodPrefsCount: Int?

  enum CodingKeys: String, CodingKey {
    case date
    case profileId = "profile_id"
    case daily, sessions
    case checkins
    case nutritionTargets = "nutrition_targets"
    case nutritionProgress = "nutrition_progress"
    case missingFields = "missing_fields"
    case mealToday = "meal_today"
    case mealPlanTitle = "meal_plan_title"
    case foodPrefsCount = "food_prefs_count"
  }
}

struct DayCheckins: Codable {
  let morning: CheckinSummary?
  let evening: CheckinSummary?
}

struct CheckinSummary: Codable, Identifiable {
  let id: UUID
  let score: Int
  let tags: [String]?
  let note: String?
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case score
    case tags
    case note
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }
}
