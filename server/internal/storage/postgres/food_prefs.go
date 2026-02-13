package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type foodPrefsStorage struct {
	pool *pgxpool.Pool
}

func newFoodPrefsStorage(pool *pgxpool.Pool) *foodPrefsStorage {
	return &foodPrefsStorage{pool: pool}
}

func (s *foodPrefsStorage) List(ctx context.Context, ownerUserID string, profileID string, query string, limit, offset int) ([]storage.FoodPref, int, error) {
	// Build query with optional search filter
	var args []interface{}
	whereClause := "WHERE owner_user_id = $1 AND profile_id = $2"
	args = append(args, ownerUserID, profileID)

	if query != "" {
		// Search in name and tags (case-insensitive)
		whereClause += " AND (LOWER(name) LIKE $3 OR EXISTS (SELECT 1 FROM unnest(tags) AS tag WHERE LOWER(tag) LIKE $3))"
		args = append(args, "%"+strings.ToLower(query)+"%")
	}

	// Count total matching records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM food_preferences %s", whereClause)
	var total int
	err := s.pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count food preferences: %w", err)
	}

	// Fetch paginated results
	listQuery := fmt.Sprintf(`
		SELECT id, owner_user_id, profile_id, name, tags,
		       kcal_per_100g, protein_g_per_100g, fat_g_per_100g, carbs_g_per_100g,
		       created_at, updated_at
		FROM food_preferences
		%s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)

	args = append(args, limit, offset)

	rows, err := s.pool.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list food preferences: %w", err)
	}
	defer rows.Close()

	var prefs []storage.FoodPref
	for rows.Next() {
		var pref storage.FoodPref
		err := rows.Scan(
			&pref.ID,
			&pref.OwnerUserID,
			&pref.ProfileID,
			&pref.Name,
			&pref.Tags,
			&pref.KcalPer100g,
			&pref.ProteinGPer100g,
			&pref.FatGPer100g,
			&pref.CarbsGPer100g,
			&pref.CreatedAt,
			&pref.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan food preference: %w", err)
		}
		prefs = append(prefs, pref)
	}

	if rows.Err() != nil {
		return nil, 0, fmt.Errorf("error iterating food preferences: %w", rows.Err())
	}

	return prefs, total, nil
}

func (s *foodPrefsStorage) Upsert(ctx context.Context, ownerUserID string, profileID string, req storage.FoodPrefUpsert) (storage.FoodPref, error) {
	if req.ID != "" {
		// Update existing
		query := `
			UPDATE food_preferences
			SET name = $1, tags = $2, kcal_per_100g = $3, protein_g_per_100g = $4,
			    fat_g_per_100g = $5, carbs_g_per_100g = $6, updated_at = now()
			WHERE id = $7 AND owner_user_id = $8
			RETURNING id, owner_user_id, profile_id, name, tags,
			          kcal_per_100g, protein_g_per_100g, fat_g_per_100g, carbs_g_per_100g,
			          created_at, updated_at
		`

		var pref storage.FoodPref
		err := s.pool.QueryRow(
			ctx,
			query,
			req.Name,
			req.Tags,
			req.KcalPer100g,
			req.ProteinGPer100g,
			req.FatGPer100g,
			req.CarbsGPer100g,
			req.ID,
			ownerUserID,
		).Scan(
			&pref.ID,
			&pref.OwnerUserID,
			&pref.ProfileID,
			&pref.Name,
			&pref.Tags,
			&pref.KcalPer100g,
			&pref.ProteinGPer100g,
			&pref.FatGPer100g,
			&pref.CarbsGPer100g,
			&pref.CreatedAt,
			&pref.UpdatedAt,
		)

		if err == pgx.ErrNoRows {
			return storage.FoodPref{}, fmt.Errorf("food preference not found or unauthorized")
		}
		if err != nil {
			return storage.FoodPref{}, fmt.Errorf("failed to update food preference: %w", err)
		}

		return pref, nil
	}

	// Insert new
	query := `
		INSERT INTO food_preferences (owner_user_id, profile_id, name, tags,
		                              kcal_per_100g, protein_g_per_100g, fat_g_per_100g, carbs_g_per_100g)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, owner_user_id, profile_id, name, tags,
		          kcal_per_100g, protein_g_per_100g, fat_g_per_100g, carbs_g_per_100g,
		          created_at, updated_at
	`

	var pref storage.FoodPref
	err := s.pool.QueryRow(
		ctx,
		query,
		ownerUserID,
		profileID,
		req.Name,
		req.Tags,
		req.KcalPer100g,
		req.ProteinGPer100g,
		req.FatGPer100g,
		req.CarbsGPer100g,
	).Scan(
		&pref.ID,
		&pref.OwnerUserID,
		&pref.ProfileID,
		&pref.Name,
		&pref.Tags,
		&pref.KcalPer100g,
		&pref.ProteinGPer100g,
		&pref.FatGPer100g,
		&pref.CarbsGPer100g,
		&pref.CreatedAt,
		&pref.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (duplicate name)
		if strings.Contains(err.Error(), "food_preferences_name_unique_idx") {
			return storage.FoodPref{}, fmt.Errorf("food preference with this name already exists")
		}
		return storage.FoodPref{}, fmt.Errorf("failed to create food preference: %w", err)
	}

	return pref, nil
}

func (s *foodPrefsStorage) Delete(ctx context.Context, ownerUserID string, id string) error {
	query := `
		DELETE FROM food_preferences
		WHERE id = $1 AND owner_user_id = $2
	`

	result, err := s.pool.Exec(ctx, query, id, ownerUserID)
	if err != nil {
		return fmt.Errorf("failed to delete food preference: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("food preference not found or unauthorized")
	}

	return nil
}
