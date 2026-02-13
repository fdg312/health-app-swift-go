import Foundation
import Combine

class AuthManager: ObservableObject {
    static let shared = AuthManager()

    @Published var token: String?
    @Published var isAuthenticated: Bool
    @Published var didReceiveUnauthorized: Bool = false

    private let tokenKey = "health_hub_access_token"

    private init() {
        let storedToken = UserDefaults.standard.string(forKey: tokenKey)
        self.token = storedToken
        self.isAuthenticated = storedToken != nil
    }

    var accessToken: String? {
        token
    }

    func loginDev() async throws {
        let response = try await APIClient.shared.loginDev()
        saveToken(response.accessToken)
    }

    func loginSIWA(identityToken: String, user: String?, email: String?, fullName: String?) async throws {
        let response = try await APIClient.shared.loginSIWA(
            identityToken: identityToken,
            user: user,
            email: email,
            fullName: fullName
        )
        saveToken(response.accessToken)
    }

    func requestEmailOTP(email: String) async throws -> String? {
        let response = try await APIClient.shared.requestEmailOTP(email: email)
        return response.debugCode
    }

    func loginEmailOTP(email: String, code: String) async throws {
        let response = try await APIClient.shared.verifyEmailOTP(email: email, code: code)
        saveToken(response.accessToken)
    }

    func saveAuth(accessToken: String, ownerUserID: String, ownerProfileID: UUID) {
        // Backward-compatible API for existing SIWA flow.
        saveToken(accessToken)
    }

    func logout() {
        UserDefaults.standard.removeObject(forKey: tokenKey)
        var syncPreferences = SyncPreferences()
        syncPreferences.cachedOwnerProfileID = nil
        DispatchQueue.main.async {
            self.token = nil
            self.isAuthenticated = false
        }
    }

    func handleUnauthorized() {
        DispatchQueue.main.async {
            self.didReceiveUnauthorized = true
        }
        logout()
    }

    @discardableResult
    func checkUnauthorized(_ error: Error) -> Bool {
        if let apiError = error as? APIError, apiError == .unauthorized {
            handleUnauthorized()
            return true
        }
        return false
    }

    private func saveToken(_ token: String) {
        UserDefaults.standard.set(token, forKey: tokenKey)
        DispatchQueue.main.async {
            self.token = token
            self.isAuthenticated = true
            self.didReceiveUnauthorized = false
        }
    }
}
