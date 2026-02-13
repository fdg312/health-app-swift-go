package sources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/fdg312/health-hub/internal/blob"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

var (
	ErrProfileNotFound    = errors.New("profile not found")
	ErrSourceNotFound     = errors.New("source not found")
	ErrInvalidKind        = errors.New("invalid source kind")
	ErrMissingURL         = errors.New("url required for link")
	ErrMissingText        = errors.New("text required for note")
	ErrFileTooLarge       = errors.New("file too large")
	ErrUnsupportedMime    = errors.New("unsupported mime type")
	ErrMaxSourcesExceeded = errors.New("max sources per checkin exceeded")
)

// ProfileStorageAdapter — адаптер для доступа к профилям
type ProfileStorageAdapter interface {
	GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error)
}

// Service handles sources business logic
type Service struct {
	sourcesStorage     storage.SourcesStorage
	profileStorage     ProfileStorageAdapter
	blobStore          blob.Store
	localMode          bool   // true if no S3 configured
	publicBaseURL      string // S3 public base URL (if prefer_public_url mode)
	preferPublicURL    bool   // if true, use public URLs instead of presigned
	maxUploadMB        int
	allowedMimes       []string
	maxSourcesPerCheck int
}

// NewService creates a new sources service
func NewService(
	sourcesStorage storage.SourcesStorage,
	profileStorage ProfileStorageAdapter,
	blobStore blob.Store,
	maxUploadMB int,
	allowedMimes string,
	maxSourcesPerCheck int,
	publicBaseURL string,
	preferPublicURL bool,
) *Service {
	localMode := (blobStore == nil)

	// Parse allowed mimes
	mimes := strings.Split(allowedMimes, ",")
	for i, m := range mimes {
		mimes[i] = strings.TrimSpace(m)
	}

	return &Service{
		sourcesStorage:     sourcesStorage,
		profileStorage:     profileStorage,
		blobStore:          blobStore,
		localMode:          localMode,
		publicBaseURL:      publicBaseURL,
		preferPublicURL:    preferPublicURL,
		maxUploadMB:        maxUploadMB,
		allowedMimes:       mimes,
		maxSourcesPerCheck: maxSourcesPerCheck,
	}
}

// CreateSource creates a link or note source
func (s *Service) CreateSource(ctx context.Context, req CreateSourceRequest) (*SourceDTO, error) {
	if err := s.ensureProfileAccess(ctx, req.ProfileID); err != nil {
		return nil, ErrProfileNotFound
	}

	// Validate kind
	if req.Kind != KindLink && req.Kind != KindNote {
		return nil, ErrInvalidKind
	}

	// Validate link
	if req.Kind == KindLink {
		if req.URL == nil || *req.URL == "" {
			return nil, ErrMissingURL
		}
	}

	// Validate note
	if req.Kind == KindNote {
		if req.Text == nil || *req.Text == "" {
			return nil, ErrMissingText
		}
	}

	// Create source
	source := &storage.Source{
		ProfileID: req.ProfileID,
		Kind:      req.Kind,
		Title:     req.Title,
		Text:      req.Text,
		URL:       req.URL,
		CheckinID: req.CheckinID,
	}

	if err := s.sourcesStorage.CreateSource(ctx, source); err != nil {
		return nil, err
	}

	return s.toDTO(source), nil
}

// CreateImageSource creates an image source from uploaded file
func (s *Service) CreateImageSource(ctx context.Context, profileID uuid.UUID, checkinID *uuid.UUID, title *string, fileHeader *multipart.FileHeader) (*SourceDTO, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, ErrProfileNotFound
	}

	// Check file size
	maxBytes := int64(s.maxUploadMB) * 1024 * 1024
	if fileHeader.Size > maxBytes {
		return nil, ErrFileTooLarge
	}

	// Check MIME type
	contentType := fileHeader.Header.Get("Content-Type")
	if !s.isAllowedMime(contentType) {
		return nil, ErrUnsupportedMime
	}

	// Check max sources per checkin (if checkin_id specified)
	if checkinID != nil {
		existingSources, err := s.sourcesStorage.ListSources(ctx, profileID, "", checkinID, 100, 0)
		if err != nil {
			return nil, err
		}
		if len(existingSources) >= s.maxSourcesPerCheck {
			return nil, ErrMaxSourcesExceeded
		}
	}

	// Read file data
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Generate source ID first (for S3 key)
	sourceID := uuid.New()

	// Create source metadata
	source := &storage.Source{
		ID:          sourceID,
		ProfileID:   profileID,
		Kind:        KindImage,
		Title:       title,
		CheckinID:   checkinID,
		ContentType: &contentType,
		SizeBytes:   fileHeader.Size,
	}

	// Store blob first (before creating metadata)
	if s.localMode {
		// Memory mode: store blob in memory after creating metadata
		if err := s.sourcesStorage.CreateSource(ctx, source); err != nil {
			return nil, err
		}
		if err := s.sourcesStorage.PutSourceBlob(ctx, source.ID, data, contentType); err != nil {
			_ = s.sourcesStorage.DeleteSource(ctx, source.ID)
			return nil, fmt.Errorf("failed to store blob: %w", err)
		}
	} else {
		// S3 mode: upload to S3 first
		objectKey := fmt.Sprintf("sources/%s/%s", source.ProfileID.String(), source.ID.String())
		if _, err := s.blobStore.PutObject(ctx, objectKey, data, contentType); err != nil {
			return nil, fmt.Errorf("failed to upload to S3: %w", err)
		}

		// Set object_key and create metadata
		source.ObjectKey = &objectKey
		if err := s.sourcesStorage.CreateSource(ctx, source); err != nil {
			// Rollback: delete from S3
			_ = s.blobStore.DeleteObject(ctx, objectKey)
			return nil, err
		}
	}

	return s.toDTO(source), nil
}

// GetSource retrieves a source by ID
func (s *Service) GetSource(ctx context.Context, id uuid.UUID) (*storage.Source, error) {
	source, err := s.sourcesStorage.GetSource(ctx, id)
	if err != nil {
		return nil, ErrSourceNotFound
	}
	if err := s.ensureProfileAccess(ctx, source.ProfileID); err != nil {
		return nil, ErrSourceNotFound
	}
	return source, nil
}

// ListSources lists sources for a profile with optional filters
func (s *Service) ListSources(ctx context.Context, profileID uuid.UUID, query string, checkinID *uuid.UUID, limit, offset int) ([]SourceDTO, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, ErrProfileNotFound
	}

	sources, err := s.sourcesStorage.ListSources(ctx, profileID, query, checkinID, limit, offset)
	if err != nil {
		return nil, err
	}

	dtos := make([]SourceDTO, len(sources))
	for i, src := range sources {
		dtos[i] = *s.toDTO(&src)
	}

	return dtos, nil
}

// DeleteSource deletes a source and its blob (if S3)
func (s *Service) DeleteSource(ctx context.Context, id uuid.UUID) error {
	source, err := s.sourcesStorage.GetSource(ctx, id)
	if err != nil {
		return ErrSourceNotFound
	}
	if err := s.ensureProfileAccess(ctx, source.ProfileID); err != nil {
		return ErrSourceNotFound
	}

	// Delete blob if S3 mode and object_key exists
	if !s.localMode && source.ObjectKey != nil && *source.ObjectKey != "" {
		if err := s.blobStore.DeleteObject(ctx, *source.ObjectKey); err != nil {
			// Log error but continue with metadata deletion
		}
	}

	return s.sourcesStorage.DeleteSource(ctx, id)
}

// GetImageDownloadURL returns download URL and whether to redirect (true for S3, false for local)
func (s *Service) GetImageDownloadURL(ctx context.Context, id uuid.UUID) (string, bool, error) {
	source, err := s.sourcesStorage.GetSource(ctx, id)
	if err != nil {
		return "", false, ErrSourceNotFound
	}
	if err := s.ensureProfileAccess(ctx, source.ProfileID); err != nil {
		return "", false, ErrSourceNotFound
	}

	if source.Kind != KindImage {
		return "", false, errors.New("source is not an image")
	}

	// Local mode: no redirect, will serve directly
	if s.localMode {
		return "", false, nil
	}

	// S3 mode
	if source.ObjectKey == nil || *source.ObjectKey == "" {
		return "", false, errors.New("object key not found")
	}

	// If prefer public URL mode, construct public URL directly
	if s.preferPublicURL && s.publicBaseURL != "" {
		publicURL := strings.TrimSuffix(s.publicBaseURL, "/") + "/" + *source.ObjectKey
		return publicURL, true, nil
	}

	// Otherwise, generate presigned URL
	presignedURL, err := s.blobStore.PresignGet(ctx, *source.ObjectKey, 900) // 15 min default
	if err != nil {
		return "", false, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL, true, nil
}

// GetImageData retrieves image bytes for download (local mode only)
func (s *Service) GetImageData(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	source, err := s.sourcesStorage.GetSource(ctx, id)
	if err != nil {
		return nil, "", ErrSourceNotFound
	}
	if err := s.ensureProfileAccess(ctx, source.ProfileID); err != nil {
		return nil, "", ErrSourceNotFound
	}

	if source.Kind != KindImage {
		return nil, "", errors.New("source is not an image")
	}

	if s.localMode {
		// Memory mode: get from storage
		return s.sourcesStorage.GetSourceBlob(ctx, id)
	}

	// S3 mode: download from S3
	if source.ObjectKey == nil || *source.ObjectKey == "" {
		return nil, "", errors.New("object key not found")
	}

	data, err := s.blobStore.GetObject(ctx, *source.ObjectKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object from S3: %w", err)
	}

	contentType := "application/octet-stream"
	if source.ContentType != nil {
		contentType = *source.ContentType
	}

	return data, contentType, nil
}

// Helper methods

func (s *Service) toDTO(source *storage.Source) *SourceDTO {
	return &SourceDTO{
		ID:          source.ID,
		ProfileID:   source.ProfileID,
		Kind:        source.Kind,
		Title:       source.Title,
		Text:        source.Text,
		URL:         source.URL,
		CheckinID:   source.CheckinID,
		ContentType: source.ContentType,
		SizeBytes:   source.SizeBytes,
		CreatedAt:   source.CreatedAt,
	}
}

func (s *Service) isAllowedMime(contentType string) bool {
	for _, allowed := range s.allowedMimes {
		if strings.EqualFold(contentType, allowed) {
			return true
		}
	}
	return false
}

func (s *Service) ensureProfileAccess(ctx context.Context, profileID uuid.UUID) error {
	profile, err := s.profileStorage.GetProfile(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}

	if userID, ok := userctx.GetUserID(ctx); ok && strings.TrimSpace(userID) != "" && profile.OwnerUserID != userID {
		return ErrProfileNotFound
	}

	return nil
}
