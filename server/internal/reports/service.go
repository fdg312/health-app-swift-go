package reports

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/blob"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

// Service handles reports business logic
type Service struct {
	reportsStorage  storage.ReportsStorage
	metricsStorage  storage.MetricsStorage
	checkinsStorage CheckinsStorageAdapter
	profileStorage  ProfileStorageAdapter
	generator       *Generator
	blobStore       blob.Store
	maxRangeDays    int
	presignTTL      int
	localMode       bool   // true if no S3 configured
	publicBaseURL   string // S3 public base URL (if prefer_public_url mode)
	preferPublicURL bool   // if true, use public URLs instead of presigned
}

// NewService creates a new reports service
func NewService(
	reportsStorage storage.ReportsStorage,
	metricsStorage storage.MetricsStorage,
	checkinsStorage CheckinsStorageAdapter,
	profileStorage ProfileStorageAdapter,
	blobStore blob.Store,
	maxRangeDays int,
	presignTTL int,
	publicBaseURL string,
	preferPublicURL bool,
) *Service {
	generator := NewGenerator(metricsStorage, checkinsStorage, profileStorage)

	localMode := (blobStore == nil)

	return &Service{
		reportsStorage:  reportsStorage,
		metricsStorage:  metricsStorage,
		checkinsStorage: checkinsStorage,
		profileStorage:  profileStorage,
		generator:       generator,
		blobStore:       blobStore,
		maxRangeDays:    maxRangeDays,
		presignTTL:      presignTTL,
		localMode:       localMode,
		publicBaseURL:   publicBaseURL,
		preferPublicURL: preferPublicURL,
	}
}

// CreateReport creates a new report
func (s *Service) CreateReport(ctx context.Context, req CreateReportRequest) (*Report, error) {
	// Validate format
	if req.Format != FormatPDF && req.Format != FormatCSV {
		return nil, ErrInvalidFormat
	}

	// Validate dates
	fromDate, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		return nil, ErrInvalidDate
	}

	toDate, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		return nil, ErrInvalidDate
	}

	if fromDate.After(toDate) {
		return nil, ErrInvalidDateRange
	}

	// Check max range
	daysDiff := int(toDate.Sub(fromDate).Hours() / 24)
	if daysDiff > s.maxRangeDays {
		return nil, ErrRangeTooLarge
	}

	if err = s.ensureProfileAccess(ctx, req.ProfileID); err != nil {
		return nil, ErrProfileNotFound
	}

	// Generate report
	data, err := s.generator.GenerateReport(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}

	// Create report metadata
	report := &storage.ReportMeta{
		ProfileID: req.ProfileID,
		Format:    req.Format,
		FromDate:  req.From,
		ToDate:    req.To,
		SizeBytes: int64(len(data)),
		Status:    StatusReady,
	}

	// Upload to S3 or store locally
	if s.localMode {
		// Local mode: store data in memory
		report.Data = data
	} else {
		// S3 mode: upload to object storage
		objectKey := fmt.Sprintf("reports/%s/%s_%s_%s.%s",
			req.ProfileID.String(),
			req.From,
			req.To,
			uuid.New().String(),
			req.Format,
		)

		contentType := "application/pdf"
		if req.Format == FormatCSV {
			contentType = "text/csv"
		}

		_, err = s.blobStore.PutObject(ctx, objectKey, data, contentType)
		if err != nil {
			return nil, fmt.Errorf("failed to upload to S3: %w", err)
		}

		report.ObjectKey = &objectKey
	}

	// Save metadata
	if err := s.reportsStorage.CreateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to save report metadata: %w", err)
	}

	// Convert to Report model
	return s.toReport(report), nil
}

// GetReport retrieves a report by ID
func (s *Service) GetReport(ctx context.Context, id uuid.UUID) (*Report, error) {
	meta, err := s.reportsStorage.GetReport(ctx, id)
	if err != nil {
		return nil, ErrReportNotFound
	}
	if err := s.ensureProfileAccess(ctx, meta.ProfileID); err != nil {
		return nil, ErrReportNotFound
	}

	return s.toReport(meta), nil
}

// ListReports lists reports for a profile
func (s *Service) ListReports(ctx context.Context, profileID uuid.UUID, limit, offset int) ([]Report, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, ErrProfileNotFound
	}

	metaList, err := s.reportsStorage.ListReports(ctx, profileID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}

	reports := make([]Report, len(metaList))
	for i, meta := range metaList {
		reports[i] = *s.toReport(&meta)
	}

	return reports, nil
}

// DeleteReport deletes a report
func (s *Service) DeleteReport(ctx context.Context, id uuid.UUID) error {
	// Get report metadata
	meta, err := s.reportsStorage.GetReport(ctx, id)
	if err != nil {
		return ErrReportNotFound
	}
	if err := s.ensureProfileAccess(ctx, meta.ProfileID); err != nil {
		return ErrReportNotFound
	}

	// Delete from S3 if applicable
	if !s.localMode && meta.ObjectKey != nil {
		if err := s.blobStore.DeleteObject(ctx, *meta.ObjectKey); err != nil {
			// Log but don't fail - metadata deletion is more important
			fmt.Printf("warning: failed to delete S3 object: %v\n", err)
		}
	}

	// Delete metadata
	if err := s.reportsStorage.DeleteReport(ctx, id); err != nil {
		return fmt.Errorf("failed to delete report metadata: %w", err)
	}

	return nil
}

// GetReportDownloadURL generates a download URL for a report
func (s *Service) GetReportDownloadURL(ctx context.Context, id uuid.UUID, baseURL string) (string, error) {
	meta, err := s.reportsStorage.GetReport(ctx, id)
	if err != nil {
		return "", ErrReportNotFound
	}
	if err := s.ensureProfileAccess(ctx, meta.ProfileID); err != nil {
		return "", ErrReportNotFound
	}

	if s.localMode {
		// Local mode: return direct download endpoint
		return fmt.Sprintf("%s/v1/reports/%s/download", strings.TrimSuffix(baseURL, "/"), id.String()), nil
	}

	// S3 mode
	if meta.ObjectKey == nil {
		return "", fmt.Errorf("object key is missing")
	}

	// If prefer public URL mode, construct public URL directly
	if s.preferPublicURL && s.publicBaseURL != "" {
		publicURL := strings.TrimSuffix(s.publicBaseURL, "/") + "/" + *meta.ObjectKey
		return publicURL, nil
	}

	// Otherwise, generate presigned URL
	presignedURL, err := s.blobStore.PresignGet(ctx, *meta.ObjectKey, s.presignTTL)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL, nil
}

// GetReportData retrieves the raw report data (for local mode download)
func (s *Service) GetReportData(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	meta, err := s.reportsStorage.GetReport(ctx, id)
	if err != nil {
		return nil, "", ErrReportNotFound
	}
	if err := s.ensureProfileAccess(ctx, meta.ProfileID); err != nil {
		return nil, "", ErrReportNotFound
	}

	contentType := "application/pdf"
	if meta.Format == FormatCSV {
		contentType = "text/csv"
	}

	if s.localMode {
		// Return data from memory
		return meta.Data, contentType, nil
	}

	// S3 mode: fetch from object storage
	if meta.ObjectKey == nil {
		return nil, "", fmt.Errorf("object key is missing")
	}

	// Get from S3 (requires implementing GetObject in blob.Store)
	// For now, we'll just redirect to presigned URL in handler
	return nil, contentType, fmt.Errorf("S3 mode should use presigned URL redirect")
}

// toReport converts ReportMeta to Report model
func (s *Service) toReport(meta *storage.ReportMeta) *Report {
	return &Report{
		ID:        meta.ID,
		ProfileID: meta.ProfileID,
		Format:    meta.Format,
		FromDate:  meta.FromDate,
		ToDate:    meta.ToDate,
		ObjectKey: meta.ObjectKey,
		SizeBytes: meta.SizeBytes,
		Status:    meta.Status,
		Error:     meta.Error,
		CreatedAt: meta.CreatedAt,
		UpdatedAt: meta.UpdatedAt,
		Data:      meta.Data,
	}
}

// Errors
var (
	ErrInvalidFormat    = fmt.Errorf("invalid format")
	ErrInvalidDate      = fmt.Errorf("invalid date format")
	ErrInvalidDateRange = fmt.Errorf("from date must be before to date")
	ErrRangeTooLarge    = fmt.Errorf("date range too large")
	ErrProfileNotFound  = fmt.Errorf("profile not found")
	ErrReportNotFound   = fmt.Errorf("report not found")
)

// Adapter interfaces
type CheckinsStorageAdapter interface {
	ListCheckins(ctx context.Context, profileID uuid.UUID, from, to string) ([]Checkin, error)
}

type ProfileStorageAdapter interface {
	GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error)
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
