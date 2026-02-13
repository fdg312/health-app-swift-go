import Foundation

// MARK: - Source Models

struct SourceDTO: Codable, Identifiable {
  let id: UUID
  let profileId: UUID
  let kind: String  // "link", "note", "image"
  let title: String?
  let text: String?
  let url: String?
  let checkinId: UUID?
  let contentType: String?  // MIME type for images
  let sizeBytes: Int64?
  let createdAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case profileId = "profile_id"
    case kind, title, text, url
    case checkinId = "checkin_id"
    case contentType = "content_type"
    case sizeBytes = "size_bytes"
    case createdAt = "created_at"
  }
}

struct SourcesResponse: Decodable {
  let sources: [SourceDTO]
  let limit: Int?
  let offset: Int?
  let total: Int?

  private enum CodingKeys: String, CodingKey {
    case sources
    case items
    case data
    case limit
    case offset
    case total
    case pagination
  }

  private enum PaginationKeys: String, CodingKey {
    case limit
    case offset
    case total
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)

    if let value = try? container.decode([SourceDTO].self, forKey: .sources) {
      self.sources = value
    } else if let value = try? container.decode([SourceDTO].self, forKey: .items) {
      self.sources = value
    } else if let value = try? container.decode([SourceDTO].self, forKey: .data) {
      self.sources = value
    } else if container.contains(.data) {
      let dataContainer = try container.nestedContainer(keyedBy: CodingKeys.self, forKey: .data)
      if let value = try? dataContainer.decode([SourceDTO].self, forKey: .sources) {
        self.sources = value
      } else if let value = try? dataContainer.decode([SourceDTO].self, forKey: .items) {
        self.sources = value
      } else if let value = try? dataContainer.decode([SourceDTO].self, forKey: .data) {
        self.sources = value
      } else {
        throw DecodingError.keyNotFound(
          CodingKeys.sources,
          DecodingError.Context(
            codingPath: decoder.codingPath,
            debugDescription: "Expected one of data.sources/data.items/data.data"
          )
        )
      }
    } else {
      throw DecodingError.keyNotFound(
        CodingKeys.sources,
        DecodingError.Context(
          codingPath: decoder.codingPath,
          debugDescription: "Expected one of sources/items/data"
        )
      )
    }

    var decodedLimit = try container.decodeIfPresent(Int.self, forKey: .limit)
    var decodedOffset = try container.decodeIfPresent(Int.self, forKey: .offset)
    var decodedTotal = try container.decodeIfPresent(Int.self, forKey: .total)

    if container.contains(.pagination) {
      let pagination = try container.nestedContainer(
        keyedBy: PaginationKeys.self, forKey: .pagination)
      if decodedLimit == nil {
        decodedLimit = try pagination.decodeIfPresent(Int.self, forKey: .limit)
      }
      if decodedOffset == nil {
        decodedOffset = try pagination.decodeIfPresent(Int.self, forKey: .offset)
      }
      if decodedTotal == nil {
        decodedTotal = try pagination.decodeIfPresent(Int.self, forKey: .total)
      }
    }

    self.limit = decodedLimit
    self.offset = decodedOffset
    self.total = decodedTotal
  }
}

struct CreateSourceRequest: Codable {
  let profileId: UUID
  let kind: String
  let title: String?
  let text: String?
  let url: String?
  let checkinId: UUID?

  enum CodingKeys: String, CodingKey {
    case profileId = "profile_id"
    case kind, title, text, url
    case checkinId = "checkin_id"
  }
}
