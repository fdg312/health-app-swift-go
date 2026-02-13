import Foundation
import SwiftUI
import UserNotifications

@MainActor
class LocalNotificationScheduler {
    static let shared = LocalNotificationScheduler()

    private let center = UNUserNotificationCenter.current()
    private let settings = NotificationSettings()

    // Rate limiting: track last schedule time
    @AppStorage("lastScheduleTimestamp") private var lastScheduleTimestamp: Double = 0
    private let minScheduleInterval: TimeInterval = 30 * 60 // 30 minutes

    private init() {}

    // MARK: - Permission

    func requestPermissionIfNeeded() async -> Bool {
        do {
            let currentSettings = await center.notificationSettings()

            if currentSettings.authorizationStatus == .notDetermined {
                return try await center.requestAuthorization(options: [.alert, .sound, .badge])
            }

            return currentSettings.authorizationStatus == .authorized
        } catch {
            print("Failed to request notification permission: \(error)")
            return false
        }
    }

    // MARK: - Scheduling

    func rescheduleForToday(profileId: UUID) async {
        // Check if reminders are enabled
        guard settings.remindersEnabled else {
            // Cancel all scheduled notifications
            await cancelAllNotifications()
            return
        }

        // Rate limiting: don't reschedule too often
        let now = Date()
        if now.timeIntervalSince1970 - lastScheduleTimestamp < minScheduleInterval {
            print("Skipping reschedule: too soon since last schedule")
            return
        }

        // Check permission
        let hasPermission = await requestPermissionIfNeeded()
        guard hasPermission else {
            print("No notification permission")
            return
        }

        // Get today's date
        let calendar = Calendar.current
        let today = calendar.startOfDay(for: now)
        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd"
        let dateString = dateFormatter.string(from: today)

        do {
            // 1. Generate inbox on server for today
            let thresholds = GenerateThresholds(
                sleepMinMinutes: 420,
                stepsMin: 6000,
                activeEnergyMinKcal: 200
            )

            _ = try await APIClient.shared.generateNotifications(
                profileId: profileId,
                date: dateString,
                timeZone: TimeZone.current.identifier,
                now: now,
                thresholds: thresholds
            )

            // 2. Fetch unread notifications from inbox
            let allNotifications = try await APIClient.shared.fetchInbox(
                profileId: profileId,
                onlyUnread: true,
                limit: 20,
                offset: 0
            )

            // 3. Schedule local reminders from unread inbox
            await schedule(fromInbox: allNotifications, for: today)

        } catch {
            print("Failed to reschedule notifications: \(error)")
        }
    }

    /// Build local notifications from inbox entities for a specific day.
    func schedule(fromInbox notifications: [NotificationDTO], for day: Date = Date(), force: Bool = false) async {
        guard settings.remindersEnabled else {
            await cancelAllNotifications()
            return
        }

        let now = Date()
        if !force && now.timeIntervalSince1970 - lastScheduleTimestamp < minScheduleInterval {
            #if DEBUG
            print("Skipping schedule(fromInbox): too soon since last schedule")
            #endif
            return
        }

        let hasPermission = await requestPermissionIfNeeded()
        guard hasPermission else {
            #if DEBUG
            print("No notification permission for schedule(fromInbox)")
            #endif
            return
        }

        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let dateString = formatter.string(from: day)

        let todayNotifications = notifications.filter { notification in
            guard let sourceDate = notification.sourceDate else { return false }
            return sourceDate == dateString
        }

        var scheduledItems: [(notification: NotificationDTO, scheduledTime: Date)] = []
        for notification in todayNotifications {
            guard let scheduledTime = getScheduledTime(for: notification.kind, on: day) else {
                continue
            }

            if scheduledTime > now {
                scheduledItems.append((notification, scheduledTime))
            }
        }

        scheduledItems.sort { item1, item2 in
            if item1.notification.severity == "warn" && item2.notification.severity != "warn" {
                return true
            }
            if item1.notification.severity != "warn" && item2.notification.severity == "warn" {
                return false
            }
            return item1.scheduledTime < item2.scheduledTime
        }

        let maxCount = settings.maxLocalPerDay
        if scheduledItems.count > maxCount {
            scheduledItems = Array(scheduledItems.prefix(maxCount))
        }

        await cancelAllNotifications()

        for item in scheduledItems {
            await scheduleNotification(notification: item.notification, at: item.scheduledTime)
        }

        lastScheduleTimestamp = now.timeIntervalSince1970
        #if DEBUG
        print("Scheduled \(scheduledItems.count) notifications from inbox")
        #endif
    }

    // MARK: - Private Helpers

    private func getScheduledTime(for kind: String, on date: Date) -> Date? {
        let calendar = Calendar.current
        var components = calendar.dateComponents([.year, .month, .day], from: date)

        let minutes: Int
        switch kind {
        case "missing_morning_checkin":
            minutes = settings.morningCheckinMinutes
        case "missing_evening_checkin":
            minutes = settings.eveningCheckinMinutes
        case "low_activity":
            minutes = settings.activityNudgeMinutes
        case "low_sleep":
            minutes = settings.sleepReminderMinutes
        default:
            return nil
        }

        components.hour = minutes / 60
        components.minute = minutes % 60

        guard let scheduledTime = calendar.date(from: components) else {
            return nil
        }

        // Adjust for quiet mode
        return settings.adjustForQuietMode(scheduledTime)
    }

    private func scheduleNotification(notification: NotificationDTO, at scheduledTime: Date) async {
        let content = UNMutableNotificationContent()
        content.title = notification.title
        content.body = notification.body
        content.sound = .default

        // Add metadata
        content.userInfo = [
            "notification_id": notification.id.uuidString,
            "profile_id": notification.profileId.uuidString,
            "kind": notification.kind
        ]

        // Create date components for trigger
        let calendar = Calendar.current
        let components = calendar.dateComponents([.year, .month, .day, .hour, .minute], from: scheduledTime)
        let trigger = UNCalendarNotificationTrigger(dateMatching: components, repeats: false)

        // Create request with unique identifier
        let identifier = "inbox_\(notification.id.uuidString)"
        let request = UNNotificationRequest(identifier: identifier, content: content, trigger: trigger)

        do {
            try await center.add(request)
            print("Scheduled notification '\(notification.title)' at \(scheduledTime)")
        } catch {
            print("Failed to schedule notification: \(error)")
        }
    }

    private func cancelAllNotifications() async {
        // Remove all pending notifications scheduled by this app
        let pending = await center.pendingNotificationRequests()
        let identifiers = pending
            .filter { $0.identifier.hasPrefix("inbox_") }
            .map { $0.identifier }

        center.removePendingNotificationRequests(withIdentifiers: identifiers)
        print("Cancelled \(identifiers.count) pending notifications")
    }

    // MARK: - Badge Management

    func updateBadge(unreadCount: Int) {
        Task { @MainActor in
            // Check permission first
            let currentSettings = await center.notificationSettings()
            if currentSettings.badgeSetting == .enabled {
                try? await UNUserNotificationCenter.current().setBadgeCount(unreadCount)
            }
        }
    }

    func clearBadge() {
        Task { @MainActor in
            try? await UNUserNotificationCenter.current().setBadgeCount(0)
        }
    }
}
