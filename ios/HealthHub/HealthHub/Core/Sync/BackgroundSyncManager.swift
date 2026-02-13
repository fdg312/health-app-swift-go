import Foundation
import BackgroundTasks
import UIKit
import Combine

@MainActor
final class BackgroundSyncManager: ObservableObject {
    static let shared = BackgroundSyncManager()

    enum SyncStatus: String {
        case idle
        case syncing
        case ok
        case error
    }

    @Published private(set) var lastSyncAt: Date?
    @Published private(set) var lastSyncStatus: SyncStatus
    @Published private(set) var lastErrorMessage: String?
    @Published private(set) var isSyncInProgress = false

    private var preferences = SyncPreferences()
    private let defaults = UserDefaults.standard

    private var isTaskRegistered = false
    private var observerDebounceTask: Task<Void, Never>?

    private enum Keys {
        static let lastSyncAt = "background_sync_last_at"
        static let lastSyncStatus = "background_sync_last_status"
        static let lastSyncError = "background_sync_last_error"
    }

    private var refreshTaskIdentifier: String {
        let bundleID = Bundle.main.bundleIdentifier ?? "fdg312.HealthHub"
        return "\(bundleID).refresh"
    }

    private init() {
        if let savedDate = defaults.object(forKey: Keys.lastSyncAt) as? Date {
            lastSyncAt = savedDate
        }
        if let rawStatus = defaults.string(forKey: Keys.lastSyncStatus),
           let parsed = SyncStatus(rawValue: rawStatus) {
            lastSyncStatus = parsed
        } else {
            lastSyncStatus = .idle
        }
        let savedError = defaults.string(forKey: Keys.lastSyncError)
        lastErrorMessage = savedError?.isEmpty == true ? nil : savedError
    }

    func registerBGTasks() {
        guard !isTaskRegistered else { return }

        let identifier = refreshTaskIdentifier
        let didRegister = BGTaskScheduler.shared.register(forTaskWithIdentifier: identifier, using: nil) { task in
            guard let appRefreshTask = task as? BGAppRefreshTask else {
                task.setTaskCompleted(success: false)
                return
            }

            Task { @MainActor in
                self.handleAppRefresh(task: appRefreshTask)
            }
        }

        if didRegister {
            isTaskRegistered = true
        } else {
            #if DEBUG
            print("BackgroundSync: failed to register BG task \(identifier)")
            #endif
        }
    }

    func scheduleAppRefresh(after interval: TimeInterval = 2 * 60 * 60) {
        guard preferences.backgroundSyncEnabled else {
            cancelScheduledRefresh()
            return
        }

        cancelScheduledRefresh()

        let request = BGAppRefreshTaskRequest(identifier: refreshTaskIdentifier)
        request.earliestBeginDate = Date(timeIntervalSinceNow: interval)

        do {
            try BGTaskScheduler.shared.submit(request)
        } catch {
            #if DEBUG
            print("BackgroundSync: failed to schedule app refresh: \(error)")
            #endif
        }
    }

    func cancelScheduledRefresh() {
        BGTaskScheduler.shared.cancel(taskRequestWithIdentifier: refreshTaskIdentifier)
    }

    func handleHealthKitObserverUpdate() {
        scheduleAppRefresh(after: 15 * 60)

        guard preferences.backgroundSyncEnabled else { return }
        guard UIApplication.shared.applicationState == .active else { return }

        observerDebounceTask?.cancel()
        observerDebounceTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: 15_000_000_000)
            guard !Task.isCancelled, let self else { return }
            _ = await self.performSync(reason: "observer_foreground")
        }
    }

    func shouldRunLoginSync(maxAgeHours: Double = 6) -> Bool {
        guard let lastSyncAt else { return true }
        return Date().timeIntervalSince(lastSyncAt) >= maxAgeHours * 3600
    }

    func handleAppRefresh(task: BGAppRefreshTask) {
        guard preferences.backgroundSyncEnabled else {
            task.setTaskCompleted(success: true)
            return
        }

        scheduleAppRefresh()

        let syncTask = Task { [weak self] in
            guard let self else { return false }
            return await self.performSync(reason: "bg_refresh")
        }

        task.expirationHandler = {
            syncTask.cancel()
        }

        Task { @MainActor in
            let success = await syncTask.value
            task.setTaskCompleted(success: success)
            if success {
                self.scheduleAppRefresh()
            }
        }
    }

    @discardableResult
    func performSync(reason: String) async -> Bool {
        if isSyncInProgress {
            return false
        }

        guard AuthManager.shared.isAuthenticated, AuthManager.shared.accessToken != nil else {
            return false
        }

        isSyncInProgress = true
        setStatus(.syncing, at: lastSyncAt, errorMessage: nil)

        defer {
            isSyncInProgress = false
        }

        do {
            let profileID = try await resolveOwnerProfileID()
            let resolvedProfileID = try await sendSyncBatch(profileID: profileID)

            let shouldRefreshReminders = preferences.backgroundRemindersEnabled || reason == "manual"
            if shouldRefreshReminders {
                try await refreshInboxAndReminders(profileID: resolvedProfileID)
            }

            let syncedAt = Date()
            setStatus(.ok, at: syncedAt, errorMessage: nil)
            return true
        } catch is CancellationError {
            setStatus(.error, at: lastSyncAt, errorMessage: "Синхронизация отменена")
            return false
        } catch {
            if let apiError = error as? APIError, apiError == .unauthorized {
                AuthManager.shared.handleUnauthorized()
            }
            setStatus(.error, at: lastSyncAt, errorMessage: shortErrorMessage(error))
            return false
        }
    }

    private func resolveOwnerProfileID(forceRefresh: Bool = false) async throws -> UUID {
        if !forceRefresh,
           let cached = preferences.cachedOwnerProfileID,
           let cachedUUID = UUID(uuidString: cached) {
            return cachedUUID
        }

        let profiles = try await APIClient.shared.listProfiles()
        guard let owner = profiles.first(where: { $0.type == "owner" }) else {
            throw APIError.serverError("owner_profile_not_found")
        }

        preferences.cachedOwnerProfileID = owner.id.uuidString
        return owner.id
    }

    private func sendSyncBatch(profileID: UUID) async throws -> UUID {
        let request = try await buildSyncRequest(profileID: profileID)

        do {
            _ = try await APIClient.shared.sendSyncBatch(request: request)
            return profileID
        } catch let APIError.serverError(code) where code == "profile_not_found" {
            preferences.cachedOwnerProfileID = nil
            let freshProfileID = try await resolveOwnerProfileID(forceRefresh: true)
            let freshRequest = try await buildSyncRequest(profileID: freshProfileID)
            _ = try await APIClient.shared.sendSyncBatch(request: freshRequest)
            return freshProfileID
        }
    }

    private func buildSyncRequest(profileID: UUID) async throws -> SyncBatchRequest {
        let calendar = Calendar.current
        let now = Date()

        let startOfToday = calendar.startOfDay(for: now)
        guard let startOfYesterday = calendar.date(byAdding: .day, value: -1, to: startOfToday),
              let startOfTomorrow = calendar.date(byAdding: .day, value: 1, to: startOfToday) else {
            throw APIError.invalidResponse
        }

        let rawRequest = try await HealthKitManager.shared.buildSyncRequest(
            profileId: profileID,
            from: startOfYesterday,
            to: now
        )

        let todayHourly = rawRequest.hourly?.filter { bucket in
            bucket.hour >= startOfToday && bucket.hour < startOfTomorrow
        }

        return SyncBatchRequest(
            profileId: rawRequest.profileId,
            clientTimeZone: rawRequest.clientTimeZone,
            daily: rawRequest.daily,
            hourly: todayHourly?.isEmpty == true ? nil : todayHourly,
            sessions: rawRequest.sessions
        )
    }

    private func refreshInboxAndReminders(profileID: UUID) async throws {
        let dateString = apiDateString(Date())
        let thresholds = await loadThresholdsForGeneration()

        _ = try await APIClient.shared.generateNotifications(
            profileId: profileID,
            date: dateString,
            timeZone: TimeZone.current.identifier,
            now: Date(),
            thresholds: thresholds
        )

        let unread = try await APIClient.shared.fetchInbox(
            profileId: profileID,
            onlyUnread: true,
            limit: 50,
            offset: 0
        )

        await LocalNotificationScheduler.shared.schedule(fromInbox: unread, for: Date(), force: true)
        LocalNotificationScheduler.shared.updateBadge(unreadCount: unread.count)
    }

    private func loadThresholdsForGeneration() async -> GenerateThresholds {
        do {
            let response = try await APIClient.shared.fetchSettings()
            let settings = response.settings
            return GenerateThresholds(
                sleepMinMinutes: settings.minSleepMinutes,
                stepsMin: settings.minSteps,
                activeEnergyMinKcal: settings.minActiveEnergyKcal
            )
        } catch {
            return GenerateThresholds(
                sleepMinMinutes: 420,
                stepsMin: 6000,
                activeEnergyMinKcal: 200
            )
        }
    }

    private func setStatus(_ status: SyncStatus, at date: Date?, errorMessage: String?) {
        lastSyncStatus = status
        lastSyncAt = date
        lastErrorMessage = errorMessage

        defaults.set(status.rawValue, forKey: Keys.lastSyncStatus)

        if let date {
            defaults.set(date, forKey: Keys.lastSyncAt)
        }

        if let errorMessage, !errorMessage.isEmpty {
            defaults.set(errorMessage, forKey: Keys.lastSyncError)
        } else {
            defaults.removeObject(forKey: Keys.lastSyncError)
        }
    }

    private func shortErrorMessage(_ error: Error) -> String {
        if let apiError = error as? APIError {
            switch apiError {
            case .unauthorized:
                return "Сессия истекла"
            case .rateLimited:
                return "Слишком много запросов"
            case .serverError(let code):
                return code
            default:
                return apiError.localizedDescription
            }
        }

        let message = error.localizedDescription
        if message.isEmpty {
            return "Неизвестная ошибка"
        }
        return message
    }

    private func apiDateString(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar(identifier: .gregorian)
        formatter.locale = Locale(identifier: "en_US_POSIX")
        formatter.timeZone = TimeZone.current
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: date)
    }
}
