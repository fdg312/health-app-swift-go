import SwiftUI

@main
struct HealthHubApp: App {
  @Environment(\.scenePhase) private var scenePhase
  @StateObject private var authManager = AuthManager.shared
  @StateObject private var navigation = AppNavigationState()
  private let backgroundSync = BackgroundSyncManager.shared
  @State private var didConfigureAuthenticatedSession = false

  var body: some Scene {
    WindowGroup {
      Group {
        if authManager.isAuthenticated {
          ContentView()
            .environmentObject(navigation)
        } else {
          LoginView()
        }
      }
      .task {
        backgroundSync.registerBGTasks()
        await configureAuthBoundServices(isAuthenticated: authManager.isAuthenticated)
      }
      .onChange(of: authManager.isAuthenticated) { _, isAuthenticated in
        Task {
          await configureAuthBoundServices(isAuthenticated: isAuthenticated)
        }
      }
      .onChange(of: scenePhase) { _, newPhase in
        if newPhase == .background && authManager.isAuthenticated {
          backgroundSync.scheduleAppRefresh()
        }
      }
    }
  }

  private func configureAuthBoundServices(isAuthenticated: Bool) async {
    if !isAuthenticated {
      didConfigureAuthenticatedSession = false
      backgroundSync.cancelScheduledRefresh()
      HealthKitManager.shared.stopObserverQueries()
      return
    }

    if didConfigureAuthenticatedSession {
      return
    }
    didConfigureAuthenticatedSession = true

    let syncPreferences = SyncPreferences()
    guard syncPreferences.backgroundSyncEnabled else {
      HealthKitManager.shared.stopObserverQueries()
      return
    }

    do {
      try await HealthKitManager.shared.enableBackgroundDelivery()
      HealthKitManager.shared.startObserverQueries {
        Task { @MainActor in
          BackgroundSyncManager.shared.handleHealthKitObserverUpdate()
        }
      }
    } catch {
      #if DEBUG
        print("Failed to configure HealthKit background delivery: \(error)")
      #endif
    }

    if backgroundSync.shouldRunLoginSync() {
      _ = await backgroundSync.performSync(reason: "login")
    }
    backgroundSync.scheduleAppRefresh()
  }
}
