import Foundation

struct ProfileDTO: Codable, Identifiable {
    let id: UUID
    let ownerUserId: String
    let type: String
    let name: String
    let createdAt: Date
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case ownerUserId = "owner_user_id"
        case type
        case name
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

struct ProfilesResponse: Codable {
    let profiles: [ProfileDTO]
}

struct CreateProfileRequest: Codable {
    let type: String
    let name: String
}

struct UpdateProfileRequest: Codable {
    let name: String
}

struct ErrorResponse: Codable {
    let error: ErrorDetail
}

struct ErrorDetail: Codable {
    let code: String
    let message: String
}
