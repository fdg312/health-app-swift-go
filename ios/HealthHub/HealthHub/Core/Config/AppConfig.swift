import Foundation

/// Application configuration
enum AppConfig {
  /// Base URL for API endpoints
  static var apiBaseURL: String {
    // Try to read from Info.plist first
    if let baseURL = Bundle.main.object(forInfoDictionaryKey: "API_BASE_URL") as? String,
      !baseURL.isEmpty
    {
      print("📡 API Base URL from Info.plist: \(baseURL)")
      return baseURL
    }

    // Fallback to production server
    let fallback = "https://health-app-swift-go.onrender.com"
    print("📡 API Base URL (fallback): \(fallback)")
    return fallback
  }
}
