//
//  StatTile.swift
//  HealthHub
//
//  Created by HealthHub Team
//

import SwiftUI

/// Плитка статистики с заголовком, значением, подписью и опциональной иконкой
struct StatTile: View {
  let title: String
  let value: String
  let subtitle: String?
  let icon: String?
  let iconColor: Color

  init(
    title: String,
    value: String,
    subtitle: String? = nil,
    icon: String? = nil,
    iconColor: Color = .blue
  ) {
    self.title = title
    self.value = value
    self.subtitle = subtitle
    self.icon = icon
    self.iconColor = iconColor
  }

  var body: some View {
    VStack(alignment: .leading, spacing: 8) {
      HStack(spacing: 8) {
        if let icon = icon {
          Image(systemName: icon)
            .font(.system(size: 16, weight: .medium))
            .foregroundStyle(iconColor)
        }

        Text(title)
          .font(.subheadline)
          .foregroundStyle(.secondary)
      }

      Text(value)
        .font(.system(size: 24, weight: .semibold, design: .rounded))
        .foregroundStyle(.primary)

      if let subtitle = subtitle {
        Text(subtitle)
          .font(.caption)
          .foregroundStyle(.secondary)
      }
    }
    .frame(maxWidth: .infinity, alignment: .leading)
    .padding()
    .background(
      RoundedRectangle(cornerRadius: 12)
        .fill(Color(.secondarySystemGroupedBackground))
    )
  }
}

#Preview {
  VStack(spacing: 16) {
    StatTile(
      title: "Шаги",
      value: "8,542",
      subtitle: "из 10,000",
      icon: "figure.walk",
      iconColor: .green
    )

    StatTile(
      title: "Сон",
      value: "7ч 23м",
      icon: "bed.double.fill",
      iconColor: .indigo
    )

    StatTile(
      title: "Калории",
      value: "1,847",
      subtitle: "ккал съедено"
    )

    HStack(spacing: 12) {
      StatTile(
        title: "Энергия",
        value: "8/10",
        icon: "bolt.fill",
        iconColor: .orange
      )

      StatTile(
        title: "Вода",
        value: "1.5л",
        icon: "drop.fill",
        iconColor: .cyan
      )
    }
  }
  .padding()
  .background(Color(.systemGroupedBackground))
}
