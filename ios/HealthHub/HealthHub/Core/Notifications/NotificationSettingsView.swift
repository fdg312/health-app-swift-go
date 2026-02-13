import SwiftUI

struct NotificationSettingsView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var settings = NotificationSettings()
    
    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Toggle("Включить напоминания", isOn: $settings.remindersEnabled)
                } header: {
                    Text("Уведомления")
                } footer: {
                    Text("Локальные напоминания на основе серверных данных. Без push-уведомлений.")
                }
                
                if settings.remindersEnabled {
                    Section {
                        Toggle("Режим «Не беспокоить»", isOn: $settings.quietModeEnabled)
                        
                        if settings.quietModeEnabled {
                            DatePicker("Начало", selection: quietStartBinding, displayedComponents: .hourAndMinute)
                            DatePicker("Окончание", selection: quietEndBinding, displayedComponents: .hourAndMinute)
                        }
                    } header: {
                        Text("Режим «Не беспокоить»")
                    } footer: {
                        if settings.quietModeEnabled {
                            Text("Уведомления в это время будут перенесены на окончание периода.")
                        }
                    }
                    
                    Section {
                        Stepper("Максимум в день: \(settings.maxLocalPerDay)", value: $settings.maxLocalPerDay, in: 1...10)
                    } header: {
                        Text("Ограничения")
                    } footer: {
                        Text("Максимальное количество локальных уведомлений на день.")
                    }
                    
                    Section {
                        DatePicker("Утренний чек-ин", selection: morningCheckinBinding, displayedComponents: .hourAndMinute)
                        DatePicker("Вечерний чек-ин", selection: eveningCheckinBinding, displayedComponents: .hourAndMinute)
                        DatePicker("Активность (напоминание)", selection: activityNudgeBinding, displayedComponents: .hourAndMinute)
                        DatePicker("Сон (напоминание)", selection: sleepReminderBinding, displayedComponents: .hourAndMinute)
                    } header: {
                        Text("Время напоминаний")
                    } footer: {
                        Text("Когда отправлять напоминания для каждого типа.")
                    }
                }
            }
            .navigationTitle("Настройки уведомлений")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Готово") {
                        dismiss()
                    }
                }
            }
        }
    }
    
    // Bindings for time pickers
    private var quietStartBinding: Binding<Date> {
        Binding(
            get: { settings.quietStart },
            set: { settings.setQuietStart($0) }
        )
    }
    
    private var quietEndBinding: Binding<Date> {
        Binding(
            get: { settings.quietEnd },
            set: { settings.setQuietEnd($0) }
        )
    }
    
    private var morningCheckinBinding: Binding<Date> {
        Binding(
            get: { settings.morningCheckinTime },
            set: { settings.setMorningCheckinTime($0) }
        )
    }
    
    private var eveningCheckinBinding: Binding<Date> {
        Binding(
            get: { settings.eveningCheckinTime },
            set: { settings.setEveningCheckinTime($0) }
        )
    }
    
    private var activityNudgeBinding: Binding<Date> {
        Binding(
            get: { settings.activityNudgeTime },
            set: { settings.setActivityNudgeTime($0) }
        )
    }
    
    private var sleepReminderBinding: Binding<Date> {
        Binding(
            get: { settings.sleepReminderTime },
            set: { settings.setSleepReminderTime($0) }
        )
    }
}
