package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

type foodPrefsStorage struct {
	mu    sync.RWMutex
	prefs map[string]*storage.FoodPref // key: id
	// index for owner+profile lookups
	byOwnerProfile map[string][]string // key: "ownerUserID:profileID" -> []id
}

func newFoodPrefsStorage() *foodPrefsStorage {
	return &foodPrefsStorage{
		prefs:          make(map[string]*storage.FoodPref),
		byOwnerProfile: make(map[string][]string),
	}
}

func (s *foodPrefsStorage) List(ctx context.Context, ownerUserID string, profileID string, query string, limit, offset int) ([]storage.FoodPref, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", ownerUserID, profileID)
	ids, ok := s.byOwnerProfile[key]
	if !ok {
		return []storage.FoodPref{}, 0, nil
	}

	// Collect matching prefs
	var results []storage.FoodPref
	queryLower := strings.ToLower(query)

	for _, id := range ids {
		pref, exists := s.prefs[id]
		if !exists {
			continue
		}

		// Filter by query if provided
		if query != "" {
			nameLower := strings.ToLower(pref.Name)
			match := strings.Contains(nameLower, queryLower)

			// Also check tags
			if !match {
				for _, tag := range pref.Tags {
					if strings.Contains(strings.ToLower(tag), queryLower) {
						match = true
						break
					}
				}
			}

			if !match {
				continue
			}
		}

		results = append(results, *pref)
	}

	total := len(results)

	// Apply pagination
	if offset >= len(results) {
		return []storage.FoodPref{}, total, nil
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end], total, nil
}

func (s *foodPrefsStorage) Upsert(ctx context.Context, ownerUserID string, profileID string, req storage.FoodPrefUpsert) (storage.FoodPref, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	key := fmt.Sprintf("%s:%s", ownerUserID, profileID)

	if req.ID != "" {
		// Update existing
		existing, ok := s.prefs[req.ID]
		if !ok {
			return storage.FoodPref{}, fmt.Errorf("food preference not found")
		}

		if existing.OwnerUserID != ownerUserID {
			return storage.FoodPref{}, fmt.Errorf("unauthorized")
		}

		existing.Name = req.Name
		existing.Tags = req.Tags
		existing.KcalPer100g = req.KcalPer100g
		existing.ProteinGPer100g = req.ProteinGPer100g
		existing.FatGPer100g = req.FatGPer100g
		existing.CarbsGPer100g = req.CarbsGPer100g
		existing.UpdatedAt = now

		return *existing, nil
	}

	// Check for duplicate name (case-insensitive)
	reqNameLower := strings.ToLower(req.Name)
	ids, _ := s.byOwnerProfile[key]
	for _, id := range ids {
		if pref, ok := s.prefs[id]; ok {
			if strings.ToLower(pref.Name) == reqNameLower {
				return storage.FoodPref{}, fmt.Errorf("food preference with this name already exists")
			}
		}
	}

	// Create new
	newID := uuid.New().String()
	pref := &storage.FoodPref{
		ID:              newID,
		OwnerUserID:     ownerUserID,
		ProfileID:       profileID,
		Name:            req.Name,
		Tags:            req.Tags,
		KcalPer100g:     req.KcalPer100g,
		ProteinGPer100g: req.ProteinGPer100g,
		FatGPer100g:     req.FatGPer100g,
		CarbsGPer100g:   req.CarbsGPer100g,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if pref.Tags == nil {
		pref.Tags = []string{}
	}

	s.prefs[newID] = pref
	s.byOwnerProfile[key] = append(s.byOwnerProfile[key], newID)

	return *pref, nil
}

func (s *foodPrefsStorage) Delete(ctx context.Context, ownerUserID string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pref, ok := s.prefs[id]
	if !ok {
		return fmt.Errorf("food preference not found")
	}

	if pref.OwnerUserID != ownerUserID {
		return fmt.Errorf("unauthorized")
	}

	// Remove from index
	key := fmt.Sprintf("%s:%s", ownerUserID, pref.ProfileID)
	ids := s.byOwnerProfile[key]
	for i, existingID := range ids {
		if existingID == id {
			s.byOwnerProfile[key] = append(ids[:i], ids[i+1:]...)
			break
		}
	}

	delete(s.prefs, id)
	return nil
}
