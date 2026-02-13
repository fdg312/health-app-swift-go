//
//  SignInView.swift
//  HealthHub
//

import SwiftUI
import AuthenticationServices

struct SignInView: View {
    @StateObject private var authManager = AuthManager.shared
    @State private var isSigningIn = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(spacing: 24) {
            Spacer()

            // Logo / Title
            VStack(spacing: 8) {
                Image(systemName: "heart.fill")
                    .font(.system(size: 64))
                    .foregroundStyle(.red)

                Text("HealthHub")
                    .font(.largeTitle.bold())

                Text("Центр здоровья")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            // Error message
            if let error = errorMessage {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal)
            }

            // Sign in with Apple button
            if isSigningIn {
                ProgressView("Вход...")
                    .padding()
            } else {
                SignInWithAppleButton(
                    onRequest: { request in
                        request.requestedScopes = [.email]
                    },
                    onCompletion: { result in
                        Task {
                            await handleSignInResult(result)
                        }
                    }
                )
                .signInWithAppleButtonStyle(.black)
                .frame(height: 50)
                .padding(.horizontal, 40)
            }

            Spacer()

            // Dev mode note
            Text("Для разработки: AUTH_ENABLED=0 на сервере не требует авторизации")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)
                .padding(.bottom, 20)
        }
    }

    private func handleSignInResult(_ result: Result<ASAuthorization, Error>) async {
        isSigningIn = true
        errorMessage = nil

        do {
            switch result {
            case .success(let authorization):
                guard let appleIDCredential = authorization.credential as? ASAuthorizationAppleIDCredential,
                      let identityTokenData = appleIDCredential.identityToken,
                      let identityTokenString = String(data: identityTokenData, encoding: .utf8) else {
                    throw SignInError.invalidCredential
                }

                let fullName = formatName(appleIDCredential.fullName)
                try await authManager.loginSIWA(
                    identityToken: identityTokenString,
                    user: appleIDCredential.user,
                    email: appleIDCredential.email,
                    fullName: fullName
                )

            case .failure(let error):
                throw error
            }
        } catch {
            errorMessage = "Ошибка входа: \(error.localizedDescription)"
        }

        isSigningIn = false
    }

    private func formatName(_ name: PersonNameComponents?) -> String? {
        guard let name else { return nil }
        let formatter = PersonNameComponentsFormatter()
        let value = formatter.string(from: name).trimmingCharacters(in: .whitespacesAndNewlines)
        return value.isEmpty ? nil : value
    }
}

// MARK: - Models

enum SignInError: LocalizedError {
    case invalidCredential

    var errorDescription: String? {
        switch self {
        case .invalidCredential:
            return "Не удалось получить данные авторизации"
        }
    }
}

struct SignInView_Previews: PreviewProvider {
    static var previews: some View {
        SignInView()
    }
}
