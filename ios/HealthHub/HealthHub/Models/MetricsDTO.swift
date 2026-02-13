import Foundation

// MARK: - Sync Batch

struct SyncBatchRequest: Codable {
    let profileId: UUID
    let clientTimeZone: String?
    let daily: [DailyAggregate]?
    let hourly: [HourlyBucket]?
    let sessions: Sessions?

    enum CodingKeys: String, CodingKey {
        case profileId = "profile_id"
        case clientTimeZone = "client_time_zone"
        case daily, hourly, sessions
    }
}

struct SyncBatchResponse: Codable {
    let status: String
    let upsertedDaily: Int
    let upsertedHourly: Int
    let insertedSleepSegments: Int
    let insertedWorkouts: Int

    enum CodingKeys: String, CodingKey {
        case status
        case upsertedDaily = "upserted_daily"
        case upsertedHourly = "upserted_hourly"
        case insertedSleepSegments = "inserted_sleep_segments"
        case insertedWorkouts = "inserted_workouts"
    }
}

// MARK: - Daily Aggregate

struct DailyAggregate: Codable {
    let date: String // YYYY-MM-DD
    let sleep: SleepDaily?
    let activity: ActivityDaily?
    let body: BodyDaily?
    let heart: HeartDaily?
    let nutrition: NutritionDaily?
    let intakes: IntakesDaily?
    let temperature: TemperatureDaily?
}

struct SleepDaily: Codable {
    let totalMinutes: Int?
    let stages: SleepStages?

    enum CodingKeys: String, CodingKey {
        case totalMinutes = "total_minutes"
        case stages
    }
}

struct SleepStages: Codable {
    let rem: Int
    let deep: Int
    let core: Int
    let awake: Int
}

struct ActivityDaily: Codable {
    let steps: Int?
    let activeEnergyKcal: Int?
    let exerciseMin: Int?
    let standHours: Int?
    let distanceKm: Double?

    enum CodingKeys: String, CodingKey {
        case steps
        case activeEnergyKcal = "active_energy_kcal"
        case exerciseMin = "exercise_min"
        case standHours = "stand_hours"
        case distanceKm = "distance_km"
    }
}

struct BodyDaily: Codable {
    let weightKgLast: Double?
    let bmi: Double?
    let bodyFatPct: Double?

    enum CodingKeys: String, CodingKey {
        case weightKgLast = "weight_kg_last"
        case bmi
        case bodyFatPct = "body_fat_pct"
    }
}

struct HeartDaily: Codable {
    let restingHrBpm: Int?

    enum CodingKeys: String, CodingKey {
        case restingHrBpm = "resting_hr_bpm"
    }
}

struct NutritionDaily: Codable {
    let energyKcal: Int
    let proteinG: Int
    let fatG: Int
    let carbsG: Int
    let calciumMg: Int

    enum CodingKeys: String, CodingKey {
        case energyKcal = "energy_kcal"
        case proteinG = "protein_g"
        case fatG = "fat_g"
        case carbsG = "carbs_g"
        case calciumMg = "calcium_mg"
    }
}

struct IntakesDaily: Codable {
    let waterMl: Int
    let vitaminsTaken: [String]?

    enum CodingKeys: String, CodingKey {
        case waterMl = "water_ml"
        case vitaminsTaken = "vitamins_taken"
    }
}

struct TemperatureDaily: Codable {
    let wristCAvg: Double
    let wristCMin: Double?
    let wristCMax: Double?

    enum CodingKeys: String, CodingKey {
        case wristCAvg = "wrist_c_avg"
        case wristCMin = "wrist_c_min"
        case wristCMax = "wrist_c_max"
    }
}

// MARK: - Hourly Bucket

struct HourlyBucket: Codable {
    let hour: Date
    let steps: Int?
    let hr: HRData?
}

struct HRData: Codable {
    let min: Int
    let max: Int
    let avg: Int
}

// MARK: - Sessions

struct Sessions: Codable {
    let sleepSegments: [SleepSegment]?
    let workouts: [WorkoutSession]?

    enum CodingKeys: String, CodingKey {
        case sleepSegments = "sleep_segments"
        case workouts
    }
}

struct SleepSegment: Codable {
    let start: Date
    let end: Date
    let stage: String // "rem"|"deep"|"core"|"awake"
}

struct WorkoutSession: Codable {
    let start: Date
    let end: Date
    let label: String
    let caloriesKcal: Int?

    enum CodingKeys: String, CodingKey {
        case start, end, label
        case caloriesKcal = "calories_kcal"
    }
}

// MARK: - Metrics Responses

struct DailyMetricsResponse: Codable {
    let daily: [DailyAggregate]
}

struct HourlyMetricsResponse: Codable {
    let hourly: [HourlyBucket]
}
