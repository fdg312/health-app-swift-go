package profiles

import (
	"context"
	"errors"
	"strings"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

var (
	ErrInvalidType       = errors.New("invalid profile type")
	ErrEmptyName         = errors.New("name cannot be empty")
	ErrCannotDeleteOwner = errors.New("cannot delete owner profile")
	ErrNotFound          = errors.New("profile not found")
)

// Service содержит бизнес-логику профилей
type Service struct {
	storage storage.Storage
}

// NewService создаёт новый сервис
func NewService(st storage.Storage) *Service {
	return &Service{storage: st}
}

// ListProfiles возвращает все профили
func (s *Service) ListProfiles(ctx context.Context) ([]ProfileDTO, error) {
	userID := userIDFromContext(ctx)

	if err := s.ensureOwnerProfile(ctx, userID); err != nil {
		return nil, err
	}

	profiles, err := s.storage.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]ProfileDTO, 0, len(profiles))
	for _, p := range profiles {
		if p.OwnerUserID != userID {
			continue
		}
		dtos = append(dtos, toDTO(p))
	}

	return dtos, nil
}

// GetProfile возвращает профиль по ID
func (s *Service) GetProfile(ctx context.Context, id uuid.UUID) (*ProfileDTO, error) {
	userID := userIDFromContext(ctx)

	profile, err := s.storage.GetProfile(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if profile.OwnerUserID != userID {
		return nil, ErrNotFound
	}

	dto := toDTO(*profile)
	return &dto, nil
}

// CreateProfile создаёт новый профиль (только guest)
func (s *Service) CreateProfile(ctx context.Context, req CreateProfileRequest) (*ProfileDTO, error) {
	userID := userIDFromContext(ctx)

	// Валидация
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrEmptyName
	}

	if req.Type != "guest" {
		return nil, ErrInvalidType
	}

	profile := &storage.Profile{
		OwnerUserID: userID,
		Type:        req.Type,
		Name:        strings.TrimSpace(req.Name),
	}

	if err := s.storage.CreateProfile(ctx, profile); err != nil {
		return nil, err
	}

	dto := toDTO(*profile)
	return &dto, nil
}

// UpdateProfile обновляет имя профиля
func (s *Service) UpdateProfile(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*ProfileDTO, error) {
	userID := userIDFromContext(ctx)

	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrEmptyName
	}

	profile, err := s.storage.GetProfile(ctx, id)
	if err != nil {
		return nil, ErrNotFound
	}
	if profile.OwnerUserID != userID {
		return nil, ErrNotFound
	}

	profile.Name = strings.TrimSpace(req.Name)

	if err := s.storage.UpdateProfile(ctx, profile); err != nil {
		return nil, err
	}

	dto := toDTO(*profile)
	return &dto, nil
}

// DeleteProfile удаляет профиль (только guest)
func (s *Service) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	userID := userIDFromContext(ctx)

	profile, err := s.storage.GetProfile(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	if profile.OwnerUserID != userID {
		return ErrNotFound
	}

	if profile.Type == "owner" {
		return ErrCannotDeleteOwner
	}

	return s.storage.DeleteProfile(ctx, id)
}

// toDTO конвертирует storage.Profile в ProfileDTO
func toDTO(p storage.Profile) ProfileDTO {
	return ProfileDTO{
		ID:          p.ID,
		OwnerUserID: p.OwnerUserID,
		Type:        p.Type,
		Name:        p.Name,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func userIDFromContext(ctx context.Context) string {
	if userID, ok := userctx.GetUserID(ctx); ok && strings.TrimSpace(userID) != "" {
		return userID
	}
	return "default"
}

func (s *Service) ensureOwnerProfile(ctx context.Context, userID string) error {
	profiles, err := s.storage.ListProfiles(ctx)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if p.OwnerUserID == userID && p.Type == "owner" {
			return nil
		}
	}
	profile := &storage.Profile{
		OwnerUserID: userID,
		Type:        "owner",
		Name:        "Я",
	}
	return s.storage.CreateProfile(ctx, profile)
}
