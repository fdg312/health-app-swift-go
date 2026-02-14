import SwiftUI
import PhotosUI

struct CheckinAttachmentsView: View {
    let profileId: UUID
    let checkinId: UUID?
    let maxCount: Int

    @State private var attachments: [SourceDTO] = []
    @State private var isUploading = false
    @State private var uploadError: String?
    @State private var selectedPhotoItem: PhotosPickerItem?
    @State private var previewSource: SourceDTO?
    @State private var sourceToDelete: SourceDTO?
    @State private var showDeleteConfirm = false
    @State private var showCreateLink = false
    @State private var showCreateNote = false

    private var imageAttachments: [SourceDTO] {
        attachments.filter { $0.kind == "image" }
    }

    private var otherAttachments: [SourceDTO] {
        attachments.filter { $0.kind != "image" }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Вложения")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
                Spacer()
                if checkinId != nil {
                    Text("Фото: \(imageAttachments.count)/\(maxCount)")
                        .font(.caption)
                        .foregroundStyle(imageAttachments.count >= maxCount ? .orange : .secondary)
                }
            }

            if checkinId == nil {
                Text("Сначала сохраните чек-ин, затем можно прикрепить вложения.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .padding(.vertical, 8)
            } else {
                VStack(alignment: .leading, spacing: 12) {
                    // Action buttons
                    HStack(spacing: 8) {
                        PhotosPicker(
                            selection: $selectedPhotoItem,
                            matching: .images,
                            photoLibrary: .shared()
                        ) {
                            Label("Фото", systemImage: "photo")
                                .font(.caption)
                        }
                        .buttonStyle(.bordered)
                        .disabled(isUploading || imageAttachments.count >= maxCount)
                        .onChange(of: selectedPhotoItem) { _, newItem in
                            if let newItem = newItem {
                                Task { await uploadPhoto(newItem) }
                            }
                        }

                        Button {
                            showCreateLink = true
                        } label: {
                            Label("Ссылка", systemImage: "link")
                                .font(.caption)
                        }
                        .buttonStyle(.bordered)
                        .disabled(isUploading)

                        Button {
                            showCreateNote = true
                        } label: {
                            Label("Заметка", systemImage: "note.text")
                                .font(.caption)
                        }
                        .buttonStyle(.bordered)
                        .disabled(isUploading)

                        if isUploading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }

                    if imageAttachments.count >= maxCount {
                        Text("Достигнут лимит \(maxCount) фото.")
                            .font(.caption)
                            .foregroundStyle(.orange)
                    }

                    // Upload error
                    if let error = uploadError {
                        Text(error)
                            .font(.caption)
                            .foregroundStyle(.red)
                    }

                    // Photo grid
                    if !imageAttachments.isEmpty {
                        LazyVGrid(columns: [
                            GridItem(.flexible()),
                            GridItem(.flexible()),
                            GridItem(.flexible()),
                            GridItem(.flexible())
                        ], spacing: 8) {
                            ForEach(imageAttachments) { source in
                                photoThumbnail(source)
                            }
                        }
                    }

                    // Links and notes list
                    if !otherAttachments.isEmpty {
                        VStack(spacing: 4) {
                            ForEach(otherAttachments) { source in
                                nonImageRow(source)
                            }
                        }
                    }
                }
            }
        }
        .task(id: checkinId) {
            await loadAttachments()
        }
        .sheet(item: $previewSource) { source in
            if source.kind == "image" {
                PhotoPreviewSheet(source: source)
            } else {
                NotePreviewSheet(source: source)
            }
        }
        .sheet(isPresented: $showCreateLink) {
            CreateCheckinSourceSheet(
                type: .link,
                profileId: profileId,
                checkinId: checkinId,
                onSave: { await loadAttachments() }
            )
        }
        .sheet(isPresented: $showCreateNote) {
            CreateCheckinSourceSheet(
                type: .note,
                profileId: profileId,
                checkinId: checkinId,
                onSave: { await loadAttachments() }
            )
        }
        .alert("Удалить вложение?", isPresented: $showDeleteConfirm) {
            Button("Удалить", role: .destructive) {
                if let source = sourceToDelete {
                    Task { await deletePhoto(source) }
                }
            }
            Button("Отмена", role: .cancel) {}
        }
    }

    @ViewBuilder
    private func photoThumbnail(_ source: SourceDTO) -> some View {
        let imageURL = URL(string: "\(AppConfig.apiBaseURL)/v1/sources/\(source.id.uuidString)/download")

        ZStack(alignment: .topTrailing) {
            AsyncImage(url: imageURL) { phase in
                switch phase {
                case .empty:
                    ProgressView()
                        .frame(width: 90, height: 90)
                case .success(let image):
                    image
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(width: 90, height: 90)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                        .onTapGesture {
                            previewSource = source
                        }
                case .failure:
                    Image(systemName: "photo")
                        .foregroundStyle(.gray)
                        .frame(width: 90, height: 90)
                        .background(Color.gray.opacity(0.1))
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                @unknown default:
                    EmptyView()
                }
            }

            // Delete button
            Button {
                sourceToDelete = source
                showDeleteConfirm = true
            } label: {
                Image(systemName: "xmark.circle.fill")
                    .font(.caption)
                    .foregroundStyle(.white)
                    .background(Circle().fill(Color.red))
            }
            .offset(x: 4, y: -4)
        }
    }

    @ViewBuilder
    private func nonImageRow(_ source: SourceDTO) -> some View {
        HStack(spacing: 8) {
            Image(systemName: source.kind == "link" ? "link" : "note.text")
                .foregroundStyle(source.kind == "link" ? .purple : .orange)
                .frame(width: 20)

            VStack(alignment: .leading, spacing: 2) {
                if let title = source.title, !title.isEmpty {
                    Text(title)
                        .font(.caption)
                        .lineLimit(1)
                } else if let url = source.url {
                    Text(url)
                        .font(.caption)
                        .lineLimit(1)
                } else if let text = source.text {
                    Text(text)
                        .font(.caption)
                        .lineLimit(1)
                }
            }

            Spacer()

            Button {
                sourceToDelete = source
                showDeleteConfirm = true
            } label: {
                Image(systemName: "trash")
                    .font(.caption)
                    .foregroundStyle(.red)
            }
            .buttonStyle(.plain)
        }
        .padding(.vertical, 4)
        .padding(.horizontal, 8)
        .background(Color.gray.opacity(0.1))
        .clipShape(RoundedRectangle(cornerRadius: 6))
        .contentShape(Rectangle())
        .onTapGesture {
            previewSource = source
        }
    }

    private func loadAttachments() async {
        guard let checkinId = checkinId else {
            attachments = []
            return
        }

        do {
            attachments = try await APIClient.shared.listSources(profileId: profileId, checkinId: checkinId)
        } catch {
            // Silently fail - attachments are non-critical
            attachments = []
        }
    }

    private func uploadPhoto(_ photoItem: PhotosPickerItem) async {
        isUploading = true
        uploadError = nil

        do {
            guard let imageData = try await photoItem.loadTransferable(type: Data.self) else {
                uploadError = "Не удалось загрузить фото"
                isUploading = false
                return
            }

            // Convert to JPEG if needed
            let jpegData: Data
            if let uiImage = UIImage(data: imageData) {
                jpegData = uiImage.jpegData(compressionQuality: 0.8) ?? imageData
            } else {
                jpegData = imageData
            }

            _ = try await APIClient.shared.uploadSourceImage(
                profileId: profileId,
                checkinId: checkinId,
                imageData: jpegData,
                title: nil
            )

            await loadAttachments()
        } catch {
            uploadError = "Ошибка загрузки: \(error.localizedDescription)"
        }

        isUploading = false
        selectedPhotoItem = nil
    }

    private func deletePhoto(_ source: SourceDTO) async {
        do {
            try await APIClient.shared.deleteSource(sourceId: source.id)
            await loadAttachments()
        } catch {
            uploadError = "Ошибка удаления: \(error.localizedDescription)"
        }
    }
}

// MARK: - Create Source Sheet (for Checkin)

struct CreateCheckinSourceSheet: View {
    enum SourceType {
        case link, note
    }

    let type: SourceType
    let profileId: UUID
    let checkinId: UUID?
    let onSave: () async -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var title = ""
    @State private var url = ""
    @State private var text = ""
    @State private var isSaving = false
    @State private var errorMessage: String?

    var body: some View {
        NavigationStack {
            Form {
                if type == .link {
                    Section("Ссылка") {
                        TextField("Название (опционально)", text: $title)
                        TextField("URL", text: $url)
                            .keyboardType(.URL)
                            .autocapitalization(.none)
                            .autocorrectionDisabled()
                    }
                } else {
                    Section("Заметка") {
                        TextField("Название (опционально)", text: $title)
                        TextEditor(text: $text)
                            .frame(minHeight: 100)
                    }
                }

                if let error = errorMessage {
                    Section {
                        Text(error)
                            .font(.caption)
                            .foregroundStyle(.red)
                    }
                }
            }
            .navigationTitle(type == .link ? "Добавить ссылку" : "Добавить заметку")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Отмена") {
                        dismiss()
                    }
                    .disabled(isSaving)
                }

                ToolbarItem(placement: .confirmationAction) {
                    Button {
                        Task { await saveSource() }
                    } label: {
                        if isSaving {
                            ProgressView()
                        } else {
                            Text("Сохранить")
                        }
                    }
                    .disabled(!isValid || isSaving)
                }
            }
        }
    }

    private var isValid: Bool {
        if type == .link {
            return !url.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        } else {
            return !text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        }
    }

    private func saveSource() async {
        isSaving = true
        errorMessage = nil

        do {
            let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
            let titleValue = trimmedTitle.isEmpty ? nil : trimmedTitle

            if type == .link {
                _ = try await APIClient.shared.createSourceLink(
                    profileId: profileId,
                    checkinId: checkinId,
                    title: titleValue,
                    url: url.trimmingCharacters(in: .whitespacesAndNewlines)
                )
            } else {
                _ = try await APIClient.shared.createSourceNote(
                    profileId: profileId,
                    checkinId: checkinId,
                    title: titleValue,
                    text: text.trimmingCharacters(in: .whitespacesAndNewlines)
                )
            }

            await onSave()
            dismiss()
        } catch {
            errorMessage = "Ошибка сохранения: \(error.localizedDescription)"
        }

        isSaving = false
    }
}

// MARK: - Photo Preview Sheet

struct PhotoPreviewSheet: View {
    let source: SourceDTO
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ZStack {
                Color.black.ignoresSafeArea()

                AsyncImage(url: URL(string: "\(AppConfig.apiBaseURL)/v1/sources/\(source.id.uuidString)/download")) { phase in
                    switch phase {
                    case .empty:
                        ProgressView()
                            .tint(.white)
                    case .success(let image):
                        image
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                    case .failure:
                        VStack {
                            Image(systemName: "exclamationmark.triangle")
                                .font(.largeTitle)
                                .foregroundStyle(.white)
                            Text("Не удалось загрузить")
                                .foregroundStyle(.white)
                        }
                    @unknown default:
                        EmptyView()
                    }
                }
            }
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Закрыть") {
                        dismiss()
                    }
                    .foregroundStyle(.white)
                }
            }
        }
    }
}

// MARK: - Note Preview Sheet (reused from ActivityView)

struct NotePreviewSheet: View {
    let source: SourceDTO
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 16) {
                    if let title = source.title, !title.isEmpty {
                        Text(title)
                            .font(.title2)
                            .fontWeight(.bold)
                    }

                    if source.kind == "link", let url = source.url, let parsedURL = URL(string: url) {
                        Link(destination: parsedURL) {
                            HStack {
                                Image(systemName: "link")
                                Text(url)
                                    .foregroundStyle(.blue)
                            }
                        }
                    } else if source.kind == "link", let url = source.url {
                        Text(url)
                            .font(.body)
                            .foregroundStyle(.secondary)
                    }

                    if let text = source.text, !text.isEmpty {
                        Text(text)
                            .font(.body)
                    }

                    Text("Создано: \(formatFullDate(source.createdAt))")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .padding()
                .frame(maxWidth: .infinity, alignment: .leading)
            }
            .navigationTitle(source.kind == "link" ? "Ссылка" : "Заметка")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Закрыть") {
                        dismiss()
                    }
                }
            }
        }
    }

    private func formatFullDate(_ date: Date) -> String {
        let formatter = DateFormatter()
        formatter.dateStyle = .medium
        formatter.timeStyle = .short
        formatter.locale = Locale(identifier: "ru_RU")
        return formatter.string(from: date)
    }
}
