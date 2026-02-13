import Foundation

struct FoodPrefDTO: Codable, Identifiable {
  let id: String
  let profileId: String
  let name: String
  let tags: [String]
  let kcalPer100g: Int
  let proteinGPer100g: Int
  let fatGPer100g: Int
  let carbsGPer100g: Int
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id, name, tags
    case profileId = "profile_id"
    case kcalPer100g = "kcal_per_100g"
    case proteinGPer100g = "protein_g_per_100g"
    case fatGPer100g = "fat_g_per_100g"
    case carbsGPer100g = "carbs_g_per_100g"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }
}

struct ListFoodPrefsResponse: Codable {
  let items: [FoodPrefDTO]
  let total: Int
  let limit: Int
  let offset: Int
}

struct UpsertFoodPrefRequest: Codable {
  let profileId: String
  let name: String
  let tags: [String]
  let kcalPer100g: Int
  let proteinGPer100g: Int
  let fatGPer100g: Int
  let carbsGPer100g: Int

  enum CodingKeys: String, CodingKey {
    case name, tags
    case profileId = "profile_id"
    case kcalPer100g = "kcal_per_100g"
    case proteinGPer100g = "protein_g_per_100g"
    case fatGPer100g = "fat_g_per_100g"
    case carbsGPer100g = "carbs_g_per_100g"
  }
}
