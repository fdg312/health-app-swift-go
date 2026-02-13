import SwiftUI

/// Clean, shareable view of day summary (without interactive elements)
struct DaySummaryShareCard: View {
    let date: String
    let metrics: MetricsData?
    let morning: CheckinData?
    let evening: CheckinData?

    struct MetricsData {
        let steps: Int?
        let weight: Double?
        let restingHR: Int?
        let sleepMinutes: Int?
        let activeEnergyKcal: Int?
    }

    struct CheckinData {
        let score: Int
        let tags: [String]
        let note: String?
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            // Header
            VStack(alignment: .leading, spacing: 4) {
                Text("Сводка дня")
                    .font(.title2.bold())
                Text(formattedDate)
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            // Metrics Section
            if let metrics = metrics {
                VStack(alignment: .leading, spacing: 12) {
                    Text("Показатели")
                        .font(.headline)

                    LazyVGrid(columns: [
                        GridItem(.flexible()),
                        GridItem(.flexible())
                    ], spacing: 12) {
                        if let steps = metrics.steps {
                            MetricBox(icon: "figure.walk", label: "Шаги", value: "\(steps)")
                        }
                        if let weight = metrics.weight {
                            MetricBox(icon: "scalemass", label: "Вес", value: String(format: "%.1f кг", weight))
                        }
                        if let hr = metrics.restingHR {
                            MetricBox(icon: "heart.fill", label: "Пульс", value: "\(hr) bpm")
                        }
                        if let sleep = metrics.sleepMinutes {
                            let hours = sleep / 60
                            let minutes = sleep % 60
                            MetricBox(icon: "bed.double.fill", label: "Сон", value: "\(hours)ч \(minutes)м")
                        }
                        if let energy = metrics.activeEnergyKcal {
                            MetricBox(icon: "flame.fill", label: "Энергия", value: "\(energy) ккал")
                        }
                    }
                }
            }

            // Checkins Section
            if morning != nil || evening != nil {
                VStack(alignment: .leading, spacing: 12) {
                    Text("Чекины")
                        .font(.headline)

                    if let morning = morning {
                        CheckinBox(type: "Утро", data: morning)
                    }
                    if let evening = evening {
                        CheckinBox(type: "Вечер", data: evening)
                    }
                }
            }

            // Footer
            HStack {
                Spacer()
                Text("HealthHub")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                Spacer()
            }
        }
        .padding(24)
        .background(
            RoundedRectangle(cornerRadius: 16)
                .fill(Color(.systemBackground))
        )
        .frame(width: 400)
    }

    private var formattedDate: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        guard let parsedDate = formatter.date(from: date) else {
            return date
        }
        formatter.dateFormat = "d MMMM yyyy"
        formatter.locale = Locale(identifier: "ru_RU")
        return formatter.string(from: parsedDate)
    }
}

// MARK: - Metric Box

private struct MetricBox: View {
    let icon: String
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.caption)
                Text(label)
                    .font(.caption)
            }
            .foregroundStyle(.secondary)

            Text(value)
                .font(.subheadline.bold())
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(8)
        .background(
            RoundedRectangle(cornerRadius: 8)
                .fill(Color(.secondarySystemBackground))
        )
    }
}

// MARK: - Checkin Box

private struct CheckinBox: View {
    let type: String
    let data: DaySummaryShareCard.CheckinData

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text(type)
                    .font(.subheadline.bold())
                Spacer()
                StarsView(score: data.score)
            }

            if !data.tags.isEmpty {
                FlowLayout(spacing: 6) {
                    ForEach(data.tags, id: \.self) { tag in
                        Text(tag)
                            .font(.caption)
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(
                                Capsule()
                                    .fill(Color.blue.opacity(0.15))
                            )
                            .foregroundStyle(.blue)
                    }
                }
            }

            if let note = data.note, !note.isEmpty {
                Text(note)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(3)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 8)
                .fill(Color(.secondarySystemBackground))
        )
    }
}

// MARK: - Stars View

private struct StarsView: View {
    let score: Int

    private var color: Color {
        switch score {
        case 1...2: return .red
        case 3: return .orange
        case 4: return .green
        case 5: return .blue
        default: return .gray
        }
    }

    var body: some View {
        HStack(spacing: 2) {
            ForEach(1...5, id: \.self) { index in
                Image(systemName: index <= score ? "star.fill" : "star")
                    .font(.caption)
                    .foregroundStyle(index <= score ? color : .gray.opacity(0.3))
            }
        }
    }
}

// MARK: - Flow Layout (for tags)

private struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let result = FlowResult(
            in: proposal.replacingUnspecifiedDimensions().width,
            subviews: subviews,
            spacing: spacing
        )
        return result.size
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        let result = FlowResult(
            in: bounds.width,
            subviews: subviews,
            spacing: spacing
        )
        for (index, subview) in subviews.enumerated() {
            subview.place(at: CGPoint(x: bounds.minX + result.frames[index].minX,
                                     y: bounds.minY + result.frames[index].minY),
                         proposal: .unspecified)
        }
    }

    struct FlowResult {
        var frames: [CGRect] = []
        var size: CGSize = .zero

        init(in maxWidth: CGFloat, subviews: Subviews, spacing: CGFloat) {
            var currentX: CGFloat = 0
            var currentY: CGFloat = 0
            var lineHeight: CGFloat = 0

            for subview in subviews {
                let size = subview.sizeThatFits(.unspecified)

                if currentX + size.width > maxWidth && currentX > 0 {
                    currentX = 0
                    currentY += lineHeight + spacing
                    lineHeight = 0
                }

                frames.append(CGRect(x: currentX, y: currentY, width: size.width, height: size.height))
                lineHeight = max(lineHeight, size.height)
                currentX += size.width + spacing
            }

            self.size = CGSize(width: maxWidth, height: currentY + lineHeight)
        }
    }
}

struct DaySummaryShareCard_Previews: PreviewProvider {
    static var previews: some View {
        DaySummaryShareCard(
            date: "2026-02-13",
            metrics: .init(
                steps: 12500,
                weight: 75.2,
                restingHR: 62,
                sleepMinutes: 420,
                activeEnergyKcal: 450
            ),
            morning: .init(
                score: 4,
                tags: ["энергия", "хорошее настроение"],
                note: "Отличное утро!"
            ),
            evening: .init(
                score: 2,
                tags: ["стресс", "усталость"],
                note: "Тяжелый день на работе"
            )
        )
        .padding()
        .background(Color(.systemGroupedBackground))
    }
}
