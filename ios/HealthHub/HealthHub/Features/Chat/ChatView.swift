import SwiftUI

struct ChatView: View {
    var body: some View {
        NavigationStack {
            VStack {
                Spacer()
                Text("AI Ассистент")
                    .font(.title)
                    .foregroundStyle(.secondary)
                Text("Чат с помощником здоровья")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                Spacer()
            }
            .navigationTitle("Чат")
        }
    }
}

#Preview {
    ChatView()
}
