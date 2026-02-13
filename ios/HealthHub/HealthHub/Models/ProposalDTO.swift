import Foundation

struct ProposalDTO: Codable, Identifiable {
  let id: UUID
  let profileId: UUID
  var status: String
  let kind: String
  let title: String
  let summary: String
  let payload: [String: JSONValue]
  let createdAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case profileId = "profile_id"
    case status
    case kind
    case title
    case summary
    case payload
    case createdAt = "created_at"
  }
}

struct ListProposalsResponse: Codable {
  let proposals: [ProposalDTO]
}

struct ApplyProposalResponse: Codable {
  let status: String
  let applied: AppliedProposalDTO?
}

struct AppliedProposalDTO: Codable {
  let settings: SettingsDTO?
  let schedulesCreated: Int?
  let workoutItemsCreated: Int?
  let nutritionTargetsUpdated: Bool?

  enum CodingKeys: String, CodingKey {
    case settings
    case schedulesCreated = "schedules_created"
    case workoutItemsCreated = "workout_items_created"
    case nutritionTargetsUpdated = "nutrition_targets_updated"
  }
}

struct RejectProposalResponse: Codable {
  let status: String
}

enum JSONValue: Codable {
  case string(String)
  case int(Int)
  case double(Double)
  case bool(Bool)
  case object([String: JSONValue])
  case array([JSONValue])
  case null

  init(from decoder: Decoder) throws {
    let container = try decoder.singleValueContainer()
    if container.decodeNil() {
      self = .null
    } else if let value = try? container.decode(String.self) {
      self = .string(value)
    } else if let value = try? container.decode(Int.self) {
      self = .int(value)
    } else if let value = try? container.decode(Double.self) {
      self = .double(value)
    } else if let value = try? container.decode(Bool.self) {
      self = .bool(value)
    } else if let value = try? container.decode([String: JSONValue].self) {
      self = .object(value)
    } else if let value = try? container.decode([JSONValue].self) {
      self = .array(value)
    } else {
      throw DecodingError.typeMismatch(
        JSONValue.self,
        DecodingError.Context(
          codingPath: decoder.codingPath,
          debugDescription: "Unsupported payload value type"
        )
      )
    }
  }

  func encode(to encoder: Encoder) throws {
    var container = encoder.singleValueContainer()
    switch self {
    case .string(let value):
      try container.encode(value)
    case .int(let value):
      try container.encode(value)
    case .double(let value):
      try container.encode(value)
    case .bool(let value):
      try container.encode(value)
    case .object(let value):
      try container.encode(value)
    case .array(let value):
      try container.encode(value)
    case .null:
      try container.encodeNil()
    }
  }
}
