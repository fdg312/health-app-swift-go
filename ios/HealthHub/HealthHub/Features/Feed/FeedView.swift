import SwiftUI

struct FeedView: View {
    var body: some View {
        NavigationStack {
            VStack {
                Spacer()
                Text("Лента активности")
                    .font(.title)
                    .foregroundStyle(.secondary)
                Text("Здесь будут события и обновления")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                Spacer()
            }
            .navigationTitle("Лента")
        }
    }
}

#Preview {
    FeedView()
}
