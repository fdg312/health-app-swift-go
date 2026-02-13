import SwiftUI

struct InboxSheet: View {
    let profileId: UUID
    let onDismiss: () -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var notifications: [NotificationDTO] = []
    @State private var isLoading = false
    @State private var errorMessage: String?
    @State private var notificationToDelete: NotificationDTO?
    @State private var showMarkReadConfirm = false

    var body: some View {
        NavigationStack {
            Group {
                if isLoading && notifications.isEmpty {
                    ProgressView("Загрузка...")
                } else if let error = errorMessage {
                    VStack(spacing: 8) {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.largeTitle)
                            .foregroundStyle(.orange)
                        Text("Ошибка")
                            .font(.headline)
                        Text(error)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .padding()
                } else if notifications.isEmpty {
                    VStack(spacing: 8) {
                        Image(systemName: "bell.slash")
                            .font(.largeTitle)
                            .foregroundStyle(.gray)
                        Text("Нет уведомлений")
                            .font(.headline)
                            .foregroundStyle(.secondary)
                    }
                    .padding()
                } else {
                    List {
                        ForEach(notifications) { notification in
                            notificationRow(notification)
                                .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                                    if notification.readAt == nil {
                                        Button {
                                            Task {
                                                await markRead([notification.id])
                                            }
                                        } label: {
                                            Label("Прочитано", systemImage: "checkmark")
                                        }
                                        .tint(.blue)
                                    }
                                }
                        }
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Уведомления")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Закрыть") {
                        onDismiss()
                        dismiss()
                    }
                }

                ToolbarItem(placement: .topBarTrailing) {
                    if notifications.contains(where: { $0.readAt == nil }) {
                        Button {
                            showMarkReadConfirm = true
                        } label: {
                            Image(systemName: "checkmark.circle")
                        }
                    }
                }
            }
            .refreshable {
                await loadNotifications()
            }
            .task {
                await loadNotifications()
            }
            .alert("Отметить всё прочитанным?", isPresented: $showMarkReadConfirm) {
                Button("Отметить", role: .none) {
                    Task { await markAllRead() }
                }
                Button("Отмена", role: .cancel) {}
            }
        }
    }

    @ViewBuilder
    private func notificationRow(_ notification: NotificationDTO) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                // Severity indicator
                Circle()
                    .fill(notification.severity == "warn" ? Color.orange : Color.blue)
                    .frame(width: 8, height: 8)

                // Title
                Text(notification.title)
                    .font(.subheadline)
                    .fontWeight(notification.readAt == nil ? .bold : .regular)
                    .foregroundStyle(.primary)

                Spacer()

                // Unread indicator
                if notification.readAt == nil {
                    Image(systemName: "circle.fill")
                        .font(.caption2)
                        .foregroundStyle(.blue)
                }
            }

            // Body
            Text(notification.body)
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineLimit(3)

            // Date info
            HStack {
                if let sourceDate = notification.sourceDate {
                    Text("Дата: \(sourceDate)")
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }

                Spacer()

                Text(formatDate(notification.createdAt))
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.vertical, 4)
        .opacity(notification.readAt == nil ? 1.0 : 0.6)
    }

    // MARK: - Actions

    private func loadNotifications() async {
        isLoading = true
        errorMessage = nil

        do {
            notifications = try await APIClient.shared.fetchInbox(
                profileId: profileId,
                onlyUnread: false,
                limit: 50,
                offset: 0
            )
        } catch {
            errorMessage = "Не удалось загрузить уведомления: \(error.localizedDescription)"
        }

        isLoading = false
    }

    private func markRead(_ ids: [UUID]) async {
        do {
            _ = try await APIClient.shared.markNotificationsRead(profileId: profileId, ids: ids)
            await loadNotifications()
        } catch {
            errorMessage = "Не удалось отметить: \(error.localizedDescription)"
        }
    }

    private func markAllRead() async {
        do {
            _ = try await APIClient.shared.markAllNotificationsRead(profileId: profileId)
            await loadNotifications()
        } catch {
            errorMessage = "Не удалось отметить всё: \(error.localizedDescription)"
        }
    }

    // MARK: - Helpers

    private func formatDate(_ date: Date) -> String {
        let formatter = RelativeDateTimeFormatter()
        formatter.unitsStyle = .short
        formatter.locale = Locale(identifier: "ru_RU")
        return formatter.localizedString(for: date, relativeTo: Date())
    }
}
