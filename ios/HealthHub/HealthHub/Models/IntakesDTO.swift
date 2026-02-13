//
//  IntakesDTO.swift
//  HealthHub
//

import Foundation

// MARK: - Supplements

struct SupplementDTO: Codable, Identifiable {
  let id: UUID
  let profileId: UUID
  let name: String
  let notes: String?
  let components: [SupplementComponentDTO]
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case profileId = "profile_id"
    case name
    case notes
    case components
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    id = try container.decode(UUID.self, forKey: .id)
    profileId = try container.decode(UUID.self, forKey: .profileId)
    name = try container.decode(String.self, forKey: .name)
    notes = try container.decodeIfPresent(String.self, forKey: .notes)
    components =
      try container.decodeIfPresent([SupplementComponentDTO].self, forKey: .components) ?? []
    createdAt = try container.decode(Date.self, forKey: .createdAt)
    updatedAt = try container.decode(Date.self, forKey: .updatedAt)
  }
}

struct SupplementComponentDTO: Codable, Identifiable {
  let id: UUID
  let nutrientKey: String
  let hkIdentifier: String?
  let amount: Double
  let unit: String

  enum CodingKeys: String, CodingKey {
    case id
    case nutrientKey = "nutrient_key"
    case hkIdentifier = "hk_identifier"
    case amount
    case unit
  }
}

struct CreateSupplementRequest: Codable {
  let profileId: UUID
  let name: String
  let notes: String?
  let components: [ComponentInput]

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case name
    case notes
    case components
  }
}

struct ComponentInput: Codable {
  let nutrientKey: String
  let hkIdentifier: String?
  let amount: Double
  let unit: String

  enum CodingKeys: String, CodingKey {
    case nutrientKey = "nutrient_key"
    case hkIdentifier = "hk_identifier"
    case amount
    case unit
  }
}

struct SupplementsResponse: Codable {
  let supplements: [SupplementDTO]

  private enum CodingKeys: String, CodingKey {
    case supplements
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    supplements = try container.decode([SupplementDTO].self, forKey: .supplements)
  }
}

// MARK: - Intakes

struct AddWaterRequest: Codable {
  let profileId: UUID
  let takenAt: Date
  let amountMl: Int

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case takenAt = "taken_at"
    case amountMl = "amount_ml"
  }
}

struct UpsertSupplementIntakeRequest: Codable {
  let profileId: UUID
  let supplementId: UUID
  let date: String  // YYYY-MM-DD
  let status: String  // "taken" or "skipped"

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case supplementId = "supplement_id"
    case date
    case status
  }
}

struct IntakesDailyResponse: Codable {
  let date: String
  let waterTotalMl: Int
  let waterEntries: [WaterIntakeDTO]
  let supplements: [SupplementDailyStatus]

  enum CodingKeys: String, CodingKey {
    case date
    case waterTotalMl = "water_total_ml"
    case waterEntries = "water_entries"
    case supplements
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    date = try container.decode(String.self, forKey: .date)
    waterTotalMl = try container.decode(Int.self, forKey: .waterTotalMl)
    waterEntries = try container.decodeIfPresent([WaterIntakeDTO].self, forKey: .waterEntries) ?? []
    supplements = try container.decode([SupplementDailyStatus].self, forKey: .supplements)
  }
}

struct WaterIntakeDTO: Codable, Identifiable {
  let id: UUID
  let takenAt: Date
  let amountMl: Int
  let createdAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case takenAt = "taken_at"
    case amountMl = "amount_ml"
    case createdAt = "created_at"
  }
}

struct SupplementDailyStatus: Codable, Identifiable {
  let supplementId: UUID
  let name: String
  let status: String  // "taken", "skipped", "none"

  var id: UUID { supplementId }

  enum CodingKeys: String, CodingKey {
    case supplementId = "supplement_id"
    case name
    case status
  }
}
