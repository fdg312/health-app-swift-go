import SwiftUI

struct NotificationSettings {
    @AppStorage("remindersEnabled") var remindersEnabled: Bool = false
    @AppStorage("quietModeEnabled") var quietModeEnabled: Bool = true
    
    // Quiet mode times (stored as minutes since midnight)
    @AppStorage("quietStartMinutes") var quietStartMinutes: Int = 23 * 60 // 23:00
    @AppStorage("quietEndMinutes") var quietEndMinutes: Int = 7 * 60 // 07:00
    
    @AppStorage("maxLocalPerDay") var maxLocalPerDay: Int = 4
    
    // Reminder times (stored as minutes since midnight)
    @AppStorage("morningCheckinMinutes") var morningCheckinMinutes: Int = 12 * 60 // 12:00
    @AppStorage("eveningCheckinMinutes") var eveningCheckinMinutes: Int = 21 * 60 + 30 // 21:30
    @AppStorage("activityNudgeMinutes") var activityNudgeMinutes: Int = 18 * 60 // 18:00
    @AppStorage("sleepReminderMinutes") var sleepReminderMinutes: Int = 11 * 60 // 11:00
    
    // Helper properties for time conversion
    var quietStart: Date {
        minutesToDate(quietStartMinutes)
    }
    
    var quietEnd: Date {
        minutesToDate(quietEndMinutes)
    }
    
    var morningCheckinTime: Date {
        minutesToDate(morningCheckinMinutes)
    }
    
    var eveningCheckinTime: Date {
        minutesToDate(eveningCheckinMinutes)
    }
    
    var activityNudgeTime: Date {
        minutesToDate(activityNudgeMinutes)
    }
    
    var sleepReminderTime: Date {
        minutesToDate(sleepReminderMinutes)
    }
    
    // Convert minutes since midnight to Date (today)
    private func minutesToDate(_ minutes: Int) -> Date {
        let calendar = Calendar.current
        let now = Date()
        let components = calendar.dateComponents([.year, .month, .day], from: now)
        
        let hour = minutes / 60
        let minute = minutes % 60
        
        var dateComponents = DateComponents()
        dateComponents.year = components.year
        dateComponents.month = components.month
        dateComponents.day = components.day
        dateComponents.hour = hour
        dateComponents.minute = minute
        
        return calendar.date(from: dateComponents) ?? now
    }
    
    // Convert Date to minutes since midnight
    func dateToMinutes(_ date: Date) -> Int {
        let calendar = Calendar.current
        let components = calendar.dateComponents([.hour, .minute], from: date)
        return (components.hour ?? 0) * 60 + (components.minute ?? 0)
    }
    
    // Update methods
    mutating func setQuietStart(_ date: Date) {
        quietStartMinutes = dateToMinutes(date)
    }
    
    mutating func setQuietEnd(_ date: Date) {
        quietEndMinutes = dateToMinutes(date)
    }
    
    mutating func setMorningCheckinTime(_ date: Date) {
        morningCheckinMinutes = dateToMinutes(date)
    }
    
    mutating func setEveningCheckinTime(_ date: Date) {
        eveningCheckinMinutes = dateToMinutes(date)
    }
    
    mutating func setActivityNudgeTime(_ date: Date) {
        activityNudgeMinutes = dateToMinutes(date)
    }
    
    mutating func setSleepReminderTime(_ date: Date) {
        sleepReminderMinutes = dateToMinutes(date)
    }
    
    // Check if a time falls within quiet mode
    func isInQuietMode(_ date: Date) -> Bool {
        guard quietModeEnabled else { return false }
        
        let minutes = dateToMinutes(date)
        
        // Handle overnight quiet mode (e.g., 23:00 to 07:00)
        if quietStartMinutes > quietEndMinutes {
            return minutes >= quietStartMinutes || minutes < quietEndMinutes
        } else {
            return minutes >= quietStartMinutes && minutes < quietEndMinutes
        }
    }
    
    // Adjust time to avoid quiet mode (shift to quietEnd)
    func adjustForQuietMode(_ date: Date) -> Date {
        if !isInQuietMode(date) {
            return date
        }
        
        // Shift to quiet end time (same day)
        let calendar = Calendar.current
        let components = calendar.dateComponents([.year, .month, .day], from: date)
        
        let hour = quietEndMinutes / 60
        let minute = quietEndMinutes % 60
        
        var dateComponents = DateComponents()
        dateComponents.year = components.year
        dateComponents.month = components.month
        dateComponents.day = components.day
        dateComponents.hour = hour
        dateComponents.minute = minute
        
        return calendar.date(from: dateComponents) ?? date
    }
}
