package schedules

import (
	"context"
	"errors"
	"strings"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrProfileNotFound     = errors.New("profile not found")
	ErrScheduleNotFound    = errors.New("schedule not found")
	ErrSupplementNotFound  = errors.New("supplement not found")
	ErrMaxSchedulesReached = errors.New("max schedules reached")
)

type Service struct {
	schedulesStorage   storage.SupplementSchedulesStorage
	supplementsStorage storage.SupplementsStorage
	profilesStorage    storage.Storage
}

func NewService(
	schedulesStorage storage.SupplementSchedulesStorage,
	supplementsStorage storage.SupplementsStorage,
	profilesStorage storage.Storage,
) *Service {
	return &Service{
		schedulesStorage:   schedulesStorage,
		supplementsStorage: supplementsStorage,
		profilesStorage:    profilesStorage,
	}
}

func (s *Service) List(ctx context.Context, profileID uuid.UUID) (*ListSchedulesResponse, error) {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if profileID == uuid.Nil {
		return nil, ErrInvalidRequest
	}
	if err := s.ensureProfileOwned(ctx, userID, profileID); err != nil {
		return nil, err
	}

	rows, err := s.schedulesStorage.ListSchedules(ctx, userID, profileID)
	if err != nil {
		return nil, err
	}
	result := make([]ScheduleDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDTO(row))
	}
	return &ListSchedulesResponse{Schedules: result}, nil
}

func (s *Service) Upsert(ctx context.Context, req UpsertScheduleRequest) (*ScheduleDTO, error) {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if err := req.Validate(); err != nil {
		return nil, ErrInvalidRequest
	}
	if err := s.ensureProfileOwned(ctx, userID, req.ProfileID); err != nil {
		return nil, err
	}
	if err := s.ensureSupplementInProfile(ctx, req.SupplementID, req.ProfileID); err != nil {
		return nil, err
	}

	existing, err := s.schedulesStorage.ListSchedules(ctx, userID, req.ProfileID)
	if err != nil {
		return nil, err
	}
	alreadyExists := false
	for _, row := range existing {
		if row.SupplementID == req.SupplementID && row.TimeMinutes == req.TimeMinutes {
			alreadyExists = true
			break
		}
	}
	if !alreadyExists && len(existing) >= maxSchedulesPerProfile {
		return nil, ErrMaxSchedulesReached
	}

	row, err := s.schedulesStorage.UpsertSchedule(ctx, userID, req.ProfileID, storage.ScheduleUpsert{
		SupplementID: req.SupplementID,
		TimeMinutes:  req.TimeMinutes,
		DaysMask:     req.DaysMask,
		IsEnabled:    req.IsEnabled,
	})
	if err != nil {
		return nil, err
	}
	dto := toDTO(row)
	return &dto, nil
}

func (s *Service) ReplaceAll(ctx context.Context, req ReplaceSchedulesRequest) (*ListSchedulesResponse, error) {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return nil, ErrUnauthorized
	}
	if err := req.Validate(); err != nil {
		return nil, ErrInvalidRequest
	}
	if err := s.ensureProfileOwned(ctx, userID, req.ProfileID); err != nil {
		return nil, err
	}

	items := make([]storage.ScheduleUpsert, 0, len(req.Schedules))
	for _, item := range req.Schedules {
		if err := s.ensureSupplementInProfile(ctx, item.SupplementID, req.ProfileID); err != nil {
			return nil, err
		}
		items = append(items, storage.ScheduleUpsert{
			SupplementID: item.SupplementID,
			TimeMinutes:  item.TimeMinutes,
			DaysMask:     item.DaysMask,
			IsEnabled:    item.IsEnabled,
		})
	}

	rows, err := s.schedulesStorage.ReplaceAll(ctx, userID, req.ProfileID, items)
	if err != nil {
		return nil, err
	}

	result := make([]ScheduleDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDTO(row))
	}
	return &ListSchedulesResponse{Schedules: result}, nil
}

func (s *Service) Delete(ctx context.Context, scheduleID uuid.UUID) error {
	userID := normalizeOwner(userIDFromContext(ctx))
	if userID == "" {
		return ErrUnauthorized
	}
	if scheduleID == uuid.Nil {
		return ErrInvalidRequest
	}

	if err := s.schedulesStorage.DeleteSchedule(ctx, userID, scheduleID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return ErrScheduleNotFound
		}
		return err
	}
	return nil
}

func (s *Service) ensureProfileOwned(ctx context.Context, ownerUserID string, profileID uuid.UUID) error {
	profile, err := s.profilesStorage.GetProfile(ctx, profileID)
	if err != nil {
		return ErrProfileNotFound
	}
	if profile.OwnerUserID != ownerUserID {
		return ErrProfileNotFound
	}
	return nil
}

func (s *Service) ensureSupplementInProfile(ctx context.Context, supplementID, profileID uuid.UUID) error {
	supplement, err := s.supplementsStorage.GetSupplement(ctx, supplementID)
	if err != nil {
		return ErrSupplementNotFound
	}
	if supplement.ProfileID != profileID {
		return ErrSupplementNotFound
	}
	return nil
}

func userIDFromContext(ctx context.Context) string {
	userID, ok := userctx.GetUserID(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(userID)
}
