//
//  EmptyStateView.swift
//  HealthHub
//
//  Created by HealthHub Team
//

import SwiftUI

/// Пустое состояние с иконкой, заголовком, описанием и опциональной кнопкой
struct EmptyStateView: View {
  let icon: String
  let title: String
  let description: String
  let actionTitle: String?
  let action: (() -> Void)?

  init(
    icon: String,
    title: String,
    description: String,
    actionTitle: String? = nil,
    action: (() -> Void)? = nil
  ) {
    self.icon = icon
    self.title = title
    self.description = description
    self.actionTitle = actionTitle
    self.action = action
  }

  var body: some View {
    VStack(spacing: 16) {
      Image(systemName: icon)
        .font(.system(size: 48))
        .foregroundStyle(.secondary)
        .padding(.bottom, 8)

      Text(title)
        .font(.system(size: 20, weight: .semibold))
        .foregroundStyle(.primary)
        .multilineTextAlignment(.center)

      Text(description)
        .font(.system(size: 15))
        .foregroundStyle(.secondary)
        .multilineTextAlignment(.center)
        .padding(.horizontal, 32)

      if let actionTitle = actionTitle, let action = action {
        Button(action: action) {
          Text(actionTitle)
            .font(.system(size: 16, weight: .medium))
            .foregroundStyle(.white)
            .frame(height: 44)
            .padding(.horizontal, 24)
            .background(
              RoundedRectangle(cornerRadius: 10)
                .fill(Color.accentColor)
            )
        }
        .padding(.top, 8)
      }
    }
    .padding(32)
  }
}

#Preview {
  VStack(spacing: 40) {
    EmptyStateView(
      icon: "fork.knife",
      title: "Нет плана питания",
      description: "Сгенерируйте персональный план питания через чат с AI ассистентом",
      actionTitle: "Открыть чат",
      action: { print("Action tapped") }
    )

    EmptyStateView(
      icon: "figure.run",
      title: "Нет тренировок",
      description: "Добавьте свою первую тренировку, чтобы отслеживать прогресс"
    )
  }
  .background(Color(.systemGroupedBackground))
}
