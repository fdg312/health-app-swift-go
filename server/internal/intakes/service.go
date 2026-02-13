package intakes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/config"
	"github.com/fdg312/health-hub/internal/storage"
	"github.com/fdg312/health-hub/internal/userctx"
	"github.com/google/uuid"
)

type Service struct {
	supplementsStorage storage.SupplementsStorage
	intakesStorage     storage.IntakesStorage
	profileStorage     storage.Storage
	config             *config.Config
}

func NewService(
	supplementsStorage storage.SupplementsStorage,
	intakesStorage storage.IntakesStorage,
	profileStorage storage.Storage,
	cfg *config.Config,
) *Service {
	return &Service{
		supplementsStorage: supplementsStorage,
		intakesStorage:     intakesStorage,
		profileStorage:     profileStorage,
		config:             cfg,
	}
}

// MARK: - Supplements

func (s *Service) CreateSupplement(ctx context.Context, req *CreateSupplementRequest) (*SupplementDTO, error) {
	if err := s.ensureProfileAccess(ctx, req.ProfileID); err != nil {
		return nil, fmt.Errorf("profile_not_found")
	}

	// Check max supplements limit
	existing, err := s.supplementsStorage.ListSupplements(ctx, req.ProfileID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= s.config.IntakesMaxSupplements {
		return nil, fmt.Errorf("max_supplements_reached")
	}

	// Validate name
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	// Create supplement
	supplement := &storage.Supplement{
		ProfileID: req.ProfileID,
		Name:      req.Name,
		Notes:     req.Notes,
	}

	if err := s.supplementsStorage.CreateSupplement(ctx, supplement); err != nil {
		return nil, err
	}

	// Set components if provided
	if len(req.Components) > 0 {
		components := make([]storage.SupplementComponent, len(req.Components))
		for i, c := range req.Components {
			components[i] = storage.SupplementComponent{
				NutrientKey:  c.NutrientKey,
				HKIdentifier: c.HKIdentifier,
				Amount:       c.Amount,
				Unit:         c.Unit,
			}
		}

		if err := s.supplementsStorage.SetSupplementComponents(ctx, supplement.ID, components); err != nil {
			return nil, err
		}
	}

	return s.buildSupplementDTO(ctx, supplement)
}

func (s *Service) ListSupplements(ctx context.Context, profileID uuid.UUID) ([]SupplementDTO, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, fmt.Errorf("profile_not_found")
	}

	supplements, err := s.supplementsStorage.ListSupplements(ctx, profileID)
	if err != nil {
		return nil, err
	}

	dtos := make([]SupplementDTO, len(supplements))
	for i, sup := range supplements {
		dto, err := s.buildSupplementDTO(ctx, &sup)
		if err != nil {
			return nil, err
		}
		dtos[i] = *dto
	}

	return dtos, nil
}

func (s *Service) UpdateSupplement(ctx context.Context, id uuid.UUID, req *UpdateSupplementRequest) (*SupplementDTO, error) {
	supplement, err := s.supplementsStorage.GetSupplement(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("supplement_not_found")
	}
	if err := s.ensureProfileAccess(ctx, supplement.ProfileID); err != nil {
		return nil, fmt.Errorf("supplement_not_found")
	}

	// Update fields if provided
	if req.Name != nil {
		supplement.Name = *req.Name
	}
	if req.Notes != nil {
		supplement.Notes = req.Notes
	}

	if err := s.supplementsStorage.UpdateSupplement(ctx, supplement); err != nil {
		return nil, err
	}

	// Update components if provided
	if req.Components != nil {
		components := make([]storage.SupplementComponent, len(req.Components))
		for i, c := range req.Components {
			components[i] = storage.SupplementComponent{
				NutrientKey:  c.NutrientKey,
				HKIdentifier: c.HKIdentifier,
				Amount:       c.Amount,
				Unit:         c.Unit,
			}
		}

		if err := s.supplementsStorage.SetSupplementComponents(ctx, id, components); err != nil {
			return nil, err
		}
	}

	return s.buildSupplementDTO(ctx, supplement)
}

func (s *Service) DeleteSupplement(ctx context.Context, id uuid.UUID) error {
	supplement, err := s.supplementsStorage.GetSupplement(ctx, id)
	if err != nil {
		return fmt.Errorf("supplement_not_found")
	}
	if err := s.ensureProfileAccess(ctx, supplement.ProfileID); err != nil {
		return fmt.Errorf("supplement_not_found")
	}

	return s.supplementsStorage.DeleteSupplement(ctx, id)
}

// MARK: - Intakes

func (s *Service) AddWater(ctx context.Context, req *AddWaterRequest) error {
	if err := s.ensureProfileAccess(ctx, req.ProfileID); err != nil {
		return fmt.Errorf("profile_not_found")
	}

	// Validate amount
	if req.AmountMl <= 0 {
		return fmt.Errorf("amount_ml must be positive")
	}

	// Check daily limit
	date := req.TakenAt.Format("2006-01-02")
	currentTotal, err := s.intakesStorage.GetWaterDaily(ctx, req.ProfileID, date)
	if err != nil {
		return err
	}

	if currentTotal+req.AmountMl > s.config.IntakesMaxWaterMlPerDay {
		return fmt.Errorf("daily_water_limit_exceeded")
	}

	return s.intakesStorage.AddWater(ctx, req.ProfileID, req.TakenAt, req.AmountMl)
}

func (s *Service) GetIntakesDaily(ctx context.Context, profileID uuid.UUID, date string) (*IntakesDailyResponse, error) {
	if err := s.ensureProfileAccess(ctx, profileID); err != nil {
		return nil, fmt.Errorf("profile_not_found")
	}

	// Get water total and entries
	waterTotal, err := s.intakesStorage.GetWaterDaily(ctx, profileID, date)
	if err != nil {
		return nil, err
	}

	waterEntries, err := s.intakesStorage.ListWaterIntakes(ctx, profileID, date, 20)
	if err != nil {
		return nil, err
	}

	// Get all supplements for profile
	supplements, err := s.supplementsStorage.ListSupplements(ctx, profileID)
	if err != nil {
		return nil, err
	}

	// Get supplement statuses for this date
	statuses, err := s.intakesStorage.GetSupplementDaily(ctx, profileID, date)
	if err != nil {
		return nil, err
	}

	// Build supplement daily statuses
	supplementStatuses := make([]SupplementDailyStatus, len(supplements))
	for i, sup := range supplements {
		status := "none"
		if s, ok := statuses[sup.ID]; ok {
			status = s
		}

		supplementStatuses[i] = SupplementDailyStatus{
			SupplementID: sup.ID,
			Name:         sup.Name,
			Status:       status,
		}
	}

	// Build water entries DTOs
	waterDTOs := make([]WaterIntakeDTO, len(waterEntries))
	for i, entry := range waterEntries {
		waterDTOs[i] = WaterIntakeDTO{
			ID:        entry.ID,
			TakenAt:   entry.TakenAt,
			AmountMl:  entry.AmountMl,
			CreatedAt: entry.CreatedAt,
		}
	}

	return &IntakesDailyResponse{
		Date:         date,
		WaterTotalMl: waterTotal,
		WaterEntries: waterDTOs,
		Supplements:  supplementStatuses,
	}, nil
}

func (s *Service) UpsertSupplementIntake(ctx context.Context, req *UpsertSupplementIntakeRequest) error {
	if err := s.ensureProfileAccess(ctx, req.ProfileID); err != nil {
		return fmt.Errorf("profile_not_found")
	}

	// Validate supplement exists and belongs to profile
	supplement, err := s.supplementsStorage.GetSupplement(ctx, req.SupplementID)
	if err != nil {
		return fmt.Errorf("supplement_not_found")
	}

	if supplement.ProfileID != req.ProfileID {
		return fmt.Errorf("supplement_not_found")
	}

	// Validate status
	if req.Status != "taken" && req.Status != "skipped" {
		return fmt.Errorf("invalid_status")
	}

	// Parse date and create taken_at in UTC
	parsedDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return fmt.Errorf("invalid_date")
	}

	// Use noon UTC for the given date (ensures date comparison works correctly)
	takenAt := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 12, 0, 0, 0, time.UTC)

	intake := &storage.SupplementIntake{
		ProfileID:    req.ProfileID,
		SupplementID: req.SupplementID,
		TakenAt:      takenAt,
		Status:       req.Status,
	}

	return s.intakesStorage.UpsertSupplementIntake(ctx, intake)
}

// MARK: - Helpers

func (s *Service) buildSupplementDTO(ctx context.Context, supplement *storage.Supplement) (*SupplementDTO, error) {
	components, err := s.supplementsStorage.GetSupplementComponents(ctx, supplement.ID)
	if err != nil {
		return nil, err
	}

	componentDTOs := make([]SupplementComponentDTO, len(components))
	for i, c := range components {
		componentDTOs[i] = SupplementComponentDTO{
			ID:           c.ID,
			NutrientKey:  c.NutrientKey,
			HKIdentifier: c.HKIdentifier,
			Amount:       c.Amount,
			Unit:         c.Unit,
		}
	}

	return &SupplementDTO{
		ID:         supplement.ID,
		ProfileID:  supplement.ProfileID,
		Name:       supplement.Name,
		Notes:      supplement.Notes,
		Components: componentDTOs,
		CreatedAt:  supplement.CreatedAt,
		UpdatedAt:  supplement.UpdatedAt,
	}, nil
}

func (s *Service) ensureProfileAccess(ctx context.Context, profileID uuid.UUID) error {
	profile, err := s.profileStorage.GetProfile(ctx, profileID)
	if err != nil {
		return fmt.Errorf("profile_not_found")
	}

	if userID, ok := userctx.GetUserID(ctx); ok && strings.TrimSpace(userID) != "" && profile.OwnerUserID != userID {
		return fmt.Errorf("profile_not_found")
	}

	return nil
}
