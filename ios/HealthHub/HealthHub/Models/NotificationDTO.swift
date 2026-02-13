import Foundation

struct NotificationDTO: Codable, Identifiable {
    let id: UUID
    let profileId: UUID
    let kind: String
    let title: String
    let body: String
    let sourceDate: String?
    let severity: String
    let createdAt: Date
    let readAt: Date?

    enum CodingKeys: String, CodingKey {
        case id
        case profileId = "profile_id"
        case kind, title, body
        case sourceDate = "source_date"
        case severity
        case createdAt = "created_at"
        case readAt = "read_at"
    }
}

struct InboxListResponse: Codable {
    let notifications: [NotificationDTO]
}

struct UnreadCountResponse: Codable {
    let unread: Int
}

struct MarkReadRequest: Codable {
    let profileId: UUID
    let ids: [UUID]

    enum CodingKeys: String, CodingKey {
        case profileId = "profile_id"
        case ids
    }
}

struct MarkReadResponse: Codable {
    let marked: Int
}

struct MarkAllReadRequest: Codable {
    let profileId: UUID

    enum CodingKeys: String, CodingKey {
        case profileId = "profile_id"
    }
}

struct MarkAllReadResponse: Codable {
    let marked: Int
}

struct GenerateNotificationsRequest: Codable {
    let profileId: UUID
    let date: String
    let clientTimeZone: String
    let now: Date
    let thresholds: GenerateThresholds

    enum CodingKeys: String, CodingKey {
        case profileId = "profile_id"
        case date
        case clientTimeZone = "client_time_zone"
        case now
        case thresholds
    }
}

struct GenerateThresholds: Codable {
    let sleepMinMinutes: Int
    let stepsMin: Int
    let activeEnergyMinKcal: Int

    enum CodingKeys: String, CodingKey {
        case sleepMinMinutes = "sleep_min_minutes"
        case stepsMin = "steps_min"
        case activeEnergyMinKcal = "active_energy_min_kcal"
    }
}

struct GenerateNotificationsResponse: Codable {
    let created: Int
    let updated: Int
    let skipped: Int
}
