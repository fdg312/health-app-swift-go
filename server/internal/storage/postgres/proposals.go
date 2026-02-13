package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresProposalsStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresProposalsStorage(pool *pgxpool.Pool) *PostgresProposalsStorage {
	return &PostgresProposalsStorage{pool: pool}
}

func (s *PostgresProposalsStorage) InsertMany(ctx context.Context, ownerUserID string, profileID uuid.UUID, drafts []storage.ProposalDraft) ([]storage.AIProposal, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	if len(drafts) == 0 {
		return []storage.AIProposal{}, nil
	}

	const query = `
		INSERT INTO ai_proposals (
			id, owner_user_id, profile_id, created_at, status, kind, title, summary, payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	saved := make([]storage.AIProposal, 0, len(drafts))
	for _, draft := range drafts {
		proposal := storage.AIProposal{
			ID:          uuid.New(),
			OwnerUserID: ownerUserID,
			ProfileID:   profileID,
			CreatedAt:   time.Now().UTC(),
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

		if _, err := tx.Exec(ctx, query,
			proposal.ID,
			proposal.OwnerUserID,
			proposal.ProfileID,
			proposal.CreatedAt,
			proposal.Status,
			proposal.Kind,
			proposal.Title,
			proposal.Summary,
			proposal.Payload,
		); err != nil {
			return nil, err
		}
		saved = append(saved, proposal)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *PostgresProposalsStorage) Get(ctx context.Context, ownerUserID string, proposalID uuid.UUID) (storage.AIProposal, bool, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)

	const query = `
		SELECT id, owner_user_id, profile_id, created_at, status, kind, title, summary, payload
		FROM ai_proposals
		WHERE owner_user_id = $1
		  AND id = $2
	`

	var proposal storage.AIProposal
	err := s.pool.QueryRow(ctx, query, ownerUserID, proposalID).Scan(
		&proposal.ID,
		&proposal.OwnerUserID,
		&proposal.ProfileID,
		&proposal.CreatedAt,
		&proposal.Status,
		&proposal.Kind,
		&proposal.Title,
		&proposal.Summary,
		&proposal.Payload,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.AIProposal{}, false, nil
		}
		return storage.AIProposal{}, false, err
	}
	return proposal, true, nil
}

func (s *PostgresProposalsStorage) UpdateStatus(ctx context.Context, ownerUserID string, proposalID uuid.UUID, status string) error {
	ownerUserID = strings.TrimSpace(ownerUserID)
	status = normalizeProposalStatus(status)

	const query = `
		UPDATE ai_proposals
		SET status = $3
		WHERE owner_user_id = $1
		  AND id = $2
	`

	result, err := s.pool.Exec(ctx, query, ownerUserID, proposalID, status)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresProposalsStorage) List(ctx context.Context, ownerUserID string, profileID uuid.UUID, status string, limit int) ([]storage.AIProposal, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	status = strings.TrimSpace(status)
	if limit <= 0 {
		limit = 50
	}

	const query = `
		SELECT id, owner_user_id, profile_id, created_at, status, kind, title, summary, payload
		FROM ai_proposals
		WHERE owner_user_id = $1
		  AND profile_id = $2
		  AND ($3 = '' OR status = $3)
		ORDER BY created_at DESC, id DESC
		LIMIT $4
	`

	rows, err := s.pool.Query(ctx, query, ownerUserID, profileID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]storage.AIProposal, 0, limit)
	for rows.Next() {
		var proposal storage.AIProposal
		if err := rows.Scan(
			&proposal.ID,
			&proposal.OwnerUserID,
			&proposal.ProfileID,
			&proposal.CreatedAt,
			&proposal.Status,
			&proposal.Kind,
			&proposal.Title,
			&proposal.Summary,
			&proposal.Payload,
		); err != nil {
			return nil, err
		}
		result = append(result, proposal)
	}

	return result, rows.Err()
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
