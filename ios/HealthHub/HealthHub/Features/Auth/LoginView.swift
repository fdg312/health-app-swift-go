//
//  LoginView.swift
//  HealthHub
//
//  Created by HealthHub Team
//

import AuthenticationServices
import SwiftUI

struct LoginView: View {
  @StateObject private var auth = AuthManager.shared
  @State private var email = ""
  @State private var code = ""
  @State private var isLoading = false
  @State private var otpSent = false
  @State private var resendSeconds = 0
  @State private var resendTask: Task<Void, Never>?
  @State private var debugCode: String?
  @State private var alertMessage: String?
  @State private var showAlert = false

  private let resendCooldownSeconds = 60

  var body: some View {
    ZStack {
      // Background gradient
      LinearGradient(
        colors: [
          Color(.systemBackground),
          Color(.systemGroupedBackground),
        ],
        startPoint: .top,
        endPoint: .bottom
      )
      .ignoresSafeArea()

      ScrollView {
        VStack(spacing: 32) {
          Spacer(minLength: 60)

          // Header
          VStack(spacing: 16) {
            Image(systemName: "heart.text.square.fill")
              .font(.system(size: 72))
              .foregroundStyle(
                LinearGradient(
                  colors: [.red, .pink],
                  startPoint: .topLeading,
                  endPoint: .bottomTrailing
                )
              )
              .padding(.bottom, 8)

            Text("HealthHub")
              .font(.system(size: 36, weight: .bold, design: .rounded))

            Text("Ваш личный помощник\nдля здоровья и фитнеса")
              .font(.subheadline)
              .foregroundStyle(.secondary)
              .multilineTextAlignment(.center)
          }
          .padding(.bottom, 16)

          // Sign in with Apple
          SignInWithAppleButton(.signIn) { request in
            request.requestedScopes = [.fullName, .email]
          } onCompletion: { result in
            Task {
              await loginSIWA(result: result)
            }
          }
          .signInWithAppleButtonStyle(.black)
          .frame(height: 56)
          .cornerRadius(12)
          .disabled(isLoading)
          .padding(.horizontal, 24)

          // Divider
          HStack {
            Rectangle()
              .fill(Color(.separator))
              .frame(height: 1)
            Text("или")
              .font(.subheadline)
              .foregroundStyle(.secondary)
              .padding(.horizontal, 12)
            Rectangle()
              .fill(Color(.separator))
              .frame(height: 1)
          }
          .padding(.horizontal, 24)

          // Email OTP Section
          emailBlock
            .padding(.horizontal, 24)

          #if DEBUG
            // Dev mode button
            SecondaryButton(
              "Dev Mode",
              icon: "wrench.and.screwdriver.fill",
              isEnabled: !isLoading
            ) {
              Task { await loginDev() }
            }
            .padding(.horizontal, 24)

            if let debugCode {
              Text("DEBUG код: \(debugCode)")
                .font(.caption)
                .foregroundStyle(.secondary)
                .padding(.horizontal, 24)
            }
          #endif

          Spacer(minLength: 40)
        }
        .padding(.vertical, 20)
      }
    }
    .onAppear {
      if auth.didReceiveUnauthorized {
        auth.didReceiveUnauthorized = false
        presentAlert("Сессия истекла. Войдите заново.")
      }
    }
    .onDisappear {
      resendTask?.cancel()
    }
    .alert("Ошибка", isPresented: $showAlert) {
      Button("OK", role: .cancel) {}
    } message: {
      Text(alertMessage ?? "Попробуйте еще раз")
    }
  }

  // MARK: - Email Block

  private var emailBlock: some View {
    AppCard {
      VStack(alignment: .leading, spacing: 16) {
        HStack {
          Image(systemName: "envelope.fill")
            .foregroundStyle(.blue)
          Text("Вход через Email")
            .font(.headline)
        }

        VStack(spacing: 12) {
          TextField("your@email.com", text: $email)
            .textInputAutocapitalization(.never)
            .autocorrectionDisabled()
            .keyboardType(.emailAddress)
            .padding(14)
            .background(
              RoundedRectangle(cornerRadius: 10)
                .fill(Color(.tertiarySystemGroupedBackground))
            )
            .disabled(isLoading)

          if !otpSent {
            PrimaryButton(
              "Получить код",
              icon: "paperplane.fill",
              isLoading: isLoading,
              isEnabled: !email.isEmpty
            ) {
              Task { await requestCode() }
            }
          } else {
            TextField("000000", text: $code)
              .textInputAutocapitalization(.never)
              .autocorrectionDisabled()
              .keyboardType(.numberPad)
              .padding(14)
              .background(
                RoundedRectangle(cornerRadius: 10)
                  .fill(Color(.tertiarySystemGroupedBackground))
              )
              .disabled(isLoading)

            PrimaryButton(
              "Войти",
              icon: "checkmark.circle.fill",
              isLoading: isLoading,
              isEnabled: code.trimmingCharacters(in: .whitespacesAndNewlines).count == 6
            ) {
              Task { await verifyCode() }
            }

            Button {
              Task { await requestCode() }
            } label: {
              Text(resendButtonTitle)
                .font(.subheadline)
                .foregroundColor(resendSeconds > 0 ? .secondary : .blue)
            }
            .disabled(isLoading || resendSeconds > 0)
            .frame(maxWidth: .infinity)
            .padding(.top, 4)
          }
        }
      }
    }
  }

  // MARK: - Helpers

  private var resendButtonTitle: String {
    if resendSeconds > 0 {
      return "Отправить повторно через \(resendSeconds)с"
    }
    return "Отправить код повторно"
  }

  // MARK: - Actions

  private func loginDev() async {
    isLoading = true
    alertMessage = nil

    do {
      try await auth.loginDev()
    } catch {
      presentAlert(mapError(error))
    }

    isLoading = false
  }

  private func loginSIWA(result: Result<ASAuthorization, Error>) async {
    isLoading = true
    alertMessage = nil

    defer { isLoading = false }

    do {
      guard case .success(let authorization) = result else {
        if case .failure(let error) = result {
          throw error
        }
        throw APIError.invalidResponse
      }

      guard
        let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
        let tokenData = credential.identityToken,
        let identityToken = String(data: tokenData, encoding: .utf8)
      else {
        throw APIError.invalidResponse
      }

      try await auth.loginSIWA(
        identityToken: identityToken,
        user: credential.user,
        email: credential.email,
        fullName: formatFullName(credential.fullName)
      )
    } catch {
      presentAlert(mapError(error))
    }
  }

  private func requestCode() async {
    isLoading = true
    alertMessage = nil

    let normalizedEmail =
      email
      .trimmingCharacters(in: .whitespacesAndNewlines)
      .lowercased()

    defer { isLoading = false }

    do {
      let debug = try await auth.requestEmailOTP(email: normalizedEmail)
      email = normalizedEmail
      debugCode = debug
      otpSent = true
      startResendCountdown()
    } catch {
      presentAlert(mapError(error))
    }
  }

  private func verifyCode() async {
    isLoading = true
    alertMessage = nil

    let normalizedEmail =
      email
      .trimmingCharacters(in: .whitespacesAndNewlines)
      .lowercased()
    let normalizedCode = code.trimmingCharacters(in: .whitespacesAndNewlines)

    defer { isLoading = false }

    do {
      try await auth.loginEmailOTP(email: normalizedEmail, code: normalizedCode)
    } catch {
      presentAlert(mapError(error))
    }
  }

  private func startResendCountdown() {
    resendTask?.cancel()
    resendSeconds = resendCooldownSeconds
    resendTask = Task {
      while resendSeconds > 0 {
        try? await Task.sleep(nanoseconds: 1_000_000_000)
        if Task.isCancelled {
          return
        }
        await MainActor.run {
          resendSeconds = max(0, resendSeconds - 1)
        }
      }
    }
  }

  private func presentAlert(_ message: String) {
    alertMessage = message
    showAlert = true
  }

  private func mapError(_ error: Error) -> String {
    if let apiError = error as? APIError {
      switch apiError {
      case .rateLimited:
        return "Слишком много запросов, попробуйте позже."
      case .serverError(let code):
        switch code {
        case "email_auth_disabled":
          return "Вход по email сейчас отключен на сервере."
        case "invalid_email":
          return "Введите корректный email."
        case "invalid_code_format":
          return "Код должен содержать 6 цифр."
        case "otp_resend_too_soon":
          return "Код уже отправлен. Подождите и попробуйте снова."
        case "otp_rate_limited":
          return "Слишком много отправок кода. Попробуйте позже."
        case "otp_expired_or_not_found":
          return "Код истек или не найден. Запросите новый."
        case "otp_invalid_code":
          return "Неверный код. Проверьте и попробуйте снова."
        case "otp_locked":
          return "Слишком много неверных попыток. Запросите новый код."
        default:
          return "Ошибка сервера: \(code)"
        }
      default:
        return apiError.localizedDescription
      }
    }
    return error.localizedDescription
  }

  private func formatFullName(_ name: PersonNameComponents?) -> String? {
    guard let name else { return nil }
    let formatter = PersonNameComponentsFormatter()
    let value = formatter.string(from: name).trimmingCharacters(in: .whitespacesAndNewlines)
    return value.isEmpty ? nil : value
  }
}

// MARK: - Previews

#Preview {
  LoginView()
}
