import SwiftUI

struct ChatView: View {
  @ObservedObject private var auth = AuthManager.shared

  @State private var profiles: [ProfileDTO] = []
  @State private var messages: [ChatMessageDTO] = []
  @State private var pendingProposals: [ProposalDTO] = []
  @State private var resolvedProposals: [ProposalDTO] = []

  @State private var inputText = ""
  @State private var isLoading = false
  @State private var isSending = false
  @State private var inFlightProposalIDs = Set<UUID>()

  @State private var alertMessage = ""
  @State private var showErrorAlert = false
  @State private var infoMessage = ""
  @State private var showInfoAlert = false
  @State private var showRateLimited = false

  private var ownerProfile: ProfileDTO? {
    profiles.first(where: { $0.type == "owner" })
  }

  var body: some View {
    NavigationStack {
      VStack(spacing: 0) {
        if isLoading {
          Spacer()
          ProgressView("Загружаю чат...")
          Spacer()
        } else {
          messagesList
          composer
        }
      }
      .navigationTitle("Чат")
      .task {
        await loadChat()
      }
      .refreshable {
        await loadChat()
      }
      .alert("Ошибка", isPresented: $showErrorAlert) {
        Button("OK", role: .cancel) {}
      } message: {
        Text(alertMessage)
      }
      .alert("Готово", isPresented: $showInfoAlert) {
        Button("OK", role: .cancel) {}
      } message: {
        Text(infoMessage)
      }
      .alert("Слишком много запросов", isPresented: $showRateLimited) {
        Button("OK", role: .cancel) {}
      } message: {
        Text("Попробуйте позже")
      }
    }
  }

  private var messagesList: some View {
    ScrollViewReader { proxy in
      ScrollView {
        LazyVStack(alignment: .leading, spacing: 12) {
          if messages.isEmpty && pendingProposals.isEmpty && resolvedProposals.isEmpty {
            emptyState
          } else {
            ForEach(messages) { message in
              MessageRow(message: message)
            }
            if !pendingProposals.isEmpty {
              proposalsSection(title: "Предложения", proposals: pendingProposals, interactive: true)
            }
            if !resolvedProposals.isEmpty {
              proposalsSection(
                title: "Обработанные", proposals: resolvedProposals, interactive: false)
            }
          }
        }
        .padding()
      }
      .onChange(of: messages.count) { _, _ in
        if let lastID = messages.last?.id {
          withAnimation {
            proxy.scrollTo(lastID, anchor: .bottom)
          }
        }
      }
    }
  }

  private func proposalsSection(title: String, proposals: [ProposalDTO], interactive: Bool)
    -> some View
  {
    VStack(alignment: .leading, spacing: 8) {
      Text(title)
        .font(.caption)
        .foregroundStyle(.secondary)

      ForEach(proposals) { proposal in
        if interactive {
          ProposalCard(
            proposal: proposal,
            isLoading: inFlightProposalIDs.contains(proposal.id),
            onApply: { await applyProposal(proposal) },
            onReject: { await rejectProposal(proposal) }
          )
        } else {
          ProposalCard(
            proposal: proposal,
            isLoading: inFlightProposalIDs.contains(proposal.id),
            onApply: nil,
            onReject: nil
          )
        }
      }
    }
    .padding(.top, 4)
  }

  private var emptyState: some View {
    VStack(spacing: 8) {
      Text("Напишите сообщение, чтобы получить рекомендации")
        .font(.subheadline)
        .foregroundStyle(.secondary)
    }
    .frame(maxWidth: .infinity, minHeight: 220)
  }

  private var composer: some View {
    VStack(spacing: 8) {
      Divider()
      HStack(spacing: 8) {
        TextField("Введите сообщение...", text: $inputText, axis: .vertical)
          .lineLimit(1...4)
          .textFieldStyle(.roundedBorder)
          .disabled(isSending || ownerProfile == nil)

        Button {
          Task { await sendMessage() }
        } label: {
          if isSending {
            ProgressView()
              .controlSize(.small)
          } else {
            Image(systemName: "paperplane.fill")
          }
        }
        .buttonStyle(.borderedProminent)
        .disabled(
          isSending || ownerProfile == nil
            || inputText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
      }
      .padding(.horizontal)
      .padding(.vertical, 8)
    }
  }

  private func loadChat() async {
    isLoading = true
    defer { isLoading = false }

    do {
      let loadedProfiles = try await APIClient.shared.listProfiles()
      profiles = loadedProfiles

      guard let owner = loadedProfiles.first(where: { $0.type == "owner" }) else {
        messages = []
        pendingProposals = []
        resolvedProposals = []
        return
      }

      let messagesResponse = try await APIClient.shared.listChatMessages(
        profileId: owner.id,
        limit: 50,
        before: nil
      )
      messages = messagesResponse.messages.sorted(by: { $0.createdAt < $1.createdAt })
      pendingProposals = try await APIClient.shared.listProposals(
        profileId: owner.id,
        status: "pending",
        limit: 20
      )
    } catch {
      if !handleCommonError(error) {
        showError(prefix: "Не удалось загрузить чат", error: error)
      }
    }
  }

  private func sendMessage() async {
    guard let owner = ownerProfile else {
      alertMessage = "Профиль владельца не найден"
      showErrorAlert = true
      return
    }

    let content = inputText.trimmingCharacters(in: .whitespacesAndNewlines)
    guard !content.isEmpty else { return }

    let optimistic = ChatMessageDTO(
      id: UUID(),
      role: "user",
      content: content,
      createdAt: Date()
    )
    messages.append(optimistic)
    inputText = ""
    isSending = true

    defer { isSending = false }

    do {
      let response = try await APIClient.shared.sendChatMessage(
        profileId: owner.id, content: content)
      messages.append(response.assistantMessage)
      if !response.proposals.isEmpty {
        upsertPendingProposals(response.proposals)
      }
    } catch {
      messages.removeAll(where: { $0.id == optimistic.id })
      if !handleCommonError(error) {
        showError(prefix: "Не удалось отправить сообщение", error: error)
      }
    }
  }

  private func applyProposal(_ proposal: ProposalDTO) async {
    if inFlightProposalIDs.contains(proposal.id) {
      return
    }
    inFlightProposalIDs.insert(proposal.id)
    defer { inFlightProposalIDs.remove(proposal.id) }

    do {
      let response = try await APIClient.shared.applyProposal(id: proposal.id)
      markProposalResolved(proposal, status: response.status)
      if proposal.kind == "settings_update" {
        infoMessage = "Настройки обновлены"
        showInfoAlert = true
      } else if proposal.kind == "vitamins_schedule" {
        infoMessage = "Расписание витаминов обновлено"
        showInfoAlert = true
      } else if proposal.kind == "workout_plan" {
        infoMessage = "План тренировок создан"
        showInfoAlert = true
      } else if proposal.kind == "nutrition_plan" {
        infoMessage = "Цели по питанию обновлены"
        showInfoAlert = true
      } else if proposal.kind == "meal_plan" {
        infoMessage = "План питания обновлён"
        showInfoAlert = true
      }
    } catch {
      if !handleCommonError(error) {
        showError(prefix: "Не удалось применить предложение", error: error)
      }
    }
  }

  private func rejectProposal(_ proposal: ProposalDTO) async {
    if inFlightProposalIDs.contains(proposal.id) {
      return
    }
    inFlightProposalIDs.insert(proposal.id)
    defer { inFlightProposalIDs.remove(proposal.id) }

    do {
      let response = try await APIClient.shared.rejectProposal(id: proposal.id)
      markProposalResolved(proposal, status: response.status)
    } catch {
      if !handleCommonError(error) {
        showError(prefix: "Не удалось отклонить предложение", error: error)
      }
    }
  }

  private func upsertPendingProposals(_ proposals: [ProposalDTO]) {
    for proposal in proposals {
      pendingProposals.removeAll(where: { $0.id == proposal.id })
      if proposal.status == "pending" {
        pendingProposals.insert(proposal, at: 0)
      }
    }
  }

  private func markProposalResolved(_ proposal: ProposalDTO, status: String) {
    pendingProposals.removeAll(where: { $0.id == proposal.id })

    var updated = proposal
    updated.status = status
    resolvedProposals.removeAll(where: { $0.id == updated.id })
    resolvedProposals.insert(updated, at: 0)
    if resolvedProposals.count > 20 {
      resolvedProposals = Array(resolvedProposals.prefix(20))
    }
  }

  private func showError(prefix: String, error: Error) {
    if let apiError = error as? APIError {
      alertMessage = "\(prefix) (\(apiError.uiCode))"
    } else {
      alertMessage = "\(prefix) (bad_response)"
    }
    showErrorAlert = true
  }

  private func handleCommonError(_ error: Error) -> Bool {
    guard let apiError = error as? APIError else {
      return false
    }

    switch apiError {
    case .unauthorized:
      auth.handleUnauthorized()
      return true
    case .rateLimited:
      showRateLimited = true
      return true
    case .serverError(let code):
      alertMessage = "Ошибка сервера: \(code)"
      showErrorAlert = true
      return true
    default:
      return false
    }
  }
}

private struct MessageRow: View {
  let message: ChatMessageDTO

  var isUser: Bool {
    message.role == "user"
  }

  var body: some View {
    HStack {
      if isUser { Spacer() }
      VStack(alignment: .leading, spacing: 4) {
        Text(message.content)
          .font(.body)
        Text(message.createdAt.formatted(date: .omitted, time: .shortened))
          .font(.caption2)
          .foregroundStyle(.secondary)
      }
      .padding(12)
      .background(isUser ? Color.blue : Color(.secondarySystemBackground))
      .foregroundStyle(isUser ? Color.white : Color.primary)
      .clipShape(RoundedRectangle(cornerRadius: 14))
      .frame(maxWidth: 290, alignment: isUser ? .trailing : .leading)
      .id(message.id)

      if !isUser { Spacer() }
    }
  }
}

private struct ProposalCard: View {
  let proposal: ProposalDTO
  let isLoading: Bool
  let onApply: (() async -> Void)?
  let onReject: (() async -> Void)?

  private var canInteract: Bool {
    (proposal.kind == "settings_update" || proposal.kind == "vitamins_schedule"
      || proposal.kind == "workout_plan" || proposal.kind == "nutrition_plan")
      && proposal.status == "pending" && onApply != nil
      && onReject != nil
  }

  var body: some View {
    VStack(alignment: .leading, spacing: 8) {
      Text(proposal.title)
        .font(.subheadline.bold())

      Text(proposal.summary)
        .font(.caption)
        .foregroundStyle(.secondary)

      HStack {
        Text(kindLabel)
          .font(.caption2)
          .padding(.horizontal, 8)
          .padding(.vertical, 4)
          .background(Color(.tertiarySystemBackground))
          .clipShape(Capsule())

        Text(statusLabel)
          .font(.caption2)
          .padding(.horizontal, 8)
          .padding(.vertical, 4)
          .background(Color(.tertiarySystemBackground))
          .clipShape(Capsule())

        Spacer()

        if isLoading {
          ProgressView()
            .controlSize(.small)
        } else if canInteract {
          Button("Отклонить") {
            guard let onReject else { return }
            Task { await onReject() }
          }
          .buttonStyle(.bordered)

          Button("Применить") {
            guard let onApply else { return }
            Task { await onApply() }
          }
          .buttonStyle(.borderedProminent)
        } else {
          Button("Отклонить") {}
            .buttonStyle(.bordered)
            .disabled(true)
          Button("Применить") {}
            .buttonStyle(.borderedProminent)
            .disabled(true)
        }
      }
    }
    .padding(10)
    .frame(maxWidth: .infinity, alignment: .leading)
    .background(Color(.systemBackground))
    .overlay(
      RoundedRectangle(cornerRadius: 12)
        .stroke(Color(.separator), lineWidth: 0.6)
    )
  }

  private var kindLabel: String {
    switch proposal.kind {
    case "settings_update":
      return "Пороги"
    case "vitamins_schedule":
      return "Витамины"
    case "workout_plan":
      return "Тренировки"
    case "nutrition_plan":
      return "Питание"
    default:
      return "Общее"
    }
  }

  private var statusLabel: String {
    switch proposal.status {
    case "applied":
      return "Применено"
    case "rejected":
      return "Отклонено"
    default:
      return "Ожидает"
    }
  }
}

struct ChatView_Previews: PreviewProvider {
  static var previews: some View {
    ChatView()
  }
}
