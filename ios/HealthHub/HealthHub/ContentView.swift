import SwiftUI

struct ContentView: View {
    var body: some View {
        TabView {
            FeedView()
                .tabItem {
                    Label("Лента", systemImage: "list.bullet")
                }

            MetricsView()
                .tabItem {
                    Label("Показатели", systemImage: "chart.line.uptrend.xyaxis")
                }

            ActivityView()
                .tabItem {
                    Label("Активность", systemImage: "figure.walk")
                }

            ChatView()
                .tabItem {
                    Label("Чат", systemImage: "message")
                }
        }
    }
}

#Preview {
    ContentView()
}
