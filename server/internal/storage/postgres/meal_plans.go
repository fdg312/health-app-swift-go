package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type mealPlansStorage struct {
	pool *pgxpool.Pool
}

func newMealPlansStorage(pool *pgxpool.Pool) *mealPlansStorage {
	return &mealPlansStorage{pool: pool}
}

func (s *mealPlansStorage) GetActive(ctx context.Context, ownerUserID string, profileID string) (storage.MealPlan, []storage.MealPlanItem, bool, error) {
	// Get active plan
	planQuery := `
		SELECT id, owner_user_id, profile_id, title, is_active, from_date, created_at, updated_at
		FROM meal_plans
		WHERE owner_user_id = $1 AND profile_id = $2 AND is_active = true
	`

	var plan storage.MealPlan
	err := s.pool.QueryRow(ctx, planQuery, ownerUserID, profileID).Scan(
		&plan.ID,
		&plan.OwnerUserID,
		&plan.ProfileID,
		&plan.Title,
		&plan.IsActive,
		&plan.FromDate,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return storage.MealPlan{}, nil, false, nil
	}
	if err != nil {
		return storage.MealPlan{}, nil, false, fmt.Errorf("failed to get active meal plan: %w", err)
	}

	// Get items for this plan
	itemsQuery := `
		SELECT id, owner_user_id, profile_id, plan_id, day_index, meal_slot, title, notes,
		       approx_kcal, approx_protein_g, approx_fat_g, approx_carbs_g, created_at, updated_at
		FROM meal_plan_items
		WHERE owner_user_id = $1 AND profile_id = $2 AND plan_id = $3
		ORDER BY day_index, meal_slot
	`

	rows, err := s.pool.Query(ctx, itemsQuery, ownerUserID, profileID, plan.ID)
	if err != nil {
		return storage.MealPlan{}, nil, false, fmt.Errorf("failed to get meal plan items: %w", err)
	}
	defer rows.Close()

	var items []storage.MealPlanItem
	for rows.Next() {
		var item storage.MealPlanItem
		err := rows.Scan(
			&item.ID,
			&item.OwnerUserID,
			&item.ProfileID,
			&item.PlanID,
			&item.DayIndex,
			&item.MealSlot,
			&item.Title,
			&item.Notes,
			&item.ApproxKcal,
			&item.ApproxProteinG,
			&item.ApproxFatG,
			&item.ApproxCarbsG,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return storage.MealPlan{}, nil, false, fmt.Errorf("failed to scan meal plan item: %w", err)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return storage.MealPlan{}, nil, false, fmt.Errorf("error iterating meal plan items: %w", rows.Err())
	}

	return plan, items, true, nil
}

func (s *mealPlansStorage) ReplaceActive(ctx context.Context, ownerUserID string, profileID string, title string, itemsUpsert []storage.MealPlanItemUpsert) (storage.MealPlan, []storage.MealPlanItem, error) {
	// Start transaction
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return storage.MealPlan{}, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing active plan and its items (CASCADE will handle items)
	deleteQuery := `
		DELETE FROM meal_plans
		WHERE owner_user_id = $1 AND profile_id = $2 AND is_active = true
	`
	_, err = tx.Exec(ctx, deleteQuery, ownerUserID, profileID)
	if err != nil {
		return storage.MealPlan{}, nil, fmt.Errorf("failed to delete existing meal plan: %w", err)
	}

	// Create new plan
	planQuery := `
		INSERT INTO meal_plans (owner_user_id, profile_id, title, is_active, from_date)
		VALUES ($1, $2, $3, true, NULL)
		RETURNING id, owner_user_id, profile_id, title, is_active, from_date, created_at, updated_at
	`

	var plan storage.MealPlan
	err = tx.QueryRow(ctx, planQuery, ownerUserID, profileID, title).Scan(
		&plan.ID,
		&plan.OwnerUserID,
		&plan.ProfileID,
		&plan.Title,
		&plan.IsActive,
		&plan.FromDate,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		return storage.MealPlan{}, nil, fmt.Errorf("failed to create meal plan: %w", err)
	}

	// Insert items
	var items []storage.MealPlanItem
	itemQuery := `
		INSERT INTO meal_plan_items (owner_user_id, profile_id, plan_id, day_index, meal_slot, title, notes,
		                             approx_kcal, approx_protein_g, approx_fat_g, approx_carbs_g)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, owner_user_id, profile_id, plan_id, day_index, meal_slot, title, notes,
		          approx_kcal, approx_protein_g, approx_fat_g, approx_carbs_g, created_at, updated_at
	`

	for _, itemReq := range itemsUpsert {
		var item storage.MealPlanItem
		err = tx.QueryRow(
			ctx,
			itemQuery,
			ownerUserID,
			profileID,
			plan.ID,
			itemReq.DayIndex,
			itemReq.MealSlot,
			itemReq.Title,
			itemReq.Notes,
			itemReq.ApproxKcal,
			itemReq.ApproxProteinG,
			itemReq.ApproxFatG,
			itemReq.ApproxCarbsG,
		).Scan(
			&item.ID,
			&item.OwnerUserID,
			&item.ProfileID,
			&item.PlanID,
			&item.DayIndex,
			&item.MealSlot,
			&item.Title,
			&item.Notes,
			&item.ApproxKcal,
			&item.ApproxProteinG,
			&item.ApproxFatG,
			&item.ApproxCarbsG,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return storage.MealPlan{}, nil, fmt.Errorf("failed to insert meal plan item: %w", err)
		}
		items = append(items, item)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return storage.MealPlan{}, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return plan, items, nil
}

func (s *mealPlansStorage) DeleteActive(ctx context.Context, ownerUserID string, profileID string) error {
	query := `
		DELETE FROM meal_plans
		WHERE owner_user_id = $1 AND profile_id = $2 AND is_active = true
	`

	result, err := s.pool.Exec(ctx, query, ownerUserID, profileID)
	if err != nil {
		return fmt.Errorf("failed to delete active meal plan: %w", err)
	}

	// No error if nothing was deleted (no active plan)
	_ = result

	return nil
}

func (s *mealPlansStorage) GetToday(ctx context.Context, ownerUserID string, profileID string, date time.Time) ([]storage.MealPlanItem, error) {
	// Calculate day_index from date (0=Monday, 6=Sunday)
	// In Go, Sunday=0, Monday=1, ..., Saturday=6
	weekday := date.Weekday()
	dayIndex := int(weekday) - 1
	if dayIndex < 0 {
		dayIndex = 6 // Sunday becomes 6
	}

	query := `
		SELECT i.id, i.owner_user_id, i.profile_id, i.plan_id, i.day_index, i.meal_slot, i.title, i.notes,
		       i.approx_kcal, i.approx_protein_g, i.approx_fat_g, i.approx_carbs_g, i.created_at, i.updated_at
		FROM meal_plan_items i
		INNER JOIN meal_plans p ON p.id = i.plan_id
		WHERE i.owner_user_id = $1 AND i.profile_id = $2 AND p.is_active = true AND i.day_index = $3
		ORDER BY
			CASE i.meal_slot
				WHEN 'breakfast' THEN 1
				WHEN 'lunch' THEN 2
				WHEN 'dinner' THEN 3
				WHEN 'snack' THEN 4
			END
	`

	rows, err := s.pool.Query(ctx, query, ownerUserID, profileID, dayIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get today's meal plan: %w", err)
	}
	defer rows.Close()

	var items []storage.MealPlanItem
	for rows.Next() {
		var item storage.MealPlanItem
		err := rows.Scan(
			&item.ID,
			&item.OwnerUserID,
			&item.ProfileID,
			&item.PlanID,
			&item.DayIndex,
			&item.MealSlot,
			&item.Title,
			&item.Notes,
			&item.ApproxKcal,
			&item.ApproxProteinG,
			&item.ApproxFatG,
			&item.ApproxCarbsG,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan meal plan item: %w", err)
		}
		items = append(items, item)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("error iterating meal plan items: %w", rows.Err())
	}

	return items, nil
}
