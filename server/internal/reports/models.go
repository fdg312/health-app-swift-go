package reports

import (
	"time"

	"github.com/google/uuid"
)

// Report represents a generated report metadata
type Report struct {
	ID        uuid.UUID
	ProfileID uuid.UUID
	Format    string // "pdf" or "csv"
	FromDate  string // YYYY-MM-DD
	ToDate    string // YYYY-MM-DD
	ObjectKey *string
	SizeBytes int64
	Status    string // "ready" or "failed"
	Error     *string
	CreatedAt time.Time
	UpdatedAt time.Time
	Data      []byte // Only used in memory mode
}

// CreateReportRequest is the request to create a new report
type CreateReportRequest struct {
	ProfileID uuid.UUID `json:"profile_id"`
	From      string    `json:"from"` // YYYY-MM-DD
	To        string    `json:"to"`   // YYYY-MM-DD
	Format    string    `json:"format"` // "pdf" or "csv"
}

// ReportDTO is the response representation of a report
type ReportDTO struct {
	ID          uuid.UUID `json:"id"`
	ProfileID   uuid.UUID `json:"profile_id"`
	Format      string    `json:"format"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	DownloadURL string    `json:"download_url"`
	SizeBytes   int64     `json:"size_bytes"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReportsResponse is the list response
type ReportsResponse struct {
	Reports []ReportDTO `json:"reports"`
}

// Constants for validation
const (
	FormatPDF = "pdf"
	FormatCSV = "csv"

	StatusReady  = "ready"
	StatusFailed = "failed"
)
