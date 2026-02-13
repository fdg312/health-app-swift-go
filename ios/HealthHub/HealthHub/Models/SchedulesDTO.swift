import Foundation

struct ScheduleDTO: Codable, Identifiable {
  let id: UUID
  let supplementId: UUID
  let timeMinutes: Int
  let daysMask: Int
  let isEnabled: Bool
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case supplementId = "supplement_id"
    case timeMinutes = "time_minutes"
    case daysMask = "days_mask"
    case isEnabled = "is_enabled"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }
}

struct ListSchedulesResponse: Codable {
  let schedules: [ScheduleDTO]
}

struct UpsertScheduleRequest: Codable {
  let profileId: UUID
  let supplementId: UUID
  let timeMinutes: Int
  let daysMask: Int
  let isEnabled: Bool

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case supplementId = "supplement_id"
    case timeMinutes = "time_minutes"
    case daysMask = "days_mask"
    case isEnabled = "is_enabled"
  }
}

struct ReplaceScheduleItem: Codable {
  let supplementId: UUID
  let timeMinutes: Int
  let daysMask: Int
  let isEnabled: Bool

  enum CodingKeys: String, CodingKey {
    case supplementId = "supplement_id"
    case timeMinutes = "time_minutes"
    case daysMask = "days_mask"
    case isEnabled = "is_enabled"
  }
}

struct ReplaceSchedulesRequest: Codable {
  let profileId: UUID
  let schedules: [ReplaceScheduleItem]
  let replace: Bool

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case schedules
    case replace
  }
}
