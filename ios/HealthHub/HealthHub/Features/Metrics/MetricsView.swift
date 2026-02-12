import SwiftUI

struct MetricsView: View {
    var body: some View {
        NavigationStack {
            VStack {
                Spacer()
                Text("Показатели здоровья")
                    .font(.title)
                    .foregroundStyle(.secondary)
                Text("Графики и статистика")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                Spacer()
            }
            .navigationTitle("Показатели")
        }
    }
}

#Preview {
    MetricsView()
}
