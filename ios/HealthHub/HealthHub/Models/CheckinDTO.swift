import Foundation

// MARK: - Checkin Models

struct CheckinDTO: Codable, Identifiable {
    let id: UUID
    let profileId: UUID
    let date: String // YYYY-MM-DD
    let type: CheckinType
    let score: Int // 1-5
    let tags: [String]?
    let note: String?
    let createdAt: Date
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case profileId = "profile_id"
        case date
        case type
        case score
        case tags
        case note
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

enum CheckinType: String, Codable {
    case morning
    case evening
}

struct UpsertCheckinRequest: Codable {
    let profileId: UUID
    let date: String
    let type: CheckinType
    let score: Int
    let tags: [String]?
    let note: String?

    enum CodingKeys: String, CodingKey {
        case profileId = "profile_id"
        case date
        case type
        case score
        case tags
        case note
    }
}

struct CheckinsResponse: Codable {
    let checkins: [CheckinDTO]
}
