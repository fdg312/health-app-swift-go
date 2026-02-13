import Foundation

struct SettingsDTO: Codable {
    var timeZone: String?
    var quietStartMinutes: Int?
    var quietEndMinutes: Int?

    var notificationsMaxPerDay: Int

    var minSleepMinutes: Int
    var minSteps: Int
    var minActiveEnergyKcal: Int

    var morningCheckinTimeMinutes: Int
    var eveningCheckinTimeMinutes: Int
    var vitaminsTimeMinutes: Int

    enum CodingKeys: String, CodingKey {
        case timeZone = "time_zone"
        case quietStartMinutes = "quiet_start_minutes"
        case quietEndMinutes = "quiet_end_minutes"
        case notificationsMaxPerDay = "notifications_max_per_day"
        case minSleepMinutes = "min_sleep_minutes"
        case minSteps = "min_steps"
        case minActiveEnergyKcal = "min_active_energy_kcal"
        case morningCheckinTimeMinutes = "morning_checkin_time_minutes"
        case eveningCheckinTimeMinutes = "evening_checkin_time_minutes"
        case vitaminsTimeMinutes = "vitamins_time_minutes"
    }
}

struct SettingsResponse: Codable {
    let settings: SettingsDTO
    let isDefault: Bool

    enum CodingKeys: String, CodingKey {
        case settings
        case isDefault = "is_default"
    }
}

typealias PutSettingsRequest = SettingsDTO
