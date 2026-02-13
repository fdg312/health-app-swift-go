import SwiftUI

struct ContentView: View {
    @EnvironmentObject private var navigation: AppNavigationState

    var body: some View {
        TabView(selection: $navigation.selectedTab) {
            HomeView()
                .tabItem {
                    Label("Главная", systemImage: "house")
                }
                .tag(AppTab.home)

            FeedView()
                .tabItem {
                    Label("Лента", systemImage: "list.bullet")
                }
                .tag(AppTab.feed)

            MetricsView()
                .tabItem {
                    Label("Показатели", systemImage: "chart.line.uptrend.xyaxis")
                }
                .tag(AppTab.metrics)

            ActivityView()
                .tabItem {
                    Label("Активность", systemImage: "figure.walk")
                }
                .tag(AppTab.activity)

            ChatView()
                .tabItem {
                    Label("Чат", systemImage: "message")
                }
                .tag(AppTab.chat)
        }
    }
}

struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView()
            .environmentObject(AppNavigationState())
    }
}
