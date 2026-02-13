import Foundation

/// Application configuration
enum AppConfig {
    /// Base URL for API endpoints
    static var apiBaseURL: String {
        // Try to read from Info.plist first
        if let baseURL = Bundle.main.object(forInfoDictionaryKey: "API_BASE_URL") as? String,
           !baseURL.isEmpty {
            return baseURL
        }

        // Default to localhost for simulator
        return "http://localhost:8080"
    }
}
