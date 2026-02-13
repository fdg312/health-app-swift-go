//
//  AppButtons.swift
//  HealthHub
//
//  Created by HealthHub Team
//

import SwiftUI

/// Основная кнопка приложения
struct PrimaryButton: View {
  let title: String
  let icon: String?
  let action: () -> Void
  var isLoading: Bool = false
  var isEnabled: Bool = true

  init(
    _ title: String,
    icon: String? = nil,
    isLoading: Bool = false,
    isEnabled: Bool = true,
    action: @escaping () -> Void
  ) {
    self.title = title
    self.icon = icon
    self.isLoading = isLoading
    self.isEnabled = isEnabled
    self.action = action
  }

  var body: some View {
    Button(action: action) {
      HStack(spacing: 8) {
        if isLoading {
          ProgressView()
            .progressViewStyle(CircularProgressViewStyle(tint: .white))
        } else {
          if let icon = icon {
            Image(systemName: icon)
              .font(.system(size: 16, weight: .semibold))
          }

          Text(title)
            .font(.system(size: 17, weight: .semibold))
        }
      }
      .frame(maxWidth: .infinity)
      .frame(height: 50)
      .foregroundStyle(.white)
      .background(
        RoundedRectangle(cornerRadius: 12)
          .fill(isEnabled ? Color.accentColor : Color.gray)
      )
    }
    .disabled(!isEnabled || isLoading)
  }
}

/// Вторичная кнопка приложения
struct SecondaryButton: View {
  let title: String
  let icon: String?
  let action: () -> Void
  var isEnabled: Bool = true

  init(
    _ title: String,
    icon: String? = nil,
    isEnabled: Bool = true,
    action: @escaping () -> Void
  ) {
    self.title = title
    self.icon = icon
    self.isEnabled = isEnabled
    self.action = action
  }

  var body: some View {
    Button(action: action) {
      HStack(spacing: 8) {
        if let icon = icon {
          Image(systemName: icon)
            .font(.system(size: 16, weight: .medium))
        }

        Text(title)
          .font(.system(size: 17, weight: .medium))
      }
      .frame(maxWidth: .infinity)
      .frame(height: 50)
      .foregroundStyle(isEnabled ? Color.accentColor : Color.gray)
      .background(
        RoundedRectangle(cornerRadius: 12)
          .strokeBorder(isEnabled ? Color.accentColor : Color.gray, lineWidth: 2)
          .background(
            RoundedRectangle(cornerRadius: 12)
              .fill(Color(.secondarySystemGroupedBackground))
          )
      )
    }
    .disabled(!isEnabled)
  }
}

#Preview {
  VStack(spacing: 16) {
    PrimaryButton("Сохранить", icon: "checkmark") {
      print("Primary tapped")
    }

    PrimaryButton("Загрузка...", isLoading: true) {
      print("Loading")
    }

    PrimaryButton("Недоступно", isEnabled: false) {
      print("Disabled")
    }

    SecondaryButton("Отмена", icon: "xmark") {
      print("Secondary tapped")
    }

    SecondaryButton("Недоступно", isEnabled: false) {
      print("Disabled")
    }
  }
  .padding()
  .background(Color(.systemGroupedBackground))
}
