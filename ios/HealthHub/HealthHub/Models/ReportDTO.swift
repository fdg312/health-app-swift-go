import Foundation

// MARK: - Report Models

struct ReportDTO: Codable, Identifiable {
    let id: UUID
    let profileId: UUID
    let format: String     // "pdf" or "csv"
    let from: String       // YYYY-MM-DD
    let to: String         // YYYY-MM-DD
    let downloadUrl: String
    let sizeBytes: Int64
    let status: String     // "ready" or "failed"
    let createdAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case profileId = "profile_id"
        case format, from, to
        case downloadUrl = "download_url"
        case sizeBytes = "size_bytes"
        case status
        case createdAt = "created_at"
    }
}

struct ReportsListResponse: Codable {
    let reports: [ReportDTO]
}

struct CreateReportRequest: Codable {
    let profileId: UUID
    let from: String
    let to: String
    let format: String

    enum CodingKeys: String, CodingKey {
        case profileId = "profile_id"
        case from, to, format
    }
}
