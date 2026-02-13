//
//  AppCard.swift
//  HealthHub
//
//  Created by HealthHub Team
//

import SwiftUI

/// Универсальный контейнер карточки для единообразного дизайна
struct AppCard<Content: View>: View {
  let content: Content
  var padding: CGFloat = 16
  var cornerRadius: CGFloat = 12

  init(
    padding: CGFloat = 16,
    cornerRadius: CGFloat = 12,
    @ViewBuilder content: () -> Content
  ) {
    self.padding = padding
    self.cornerRadius = cornerRadius
    self.content = content()
  }

  var body: some View {
    content
      .padding(padding)
      .background(
        RoundedRectangle(cornerRadius: cornerRadius)
          .fill(Color(.systemBackground))
          .shadow(
            color: Color.black.opacity(0.05),
            radius: 8,
            x: 0,
            y: 2
          )
      )
  }
}

#Preview {
  VStack(spacing: 16) {
    AppCard {
      VStack(alignment: .leading, spacing: 8) {
        Text("Карточка")
          .font(.headline)
        Text("Пример контента в карточке")
          .font(.subheadline)
          .foregroundStyle(.secondary)
      }
    }

    AppCard(padding: 20, cornerRadius: 16) {
      Text("Кастомная карточка")
    }
  }
  .padding()
  .background(Color(.systemGroupedBackground))
}
