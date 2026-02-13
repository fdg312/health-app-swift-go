//
//  WorkoutsDTO.swift
//  HealthHub
//
//  Created by HealthHub on 2024.
//

import Foundation

// MARK: - Plan

struct WorkoutPlanDTO: Codable, Identifiable {
  let id: UUID
  let profileId: UUID
  let title: String
  let goal: String
  let isActive: Bool
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case profileId = "profile_id"
    case title
    case goal
    case isActive = "is_active"
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }
}

// MARK: - Item

struct WorkoutItemDTO: Codable, Identifiable {
  let id: UUID
  let kind: String
  let timeMinutes: Int
  let daysMask: Int
  let durationMin: Int
  let intensity: String
  let note: String
  let details: [String: AnyCodable]?
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case kind
    case timeMinutes = "time_minutes"
    case daysMask = "days_mask"
    case durationMin = "duration_min"
    case intensity
    case note
    case details
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }

  var kindLocalized: String {
    switch kind {
    case "run": return "Бег"
    case "walk": return "Прогулка"
    case "strength": return "Силовая"
    case "morning": return "Утренняя зарядка"
    case "core": return "Кор"
    case "other": return "Другое"
    default: return kind
    }
  }

  var intensityLocalized: String {
    switch intensity {
    case "low": return "Низкая"
    case "medium": return "Средняя"
    case "high": return "Высокая"
    default: return intensity
    }
  }

  var timeFormatted: String {
    let hours = timeMinutes / 60
    let minutes = timeMinutes % 60
    return String(format: "%02d:%02d", hours, minutes)
  }

  var daysFormatted: String {
    var days: [String] = []
    let dayNames = ["Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"]
    for i in 0..<7 {
      if (daysMask & (1 << i)) != 0 {
        days.append(dayNames[i])
      }
    }
    return days.isEmpty ? "Нет дней" : days.joined(separator: ", ")
  }
}

// MARK: - Completion

struct WorkoutCompletionDTO: Codable, Identifiable {
  let id: UUID
  let date: String
  let planItemId: UUID
  let status: String
  let note: String
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case date
    case planItemId = "plan_item_id"
    case status
    case note
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }

  var statusLocalized: String {
    switch status {
    case "done": return "Выполнено"
    case "skipped": return "Пропущено"
    default: return status
    }
  }
}

// MARK: - Requests

struct UpsertWorkoutPlanRequest: Codable {
  let profileId: UUID
  let title: String
  let goal: String

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case title
    case goal
  }
}

struct WorkoutItemUpsert: Codable {
  let kind: String
  let timeMinutes: Int
  let daysMask: Int
  let durationMin: Int
  let intensity: String
  let note: String
  let details: [String: AnyCodable]?

  enum CodingKeys: String, CodingKey {
    case kind
    case timeMinutes = "time_minutes"
    case daysMask = "days_mask"
    case durationMin = "duration_min"
    case intensity
    case note
    case details
  }
}

struct ReplaceWorkoutPlanRequest: Codable {
  let profileId: UUID
  let title: String
  let goal: String
  let replace: Bool
  let items: [WorkoutItemUpsert]

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case title
    case goal
    case replace
    case items
  }
}

struct UpsertWorkoutCompletionRequest: Codable {
  let profileId: UUID
  let date: String
  let planItemId: UUID
  let status: String
  let note: String

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case date
    case planItemId = "plan_item_id"
    case status
    case note
  }
}

struct MarkWorkoutDoneRequest: Codable {
  let date: String
  let planItemId: UUID
  let status: String

  enum CodingKeys: String, CodingKey {
    case date
    case planItemId = "plan_item_id"
    case status
  }
}

struct UpsertMorningRequest: Codable {
  let profileId: UUID
  let date: String
  let score: Int

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case date
    case score
  }
}

// MARK: - Responses

struct GetWorkoutPlanResponse: Codable {
  let plan: WorkoutPlanDTO?
  let items: [WorkoutItemDTO]
}

struct ReplaceWorkoutPlanResponse: Codable {
  let plan: WorkoutPlanDTO
  let items: [WorkoutItemDTO]
}

struct WorkoutTodayResponse: Codable {
  let date: String
  let profileId: UUID
  let planned: [WorkoutItemDTO]
  let completions: [WorkoutCompletionDTO]
  let actualWorkouts: [WorkoutSessionDTO]
  let isDone: Bool

  enum CodingKeys: String, CodingKey {
    case date
    case profileId = "profile_id"
    case planned
    case completions
    case actualWorkouts = "actual_workouts"
    case isDone = "is_done"
  }
}

struct WorkoutSessionDTO: Codable {
  let start: Date
  let end: Date
  let label: String
  let caloriesKcal: Int?

  enum CodingKeys: String, CodingKey {
    case start
    case end
    case label
    case caloriesKcal = "calories_kcal"
  }
}

struct ListWorkoutCompletionsResponse: Codable {
  let completions: [WorkoutCompletionDTO]
}

// MARK: - Helper for AnyCodable

struct AnyCodable: Codable {
  let value: Any

  init(_ value: Any) {
    self.value = value
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.singleValueContainer()
    if let int = try? container.decode(Int.self) {
      value = int
    } else if let double = try? container.decode(Double.self) {
      value = double
    } else if let string = try? container.decode(String.self) {
      value = string
    } else if let bool = try? container.decode(Bool.self) {
      value = bool
    } else if let array = try? container.decode([AnyCodable].self) {
      value = array.map(\.value)
    } else if let dict = try? container.decode([String: AnyCodable].self) {
      value = dict.mapValues(\.value)
    } else {
      value = NSNull()
    }
  }

  func encode(to encoder: Encoder) throws {
    var container = encoder.singleValueContainer()
    switch value {
    case let int as Int:
      try container.encode(int)
    case let double as Double:
      try container.encode(double)
    case let string as String:
      try container.encode(string)
    case let bool as Bool:
      try container.encode(bool)
    case let array as [Any]:
      try container.encode(array.map { AnyCodable($0) })
    case let dict as [String: Any]:
      try container.encode(dict.mapValues { AnyCodable($0) })
    default:
      try container.encodeNil()
    }
  }
}
