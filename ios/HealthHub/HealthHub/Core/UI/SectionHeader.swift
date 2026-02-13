//
//  SectionHeader.swift
//  HealthHub
//
//  Created by HealthHub Team
//

import SwiftUI

/// Заголовок секции с опциональной trailing action
struct SectionHeader: View {
  let title: String
  let actionTitle: String?
  let action: (() -> Void)?

  init(
    _ title: String,
    actionTitle: String? = nil,
    action: (() -> Void)? = nil
  ) {
    self.title = title
    self.actionTitle = actionTitle
    self.action = action
  }

  var body: some View {
    HStack(alignment: .center) {
      Text(title)
        .font(.system(size: 20, weight: .bold))
        .foregroundStyle(.primary)

      Spacer()

      if let actionTitle = actionTitle, let action = action {
        Button(action: action) {
          Text(actionTitle)
            .font(.system(size: 15, weight: .medium))
            .foregroundStyle(Color.accentColor)
        }
      }
    }
    .padding(.horizontal, 4)
    .padding(.vertical, 8)
  }
}

#Preview {
  VStack(spacing: 24) {
    SectionHeader("Сегодня")

    SectionHeader(
      "План питания",
      actionTitle: "Все",
      action: { print("Action tapped") }
    )

    SectionHeader(
      "Тренировки",
      actionTitle: "Добавить",
      action: { print("Add tapped") }
    )
  }
  .padding()
  .background(Color(.systemGroupedBackground))
}
