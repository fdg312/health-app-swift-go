//
//  TagChip.swift
//  HealthHub
//
//  Created by HealthHub Team
//

import SwiftUI

/// Чип для тегов и лейблов
struct TagChip: View {
  let text: String
  let icon: String?
  let color: Color
  var isSelected: Bool = false
  var action: (() -> Void)?

  init(
    _ text: String,
    icon: String? = nil,
    color: Color = .blue,
    isSelected: Bool = false,
    action: (() -> Void)? = nil
  ) {
    self.text = text
    self.icon = icon
    self.color = color
    self.isSelected = isSelected
    self.action = action
  }

  var body: some View {
    Group {
      if let action = action {
        Button(action: action) {
          chipContent
        }
      } else {
        chipContent
      }
    }
  }

  private var chipContent: some View {
    HStack(spacing: 6) {
      if let icon = icon {
        Image(systemName: icon)
          .font(.system(size: 12, weight: .medium))
      }

      Text(text)
        .font(.system(size: 14, weight: .medium))
    }
    .padding(.horizontal, 12)
    .padding(.vertical, 6)
    .foregroundStyle(isSelected ? .white : color)
    .background(
      RoundedRectangle(cornerRadius: 8)
        .fill(isSelected ? color : color.opacity(0.15))
    )
  }
}

#Preview {
  VStack(spacing: 16) {
    HStack(spacing: 8) {
      TagChip("Завтрак", icon: "sun.max.fill", color: .orange)
      TagChip("Обед", icon: "sun.max", color: .yellow)
      TagChip("Ужин", icon: "moon.fill", color: .indigo)
    }

    HStack(spacing: 8) {
      TagChip("Белки", color: .red)
      TagChip("Жиры", color: .orange)
      TagChip("Углеводы", color: .green)
    }

    HStack(spacing: 8) {
      TagChip("Выбрано", color: .blue, isSelected: true)
      TagChip("Не выбрано", color: .blue, isSelected: false)
    }

    HStack(spacing: 8) {
      TagChip("Кнопка", color: .purple, action: { print("Tapped") })
      TagChip(
        "Активная", icon: "checkmark", color: .green, isSelected: true, action: { print("Tapped") })
    }
  }
  .padding()
  .background(Color(.systemGroupedBackground))
}
