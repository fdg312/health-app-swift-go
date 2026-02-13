import Foundation

// MARK: - API Errors

enum APIError: LocalizedError, Equatable {
  case invalidURL
  case invalidResponse
  case unauthorized
  case rateLimited
  case decodingError
  case badResponse(Int)
  case serverError(String)

  var errorDescription: String? {
    switch self {
    case .invalidURL: return "Invalid URL"
    case .invalidResponse: return "Invalid server response"
    case .unauthorized: return "Session expired"
    case .rateLimited: return "Too many requests, please try again later"
    case .decodingError: return "Failed to decode response"
    case .badResponse(let status): return "Bad server response (\(status))"
    case .serverError(let msg): return msg
    }
  }

  var uiCode: String {
    switch self {
    case .unauthorized:
      return "unauthorized"
    case .rateLimited:
      return "rate_limited"
    case .serverError(let code):
      return code
    case .badResponse:
      return "bad_response"
    case .invalidURL:
      return "invalid_url"
    case .invalidResponse, .decodingError:
      return "bad_response"
    }
  }
}

struct APIErrorResponse: Codable {
  let error: APIErrorDetail
}

struct APIErrorDetail: Codable {
  let code: String
  let message: String
}

struct EmptyResponse: Codable {}

struct DevAuthResponse: Codable {
  let accessToken: String
  let tokenType: String
  let expiresIn: Int

  enum CodingKeys: String, CodingKey {
    case accessToken = "access_token"
    case tokenType = "token_type"
    case expiresIn = "expires_in"
  }
}

struct SIWAAuthResponse: Codable {
  let accessToken: String
  let tokenType: String
  let expiresIn: Int
  let userID: String

  enum CodingKeys: String, CodingKey {
    case accessToken = "access_token"
    case tokenType = "token_type"
    case expiresIn = "expires_in"
    case userID = "user_id"
  }
}

struct EmailOTPRequestResponse: Codable {
  let status: String
  let debugCode: String?

  enum CodingKeys: String, CodingKey {
    case status
    case debugCode = "debug_code"
  }
}

struct EmailOTPVerifyResponse: Codable {
  let accessToken: String
  let tokenType: String
  let expiresIn: Int
  let userID: String

  enum CodingKeys: String, CodingKey {
    case accessToken = "access_token"
    case tokenType = "token_type"
    case expiresIn = "expires_in"
    case userID = "user_id"
  }
}

private struct SIWALoginRequest: Encodable {
  let identityToken: String
  let user: String?
  let email: String?
  let fullName: String?

  enum CodingKeys: String, CodingKey {
    case identityToken = "identity_token"
    case user
    case email
    case fullName = "full_name"
  }
}

private struct EmailOTPRequestPayload: Encodable {
  let email: String
}

private struct EmailOTPVerifyPayload: Encodable {
  let email: String
  let code: String
}

// MARK: - API Client

class APIClient {
  static let shared = APIClient()

  private let baseURL: String
  private let session: URLSession
  private let decoder: JSONDecoder
  private let errorDecoder: JSONDecoder

  private init() {
    self.baseURL = AppConfig.apiBaseURL
    let config = URLSessionConfiguration.default
    config.timeoutIntervalForRequest = 30
    self.session = URLSession(configuration: config)
    self.decoder = JSONDecoder()
    self.decoder.dateDecodingStrategy = .iso8601
    self.errorDecoder = JSONDecoder()
  }

  // MARK: - Centralized Request Helpers

  /// Builds a URLRequest with Authorization header if token is available.
  private func makeRequest(url: URL, method: String = "GET") -> URLRequest {
    var request = URLRequest(url: url)
    request.httpMethod = method
    if let token = AuthManager.shared.accessToken {
      request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
    }
    return request
  }

  /// Builds a JSON URLRequest with Authorization header.
  private func makeJSONRequest(url: URL, method: String, body: some Encodable) throws -> URLRequest
  {
    var request = makeRequest(url: url, method: method)
    request.setValue("application/json", forHTTPHeaderField: "Content-Type")
    let encoder = JSONEncoder()
    encoder.dateEncodingStrategy = .iso8601
    request.httpBody = try encoder.encode(body)
    return request
  }

  /// Performs a data request and returns payload + HTTP response.
  private func performRequest(_ request: URLRequest) async throws -> (Data, HTTPURLResponse) {
    let (data, response) = try await session.data(for: request)

    guard let httpResponse = response as? HTTPURLResponse else {
      throw APIError.invalidResponse
    }

    return (data, httpResponse)
  }

  private func decodeAPIErrorResponse(from data: Data) -> APIErrorResponse? {
    guard !data.isEmpty else { return nil }
    return try? errorDecoder.decode(APIErrorResponse.self, from: data)
  }

  private func isEmptyBody(_ data: Data) -> Bool {
    if data.isEmpty { return true }
    guard let text = String(data: data, encoding: .utf8) else {
      return false
    }
    return text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
  }

  private func bodyPreview(_ data: Data, maxBytes: Int = 4096) -> String {
    let prefix = data.prefix(maxBytes)
    let suffix = data.count > maxBytes ? "… [truncated \(data.count - maxBytes) bytes]" : ""
    if let text = String(data: prefix, encoding: .utf8) {
      return text + suffix
    }
    return "<non-utf8 body, \(data.count) bytes>"
  }

  private func logResponseDetails(
    _ request: URLRequest,
    _ response: HTTPURLResponse,
    _ data: Data,
    reason: String,
    decodeError: Error? = nil,
    apiError: APIErrorResponse? = nil
  ) {
    let url = request.url?.absoluteString ?? "<unknown>"
    print("[API] \(reason) url=\(url) status=\(response.statusCode)")
    if let decodeError {
      print("[API] decode_error=\(decodeError)")
    }
    if let apiError {
      print("[API] server_error code=\(apiError.error.code) message=\(apiError.error.message)")
    }
    print("[API] raw_body=\(bodyPreview(data))")
  }

  private func mapResponseError(statusCode: Int, apiError: APIErrorResponse?) -> APIError {
    if statusCode == 401 || apiError?.error.code == "unauthorized" {
      return .unauthorized
    }
    if statusCode == 429 {
      return .rateLimited
    }
    if let code = apiError?.error.code {
      return .serverError(code)
    }
    return .badResponse(statusCode)
  }

  private func ensureExpectedStatus(
    _ request: URLRequest,
    _ response: HTTPURLResponse,
    _ data: Data,
    expectedStatus: Int
  ) throws {
    guard response.statusCode == expectedStatus else {
      let apiError = decodeAPIErrorResponse(from: data)
      logResponseDetails(request, response, data, reason: "unexpected_status", apiError: apiError)
      throw mapResponseError(statusCode: response.statusCode, apiError: apiError)
    }
  }

  /// Convenience: perform request and decode JSON response.
  private func performAndDecode<T: Decodable>(_ request: URLRequest, expectedStatus: Int = 200)
    async throws -> T
  {
    let (data, httpResponse) = try await performRequest(request)

    guard (200...299).contains(httpResponse.statusCode) else {
      let apiError = decodeAPIErrorResponse(from: data)
      logResponseDetails(request, httpResponse, data, reason: "non_2xx", apiError: apiError)
      throw mapResponseError(statusCode: httpResponse.statusCode, apiError: apiError)
    }

    if httpResponse.statusCode != expectedStatus {
      logResponseDetails(request, httpResponse, data, reason: "unexpected_2xx_status")
      throw APIError.badResponse(httpResponse.statusCode)
    }

    if httpResponse.statusCode == 204 || isEmptyBody(data) {
      if T.self == EmptyResponse.self {
        return EmptyResponse() as! T
      }
      logResponseDetails(request, httpResponse, data, reason: "empty_body")
      throw APIError.badResponse(httpResponse.statusCode)
    }

    do {
      return try decoder.decode(T.self, from: data)
    } catch {
      let apiError = decodeAPIErrorResponse(from: data)
      logResponseDetails(
        request,
        httpResponse,
        data,
        reason: "decode_failure_\(T.self)",
        decodeError: error,
        apiError: apiError
      )
      if let apiError {
        throw mapResponseError(statusCode: httpResponse.statusCode, apiError: apiError)
      }
      throw APIError.badResponse(httpResponse.statusCode)
    }
  }

  // MARK: - Health Check

  func healthCheck() async throws -> Bool {
    let url = URL(string: "\(baseURL)/healthz")!
    let request = makeRequest(url: url)
    let (data, httpResponse) = try await performRequest(request)

    guard httpResponse.statusCode == 200 else { return false }

    let json = try JSONSerialization.jsonObject(with: data) as? [String: String]
    return json?["status"] == "ok"
  }

  // MARK: - Auth

  func loginDev() async throws -> DevAuthResponse {
    let url = URL(string: "\(baseURL)/v1/auth/dev")!
    var request = URLRequest(url: url)
    request.httpMethod = "POST"
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func loginSIWA(identityToken: String, user: String?, email: String?, fullName: String?)
    async throws -> SIWAAuthResponse
  {
    let url = URL(string: "\(baseURL)/v1/auth/siwa")!
    let body = SIWALoginRequest(
      identityToken: identityToken,
      user: user,
      email: email,
      fullName: fullName
    )
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func requestEmailOTP(email: String) async throws -> EmailOTPRequestResponse {
    let url = URL(string: "\(baseURL)/v1/auth/email/request")!
    let body = EmailOTPRequestPayload(email: email)
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func verifyEmailOTP(email: String, code: String) async throws -> EmailOTPVerifyResponse {
    let url = URL(string: "\(baseURL)/v1/auth/email/verify")!
    let body = EmailOTPVerifyPayload(email: email, code: code)
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  // MARK: - User Settings

  func fetchSettings() async throws -> SettingsResponse {
    let url = URL(string: "\(baseURL)/v1/settings")!
    let request = makeRequest(url: url)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func updateSettings(_ settings: SettingsDTO) async throws -> SettingsDTO {
    let url = URL(string: "\(baseURL)/v1/settings")!
    let request = try makeJSONRequest(url: url, method: "PUT", body: settings)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  // MARK: - Chat

  func listChatMessages(profileId: UUID, limit: Int = 50, before: String? = nil) async throws
    -> ListChatMessagesResponse
  {
    var urlString = "\(baseURL)/v1/chat/messages?profile_id=\(profileId.uuidString)&limit=\(limit)"
    if let before, !before.isEmpty {
      let encoded = before.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? before
      urlString += "&before=\(encoded)"
    }

    guard let url = URL(string: urlString) else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func sendChatMessage(profileId: UUID, content: String) async throws -> SendChatMessageResponse {
    guard let url = URL(string: "\(baseURL)/v1/chat/messages") else {
      throw APIError.invalidURL
    }

    let body = SendChatMessageRequest(profileId: profileId, content: content)
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  // MARK: - AI Proposals

  func listProposals(profileId: UUID, status: String = "pending", limit: Int = 20) async throws
    -> [ProposalDTO]
  {
    var urlString = "\(baseURL)/v1/ai/proposals?profile_id=\(profileId.uuidString)&limit=\(limit)"
    if !status.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
      let encoded = status.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? status
      urlString += "&status=\(encoded)"
    }

    guard let url = URL(string: urlString) else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url)
    let response: ListProposalsResponse = try await performAndDecode(request, expectedStatus: 200)
    return response.proposals
  }

  func applyProposal(id: UUID) async throws -> ApplyProposalResponse {
    guard let url = URL(string: "\(baseURL)/v1/ai/proposals/\(id.uuidString)/apply") else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url, method: "POST")
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func rejectProposal(id: UUID) async throws -> RejectProposalResponse {
    guard let url = URL(string: "\(baseURL)/v1/ai/proposals/\(id.uuidString)/reject") else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url, method: "POST")
    return try await performAndDecode(request, expectedStatus: 200)
  }

  // MARK: - Profiles

  func listProfiles() async throws -> [ProfileDTO] {
    let url = URL(string: "\(baseURL)/v1/profiles")!
    let request = makeRequest(url: url)
    let response: ProfilesResponse = try await performAndDecode(request)
    return response.profiles
  }

  func createProfile(type: String, name: String) async throws -> ProfileDTO {
    let url = URL(string: "\(baseURL)/v1/profiles")!
    let body = CreateProfileRequest(type: type, name: name)
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    return try await performAndDecode(request, expectedStatus: 201)
  }

  func deleteProfile(id: UUID) async throws {
    let url = URL(string: "\(baseURL)/v1/profiles/\(id.uuidString)")!
    let request = makeRequest(url: url, method: "DELETE")
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 204)
  }

  // MARK: - Metrics

  func sendSyncBatch(request syncRequest: SyncBatchRequest) async throws -> SyncBatchResponse {
    let url = URL(string: "\(baseURL)/v1/sync/batch")!
    let request = try makeJSONRequest(url: url, method: "POST", body: syncRequest)
    return try await performAndDecode(request)
  }

  func fetchDailyMetrics(profileId: UUID, from: String, to: String) async throws -> [DailyAggregate]
  {
    let urlString =
      "\(baseURL)/v1/metrics/daily?profile_id=\(profileId.uuidString)&from=\(from)&to=\(to)"
    let url = URL(string: urlString)!
    let request = makeRequest(url: url)
    let response: DailyMetricsResponse = try await performAndDecode(request)
    return response.daily
  }

  func fetchHourlyMetrics(profileId: UUID, date: String, metric: String) async throws
    -> [HourlyBucket]
  {
    let urlString =
      "\(baseURL)/v1/metrics/hourly?profile_id=\(profileId.uuidString)&date=\(date)&metric=\(metric)"
    let url = URL(string: urlString)!
    let request = makeRequest(url: url)
    let response: HourlyMetricsResponse = try await performAndDecode(request)
    return response.hourly
  }

  // MARK: - Checkins

  func listCheckins(profileId: UUID, from: Date, to: Date) async throws -> CheckinsResponse {
    let dateFormatter = ISO8601DateFormatter()
    dateFormatter.formatOptions = [.withFullDate]

    let fromStr = dateFormatter.string(from: from)
    let toStr = dateFormatter.string(from: to)

    let urlString =
      "\(baseURL)/v1/checkins?profile_id=\(profileId.uuidString)&from=\(fromStr)&to=\(toStr)"
    guard let url = URL(string: urlString) else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url)
    return try await performAndDecode(request)
  }

  func upsertCheckin(_ upsertRequest: UpsertCheckinRequest) async throws -> CheckinDTO {
    guard let url = URL(string: "\(baseURL)/v1/checkins") else {
      throw APIError.invalidURL
    }

    let request = try makeJSONRequest(url: url, method: "POST", body: upsertRequest)
    return try await performAndDecode(request)
  }

  func deleteCheckin(id: UUID) async throws {
    guard let url = URL(string: "\(baseURL)/v1/checkins/\(id.uuidString)") else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url, method: "DELETE")
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 204)
  }

  // MARK: - Feed

  func fetchFeedDay(profileId: UUID, date: Date) async throws -> FeedDayResponse {
    let dateFormatter = ISO8601DateFormatter()
    dateFormatter.formatOptions = [.withFullDate]
    let dateStr = dateFormatter.string(from: date)

    let urlString = "\(baseURL)/v1/feed/day?profile_id=\(profileId.uuidString)&date=\(dateStr)"
    guard let url = URL(string: urlString) else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url)
    return try await performAndDecode(request)
  }

  // MARK: - Reports

  func createReport(profileId: UUID, from: String, to: String, format: String) async throws
    -> ReportDTO
  {
    let url = URL(string: "\(baseURL)/v1/reports")!
    let body = CreateReportRequest(profileId: profileId, from: from, to: to, format: format)
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    return try await performAndDecode(request, expectedStatus: 201)
  }

  func listReports(profileId: UUID, limit: Int = 10, offset: Int = 0) async throws -> [ReportDTO] {
    let urlString =
      "\(baseURL)/v1/reports?profile_id=\(profileId.uuidString)&limit=\(limit)&offset=\(offset)"
    let url = URL(string: urlString)!
    let request = makeRequest(url: url)
    let response: ReportsListResponse = try await performAndDecode(request)
    return response.reports
  }

  func deleteReport(reportId: UUID) async throws {
    let url = URL(string: "\(baseURL)/v1/reports/\(reportId.uuidString)")!
    let request = makeRequest(url: url, method: "DELETE")
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 204)
  }

  func downloadReport(report: ReportDTO) async throws -> URL {
    let downloadURL: URL

    if report.downloadUrl.hasPrefix("http://") || report.downloadUrl.hasPrefix("https://") {
      downloadURL = URL(string: report.downloadUrl)!
    } else {
      downloadURL = URL(string: "\(baseURL)/v1/reports/\(report.id.uuidString)/download")!
    }

    // Build request with auth
    var request = makeRequest(url: downloadURL)
    request.httpMethod = "GET"

    let (tempURL, response) = try await session.download(for: request)

    guard let httpResponse = response as? HTTPURLResponse else {
      throw APIError.invalidResponse
    }

    guard httpResponse.statusCode == 200 else {
      let data = (try? Data(contentsOf: tempURL)) ?? Data()
      let apiError = decodeAPIErrorResponse(from: data)
      logResponseDetails(
        request, httpResponse, data, reason: "download_non_2xx", apiError: apiError)
      throw mapResponseError(statusCode: httpResponse.statusCode, apiError: apiError)
    }

    let fileName = "report_\(report.from)_\(report.to).\(report.format)"
    let destinationURL = FileManager.default.temporaryDirectory.appendingPathComponent(fileName)

    // Remove existing file if present
    try? FileManager.default.removeItem(at: destinationURL)
    try FileManager.default.moveItem(at: tempURL, to: destinationURL)

    return destinationURL
  }

  // MARK: - Sources

  func uploadSourceImage(profileId: UUID, checkinId: UUID?, imageData: Data, title: String?)
    async throws -> SourceDTO
  {
    let url = URL(string: "\(baseURL)/v1/sources/image")!
    var request = makeRequest(url: url, method: "POST")

    // Build multipart form data
    let boundary = "Boundary-\(UUID().uuidString)"
    request.setValue(
      "multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")

    var body = Data()

    // Add profile_id
    body.append("--\(boundary)\r\n".data(using: .utf8)!)
    body.append("Content-Disposition: form-data; name=\"profile_id\"\r\n\r\n".data(using: .utf8)!)
    body.append("\(profileId.uuidString)\r\n".data(using: .utf8)!)

    // Add checkin_id if present
    if let checkinId = checkinId {
      body.append("--\(boundary)\r\n".data(using: .utf8)!)
      body.append("Content-Disposition: form-data; name=\"checkin_id\"\r\n\r\n".data(using: .utf8)!)
      body.append("\(checkinId.uuidString)\r\n".data(using: .utf8)!)
    }

    // Add title if present
    if let title = title {
      body.append("--\(boundary)\r\n".data(using: .utf8)!)
      body.append("Content-Disposition: form-data; name=\"title\"\r\n\r\n".data(using: .utf8)!)
      body.append("\(title)\r\n".data(using: .utf8)!)
    }

    // Add file
    body.append("--\(boundary)\r\n".data(using: .utf8)!)
    body.append(
      "Content-Disposition: form-data; name=\"file\"; filename=\"image.jpg\"\r\n".data(
        using: .utf8)!)
    body.append("Content-Type: image/jpeg\r\n\r\n".data(using: .utf8)!)
    body.append(imageData)
    body.append("\r\n".data(using: .utf8)!)

    body.append("--\(boundary)--\r\n".data(using: .utf8)!)

    request.httpBody = body

    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 201)
    return try decoder.decode(SourceDTO.self, from: data)
  }

  func createSourceLink(profileId: UUID, checkinId: UUID?, title: String?, url: String) async throws
    -> SourceDTO
  {
    let urlObj = URL(string: "\(baseURL)/v1/sources")!
    let body = CreateSourceRequest(
      profileId: profileId,
      kind: "link",
      title: title,
      text: nil,
      url: url,
      checkinId: checkinId
    )
    let request = try makeJSONRequest(url: urlObj, method: "POST", body: body)
    return try await performAndDecode(request, expectedStatus: 201)
  }

  func createSourceNote(profileId: UUID, checkinId: UUID?, title: String?, text: String)
    async throws -> SourceDTO
  {
    let urlObj = URL(string: "\(baseURL)/v1/sources")!
    let body = CreateSourceRequest(
      profileId: profileId,
      kind: "note",
      title: title,
      text: text,
      url: nil,
      checkinId: checkinId
    )
    let request = try makeJSONRequest(url: urlObj, method: "POST", body: body)
    return try await performAndDecode(request, expectedStatus: 201)
  }

  func listSources(
    profileId: UUID, query: String? = nil, checkinId: UUID? = nil, limit: Int = 50, offset: Int = 0
  ) async throws -> [SourceDTO] {
    var urlString =
      "\(baseURL)/v1/sources?profile_id=\(profileId.uuidString)&limit=\(limit)&offset=\(offset)"

    if let query = query, !query.isEmpty {
      urlString +=
        "&query=\(query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? query)"
    }

    if let checkinId = checkinId {
      urlString += "&checkin_id=\(checkinId.uuidString)"
    }

    let url = URL(string: urlString)!
    let request = makeRequest(url: url)
    let response: SourcesResponse = try await performAndDecode(request)
    return response.sources
  }

  func deleteSource(sourceId: UUID) async throws {
    let url = URL(string: "\(baseURL)/v1/sources/\(sourceId.uuidString)")!
    let request = makeRequest(url: url, method: "DELETE")
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 204)
  }

  // MARK: - Inbox/Notifications

  func fetchInbox(profileId: UUID, onlyUnread: Bool = false, limit: Int = 20, offset: Int = 0)
    async throws -> [NotificationDTO]
  {
    var urlString =
      "\(baseURL)/v1/inbox?profile_id=\(profileId.uuidString)&limit=\(limit)&offset=\(offset)"
    if onlyUnread {
      urlString += "&only_unread=true"
    }

    let url = URL(string: urlString)!
    let request = makeRequest(url: url)
    let response: InboxListResponse = try await performAndDecode(request)
    return response.notifications
  }

  func fetchUnreadCount(profileId: UUID) async throws -> Int {
    let urlString = "\(baseURL)/v1/inbox/unread-count?profile_id=\(profileId.uuidString)"
    let url = URL(string: urlString)!
    let request = makeRequest(url: url)
    let response: UnreadCountResponse = try await performAndDecode(request)
    return response.unread
  }

  func markNotificationsRead(profileId: UUID, ids: [UUID]) async throws -> Int {
    let url = URL(string: "\(baseURL)/v1/inbox/mark-read")!
    let body = MarkReadRequest(profileId: profileId, ids: ids)
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    let response: MarkReadResponse = try await performAndDecode(request)
    return response.marked
  }

  func markAllNotificationsRead(profileId: UUID) async throws -> Int {
    let url = URL(string: "\(baseURL)/v1/inbox/mark-all-read")!
    let body = MarkAllReadRequest(profileId: profileId)
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    let response: MarkAllReadResponse = try await performAndDecode(request)
    return response.marked
  }

  func generateNotifications(
    profileId: UUID, date: String, timeZone: String, now: Date, thresholds: GenerateThresholds
  ) async throws -> GenerateNotificationsResponse {
    let url = URL(string: "\(baseURL)/v1/inbox/generate")!
    let body = GenerateNotificationsRequest(
      profileId: profileId,
      date: date,
      clientTimeZone: timeZone,
      now: now,
      thresholds: thresholds
    )
    let request = try makeJSONRequest(url: url, method: "POST", body: body)
    return try await performAndDecode(request)
  }

  // MARK: - Intakes (Water & Supplements)

  func listSupplements(profileId: UUID) async throws -> [SupplementDTO] {
    guard let url = URL(string: "\(baseURL)/v1/supplements?profile_id=\(profileId.uuidString)")
    else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url)
    let response: SupplementsResponse = try await performAndDecode(request)
    return response.supplements
  }

  func createSupplement(profileId: UUID, name: String, notes: String?, components: [ComponentInput])
    async throws -> SupplementDTO
  {
    guard let url = URL(string: "\(baseURL)/v1/supplements") else {
      throw APIError.invalidURL
    }

    let requestBody = CreateSupplementRequest(
      profileId: profileId,
      name: name,
      notes: notes,
      components: components
    )

    let request = try makeJSONRequest(url: url, method: "POST", body: requestBody)
    return try await performAndDecode(request, expectedStatus: 201)
  }

  func fetchIntakesDaily(profileId: UUID, date: String) async throws -> IntakesDailyResponse {
    guard
      let url = URL(
        string: "\(baseURL)/v1/intakes/daily?profile_id=\(profileId.uuidString)&date=\(date)")
    else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url)
    return try await performAndDecode(request)
  }

  func addWater(profileId: UUID, takenAt: Date, amountMl: Int) async throws {
    guard let url = URL(string: "\(baseURL)/v1/intakes/water") else {
      throw APIError.invalidURL
    }

    let requestBody = AddWaterRequest(
      profileId: profileId,
      takenAt: takenAt,
      amountMl: amountMl
    )
    let request = try makeJSONRequest(url: url, method: "POST", body: requestBody)
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 201)
  }

  func upsertSupplementIntake(profileId: UUID, supplementId: UUID, date: String, status: String)
    async throws
  {
    guard let url = URL(string: "\(baseURL)/v1/intakes/supplements") else {
      throw APIError.invalidURL
    }

    let requestBody = UpsertSupplementIntakeRequest(
      profileId: profileId,
      supplementId: supplementId,
      date: date,
      status: status
    )
    let request = try makeJSONRequest(url: url, method: "POST", body: requestBody)
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 201)
  }

  // MARK: - Supplement Schedules

  func listSupplementSchedules(profileId: UUID) async throws -> [ScheduleDTO] {
    guard
      let url = URL(
        string: "\(baseURL)/v1/schedules/supplements?profile_id=\(profileId.uuidString)")
    else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url)
    let response: ListSchedulesResponse = try await performAndDecode(request, expectedStatus: 200)
    return response.schedules
  }

  func upsertSupplementSchedule(_ req: UpsertScheduleRequest) async throws -> ScheduleDTO {
    guard let url = URL(string: "\(baseURL)/v1/schedules/supplements") else {
      throw APIError.invalidURL
    }
    let request = try makeJSONRequest(url: url, method: "POST", body: req)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func replaceSupplementSchedules(_ req: ReplaceSchedulesRequest) async throws -> [ScheduleDTO] {
    guard let url = URL(string: "\(baseURL)/v1/schedules/supplements/replace") else {
      throw APIError.invalidURL
    }
    let request = try makeJSONRequest(url: url, method: "PUT", body: req)
    let response: ListSchedulesResponse = try await performAndDecode(request, expectedStatus: 200)
    return response.schedules
  }

  func deleteSupplementSchedule(id: UUID) async throws {
    guard let url = URL(string: "\(baseURL)/v1/schedules/supplements/\(id.uuidString)") else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url, method: "DELETE")
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 204)
  }

  // MARK: - Workouts API

  func fetchWorkoutPlan(profileId: UUID) async throws -> GetWorkoutPlanResponse {
    guard let url = URL(string: "\(baseURL)/v1/workouts/plan?profile_id=\(profileId.uuidString)")
    else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url, method: "GET")
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func replaceWorkoutPlan(profileId: UUID, title: String, goal: String, items: [WorkoutItemUpsert])
    async throws -> ReplaceWorkoutPlanResponse
  {
    guard let url = URL(string: "\(baseURL)/v1/workouts/plan/replace") else {
      throw APIError.invalidURL
    }
    let req = ReplaceWorkoutPlanRequest(
      profileId: profileId,
      title: title,
      goal: goal,
      replace: true,
      items: items
    )
    let request = try makeJSONRequest(url: url, method: "PUT", body: req)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func fetchWorkoutToday(profileId: UUID, date: String) async throws -> WorkoutTodayResponse {
    guard
      let url = URL(
        string: "\(baseURL)/v1/workouts/today?profile_id=\(profileId.uuidString)&date=\(date)")
    else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url, method: "GET")
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func upsertWorkoutCompletion(
    profileId: UUID, date: String, planItemId: UUID, status: String, note: String = ""
  ) async throws -> WorkoutCompletionDTO {
    guard let url = URL(string: "\(baseURL)/v1/workouts/completions") else {
      throw APIError.invalidURL
    }
    let req = UpsertWorkoutCompletionRequest(
      profileId: profileId,
      date: date,
      planItemId: planItemId,
      status: status,
      note: note
    )
    let request = try makeJSONRequest(url: url, method: "POST", body: req)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func listWorkoutCompletions(profileId: UUID, from: String, to: String) async throws
    -> [WorkoutCompletionDTO]
  {
    guard
      let url = URL(
        string:
          "\(baseURL)/v1/workouts/completions?profile_id=\(profileId.uuidString)&from=\(from)&to=\(to)"
      )
    else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url, method: "GET")
    let response: ListWorkoutCompletionsResponse = try await performAndDecode(
      request, expectedStatus: 200)
    return response.completions
  }

  // MARK: - Nutrition Targets

  func fetchNutritionTargets(profileId: UUID) async throws -> GetNutritionTargetsResponse {
    guard
      let url = URL(string: "\(baseURL)/v1/nutrition/targets?profile_id=\(profileId.uuidString)")
    else {
      throw APIError.invalidURL
    }
    let request = makeRequest(url: url, method: "GET")
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func upsertNutritionTargets(
    profileId: UUID,
    caloriesKcal: Int,
    proteinG: Int,
    fatG: Int,
    carbsG: Int,
    calciumMg: Int
  ) async throws -> NutritionTargetsDTO {
    guard let url = URL(string: "\(baseURL)/v1/nutrition/targets") else {
      throw APIError.invalidURL
    }
    let req = UpsertNutritionTargetsRequest(
      profileId: profileId,
      caloriesKcal: caloriesKcal,
      proteinG: proteinG,
      fatG: fatG,
      carbsG: carbsG,
      calciumMg: calciumMg
    )
    let request = try makeJSONRequest(url: url, method: "PUT", body: req)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  // MARK: - Food Preferences

  func listFoodPrefs(
    profileId: UUID,
    query: String? = nil,
    kind: String? = nil,
    limit: Int = 50,
    offset: Int = 0
  ) async throws -> ListFoodPrefsResponse {
    var components = URLComponents(string: "\(baseURL)/v1/food/prefs")!
    var queryItems: [URLQueryItem] = [
      URLQueryItem(name: "profile_id", value: profileId.uuidString),
      URLQueryItem(name: "limit", value: String(limit)),
      URLQueryItem(name: "offset", value: String(offset)),
    ]
    if let query = query, !query.isEmpty {
      queryItems.append(URLQueryItem(name: "q", value: query))
    }
    if let kind = kind, !kind.isEmpty {
      queryItems.append(URLQueryItem(name: "kind", value: kind))
    }
    components.queryItems = queryItems

    guard let url = components.url else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url, method: "GET")
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func upsertFoodPref(
    profileId: UUID,
    name: String,
    tags: [String],
    kcalPer100g: Int,
    proteinPer100g: Int,
    fatPer100g: Int,
    carbsPer100g: Int
  ) async throws -> FoodPrefDTO {
    guard let url = URL(string: "\(baseURL)/v1/food/prefs") else {
      throw APIError.invalidURL
    }

    let req = UpsertFoodPrefRequest(
      profileId: profileId.uuidString,
      name: name,
      tags: tags,
      kcalPer100g: kcalPer100g,
      proteinGPer100g: proteinPer100g,
      fatGPer100g: fatPer100g,
      carbsGPer100g: carbsPer100g
    )

    let request = try makeJSONRequest(url: url, method: "POST", body: req)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func deleteFoodPref(id: UUID) async throws {
    guard let url = URL(string: "\(baseURL)/v1/food/prefs/\(id.uuidString)") else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url, method: "DELETE")
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 204)
  }

  // MARK: - Meal Plan

  func fetchMealPlan(profileId: UUID) async throws -> GetMealPlanResponse {
    guard let url = URL(string: "\(baseURL)/v1/meal/plan?profile_id=\(profileId.uuidString)") else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url, method: "GET")
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func replaceMealPlan(
    profileId: UUID,
    title: String,
    items: [MealPlanItemUpsertDTO]
  ) async throws -> GetMealPlanResponse {
    guard let url = URL(string: "\(baseURL)/v1/meal/plan/replace") else {
      throw APIError.invalidURL
    }

    let req = ReplaceMealPlanRequest(
      profileId: profileId.uuidString,
      title: title,
      items: items
    )

    let request = try makeJSONRequest(url: url, method: "PUT", body: req)
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func fetchMealToday(profileId: UUID, date: Date) async throws -> GetTodayResponse {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    formatter.timeZone = TimeZone.current
    let dateStr = formatter.string(from: date)

    guard
      let url = URL(
        string: "\(baseURL)/v1/meal/today?profile_id=\(profileId.uuidString)&date=\(dateStr)")
    else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url, method: "GET")
    return try await performAndDecode(request, expectedStatus: 200)
  }

  func deleteMealPlan(profileId: UUID) async throws {
    guard let url = URL(string: "\(baseURL)/v1/meal/plan?profile_id=\(profileId.uuidString)") else {
      throw APIError.invalidURL
    }

    let request = makeRequest(url: url, method: "DELETE")
    let (data, httpResponse) = try await performRequest(request)
    try ensureExpectedStatus(request, httpResponse, data, expectedStatus: 204)
  }
}
