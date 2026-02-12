import SwiftUI

struct ActivityView: View {
    var body: some View {
        NavigationStack {
            VStack {
                Spacer()
                Text("Активность")
                    .font(.title)
                    .foregroundStyle(.secondary)
                Text("Тренировки и шаги")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                Spacer()
            }
            .navigationTitle("Активность")
        }
    }
}

#Preview {
    ActivityView()
}
