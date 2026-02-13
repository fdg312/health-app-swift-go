package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
)

type ProposalsMemoryStorage struct {
	mu        sync.RWMutex
	proposals []storage.AIProposal
}

func NewProposalsMemoryStorage() *ProposalsMemoryStorage {
	return &ProposalsMemoryStorage{
		proposals: make([]storage.AIProposal, 0),
	}
}

func (s *ProposalsMemoryStorage) InsertMany(ctx context.Context, ownerUserID string, profileID uuid.UUID, drafts []storage.ProposalDraft) ([]storage.AIProposal, error) {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	ownerUserID = strings.TrimSpace(ownerUserID)
	now := time.Now().UTC()
	saved := make([]storage.AIProposal, 0, len(drafts))
	for _, draft := range drafts {
		proposal := storage.AIProposal{
			ID:          uuid.New(),
			OwnerUserID: ownerUserID,
			ProfileID:   profileID,
			CreatedAt:   now,
			Status:      "pending",
			Kind:        normalizeProposalKind(draft.Kind),
			Title:       strings.TrimSpace(draft.Title),
			Summary:     strings.TrimSpace(draft.Summary),
			Payload:     draft.Payload,
		}
		if proposal.Title == "" {
			proposal.Title = "Предложение"
		}
		if proposal.Summary == "" {
			proposal.Summary = "Ассистент сформировал предложение."
		}

		s.proposals = append(s.proposals, proposal)
		saved = append(saved, proposal)
	}
	return saved, nil
}

func (s *ProposalsMemoryStorage) Get(ctx context.Context, ownerUserID string, proposalID uuid.UUID) (storage.AIProposal, bool, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	ownerUserID = strings.TrimSpace(ownerUserID)
	for _, proposal := range s.proposals {
		if proposal.OwnerUserID == ownerUserID && proposal.ID == proposalID {
			return proposal, true, nil
		}
	}
	return storage.AIProposal{}, false, nil
}

func (s *ProposalsMemoryStorage) UpdateStatus(ctx context.Context, ownerUserID string, proposalID uuid.UUID, status string) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	ownerUserID = strings.TrimSpace(ownerUserID)
	status = normalizeProposalStatus(status)
	for i := range s.proposals {
		if s.proposals[i].OwnerUserID == ownerUserID && s.proposals[i].ID == proposalID {
			s.proposals[i].Status = status
			return nil
		}
	}
	return ErrNotFound
}

func (s *ProposalsMemoryStorage) List(ctx context.Context, ownerUserID string, profileID uuid.UUID, status string, limit int) ([]storage.AIProposal, error) {
	_ = ctx

	ownerUserID = strings.TrimSpace(ownerUserID)
	status = strings.TrimSpace(status)
	if limit <= 0 {
		limit = 50
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]storage.AIProposal, 0, len(s.proposals))
	for _, proposal := range s.proposals {
		if proposal.OwnerUserID != ownerUserID || proposal.ProfileID != profileID {
			continue
		}
		if status != "" && proposal.Status != status {
			continue
		}
		filtered = append(filtered, proposal)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID.String() < filtered[j].ID.String()
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	if len(filtered) <= limit {
		return filtered, nil
	}

	return filtered[:limit], nil
}

func normalizeProposalKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case "settings_update", "vitamins_schedule", "workout_plan", "nutrition_plan", "generic":
		return strings.TrimSpace(kind)
	default:
		return "generic"
	}
}

func normalizeProposalStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "pending", "applied", "rejected":
		return strings.TrimSpace(status)
	default:
		return "pending"
	}
}
