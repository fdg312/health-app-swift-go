import Foundation

struct SyncPreferences {
    private let defaults = UserDefaults.standard

    private enum Keys {
        static let backgroundSyncEnabled = "background_sync_enabled"
        static let backgroundRemindersEnabled = "background_reminders_enabled"
        static let cachedOwnerProfileID = "cached_owner_profile_id"
    }

    var backgroundSyncEnabled: Bool {
        get {
            if defaults.object(forKey: Keys.backgroundSyncEnabled) == nil {
                return true
            }
            return defaults.bool(forKey: Keys.backgroundSyncEnabled)
        }
        set {
            defaults.set(newValue, forKey: Keys.backgroundSyncEnabled)
        }
    }

    var backgroundRemindersEnabled: Bool {
        get {
            if defaults.object(forKey: Keys.backgroundRemindersEnabled) == nil {
                return true
            }
            return defaults.bool(forKey: Keys.backgroundRemindersEnabled)
        }
        set {
            defaults.set(newValue, forKey: Keys.backgroundRemindersEnabled)
        }
    }

    var cachedOwnerProfileID: String? {
        get {
            let value = defaults.string(forKey: Keys.cachedOwnerProfileID)
            if let value, !value.isEmpty {
                return value
            }
            return nil
        }
        set {
            let trimmed = newValue?.trimmingCharacters(in: .whitespacesAndNewlines)
            if let trimmed, !trimmed.isEmpty {
                defaults.set(trimmed, forKey: Keys.cachedOwnerProfileID)
            } else {
                defaults.removeObject(forKey: Keys.cachedOwnerProfileID)
            }
        }
    }
}
