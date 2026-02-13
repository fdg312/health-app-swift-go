package checkins

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

var (
	ErrCheckinNotFound = errors.New("checkin not found")
	ErrInvalidType     = errors.New("invalid checkin type")
	ErrInvalidScore    = errors.New("score must be between 1 and 5")
	ErrInvalidDate     = errors.New("invalid date format")
	ErrProfileNotFound = errors.New("profile not found")
)

// Storage defines the interface for checkin storage operations
type Storage interface {
	// ListCheckins returns all check-ins for a profile within a date range
	ListCheckins(profileID uuid.UUID, from, to string) ([]Checkin, error)

	// GetCheckin retrieves a check-in by ID
	GetCheckin(id uuid.UUID) (*Checkin, error)

	// UpsertCheckin creates or updates a check-in (by profile_id, date, type)
	UpsertCheckin(checkin *Checkin) error

	// DeleteCheckin deletes a check-in by ID
	DeleteCheckin(id uuid.UUID) error
}

// ProfileStorage defines the interface for profile operations
type ProfileStorage interface {
	GetProfile(ctx context.Context, id uuid.UUID) (*storage.Profile, error)
}

// Service handles checkin business logic
type Service struct {
	storage        Storage
	profileStorage ProfileStorage
}

// NewService creates a new checkin service
func NewService(storage Storage, profileStorage ProfileStorage) *Service {
	return &Service{
		storage:        storage,
		profileStorage: profileStorage,
	}
}

// ListCheckins returns all check-ins for a profile within a date range
func (s *Service) ListCheckins(ctx context.Context, profileID uuid.UUID, from, to string) ([]CheckinDTO, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, ErrProfileNotFound
	}

	// Validate dates
	if err := validateDate(from); err != nil {
		return nil, fmt.Errorf("invalid from date: %w", err)
	}
	if err := validateDate(to); err != nil {
		return nil, fmt.Errorf("invalid to date: %w", err)
	}

	checkins, err := s.storage.ListCheckins(profileID, from, to)
	if err != nil {
		return nil, err
	}

	dtos := make([]CheckinDTO, len(checkins))
	for i, c := range checkins {
		dtos[i] = c.ToDTO()
	}

	return dtos, nil
}

// UpsertCheckin creates or updates a check-in
func (s *Service) UpsertCheckin(ctx context.Context, req UpsertCheckinRequest) (*CheckinDTO, error) {
	if err := s.ensureProfileAccess(ctx, req.ProfileID); err != nil {
		return nil, ErrProfileNotFound
	}

	// Validate type
	if !isValidType(req.Type) {
		return nil, ErrInvalidType
	}

	// Validate score
	if req.Score < MinScore || req.Score > MaxScore {
		return nil, ErrInvalidScore
	}

	// Validate date
	if err := validateDate(req.Date); err != nil {
		return nil, ErrInvalidDate
	}

	// Create checkin
	now := time.Now().UTC()
	checkin := &Checkin{
		ID:        uuid.New(),
		ProfileID: req.ProfileID,
		Date:      req.Date,
		Type:      req.Type,
		Score:     req.Score,
		Tags:      req.Tags,
		Note:      req.Note,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.storage.UpsertCheckin(checkin); err != nil {
		return nil, err
	}

	dto := checkin.ToDTO()
	return &dto, nil
}

// DeleteCheckin deletes a check-in by ID
func (s *Service) DeleteCheckin(ctx context.Context, id uuid.UUID) error {
	// Check if checkin exists
	checkin, err := s.storage.GetCheckin(id)
	if err != nil {
		return ErrCheckinNotFound
	}
	if err := s.ensureProfileAccess(ctx, checkin.ProfileID); err != nil {
		return ErrCheckinNotFound
	}

	return s.storage.DeleteCheckin(id)
}

// Helper functions

func isValidType(t string) bool {
	for _, valid := range ValidTypes {
		if t == valid {
			return true
		}
	}
	return false
}

func validateDate(date string) error {
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ErrInvalidDate
	}
	return nil
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
