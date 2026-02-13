import Foundation

struct ChatMessageDTO: Codable, Identifiable {
  let id: UUID
  let role: String
  let content: String
  let createdAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case role
    case content
    case createdAt = "created_at"
  }
}

struct SendChatMessageRequest: Codable {
  let profileId: UUID
  let content: String

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case content
  }
}

struct SendChatMessageResponse: Codable {
  let assistantMessage: ChatMessageDTO
  let proposals: [ProposalDTO]

  enum CodingKeys: String, CodingKey {
    case assistantMessage = "assistant_message"
    case proposals
  }
}

struct ListChatMessagesResponse: Codable {
  let messages: [ChatMessageDTO]
  let nextCursor: String?

  enum CodingKeys: String, CodingKey {
    case messages
    case nextCursor = "next_cursor"
  }
}
