import Combine
import Foundation

enum AppTab: Hashable {
  case home
  case feed
  case metrics
  case activity
  case chat
}

enum FeedEditType: String, Hashable {
  case morning
  case evening
}

struct FeedNavigationRequest: Identifiable, Equatable {
  let id = UUID()
  let date: Date
  let editType: FeedEditType?
}

final class AppNavigationState: ObservableObject {
  @Published var selectedTab: AppTab = .home
  @Published var feedNavigationRequest: FeedNavigationRequest?

  func openFeed(date: Date, editType: FeedEditType? = nil) {
    feedNavigationRequest = FeedNavigationRequest(date: date, editType: editType)
    selectedTab = .feed
  }

  func openActivity() {
    selectedTab = .activity
  }
}
