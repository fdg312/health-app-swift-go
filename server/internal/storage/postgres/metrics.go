package postgres

import (
	"context"
	"time"

	"github.com/fdg312/health-hub/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresMetricsStorage — Postgres реализация MetricsStorage
type PostgresMetricsStorage struct {
	pool *pgxpool.Pool
}

// NewMetricsStorage создаёт PostgresMetricsStorage
func NewMetricsStorage(pool *pgxpool.Pool) *PostgresMetricsStorage {
	return &PostgresMetricsStorage{pool: pool}
}

func (p *PostgresMetricsStorage) UpsertDailyMetric(ctx context.Context, profileID uuid.UUID, date string, payload []byte) error {
	query := `
		INSERT INTO daily_metrics (profile_id, date, payload, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (profile_id, date)
		DO UPDATE SET payload = EXCLUDED.payload, updated_at = NOW()
	`

	_, err := p.pool.Exec(ctx, query, profileID, date, payload)
	return err
}

func (p *PostgresMetricsStorage) GetDailyMetrics(ctx context.Context, profileID uuid.UUID, from, to string) ([]storage.DailyMetricRow, error) {
	query := `
		SELECT profile_id, date, payload, created_at, updated_at
		FROM daily_metrics
		WHERE profile_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC
	`

	rows, err := p.pool.Query(ctx, query, profileID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []storage.DailyMetricRow
	for rows.Next() {
		var row storage.DailyMetricRow
		err := rows.Scan(&row.ProfileID, &row.Date, &row.Payload, &row.CreatedAt, &row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

func (p *PostgresMetricsStorage) UpsertHourlyMetric(ctx context.Context, profileID uuid.UUID, hour time.Time, steps *int, hrMin, hrMax, hrAvg *int) error {
	query := `
		INSERT INTO hourly_metrics (profile_id, hour, steps, hr_min, hr_max, hr_avg, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (profile_id, hour)
		DO UPDATE SET
			steps = COALESCE(EXCLUDED.steps, hourly_metrics.steps),
			hr_min = COALESCE(EXCLUDED.hr_min, hourly_metrics.hr_min),
			hr_max = COALESCE(EXCLUDED.hr_max, hourly_metrics.hr_max),
			hr_avg = COALESCE(EXCLUDED.hr_avg, hourly_metrics.hr_avg),
			updated_at = NOW()
	`

	_, err := p.pool.Exec(ctx, query, profileID, hour.Truncate(time.Hour), steps, hrMin, hrMax, hrAvg)
	return err
}

func (p *PostgresMetricsStorage) GetHourlyMetrics(ctx context.Context, profileID uuid.UUID, date string) ([]storage.HourlyMetricRow, error) {
	query := `
		SELECT profile_id, hour, steps, hr_min, hr_max, hr_avg, created_at, updated_at
		FROM hourly_metrics
		WHERE profile_id = $1 AND date(hour) = $2
		ORDER BY hour ASC
	`

	rows, err := p.pool.Query(ctx, query, profileID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []storage.HourlyMetricRow
	for rows.Next() {
		var row storage.HourlyMetricRow
		err := rows.Scan(&row.ProfileID, &row.Hour, &row.Steps, &row.HRMin, &row.HRMax, &row.HRAvg, &row.CreatedAt, &row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

func (p *PostgresMetricsStorage) InsertSleepSegment(ctx context.Context, profileID uuid.UUID, start, end time.Time, stage string) error {
	query := `
		INSERT INTO sleep_segments (profile_id, start, "end", stage, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (profile_id, start, "end", stage) DO NOTHING
	`

	_, err := p.pool.Exec(ctx, query, profileID, start, end, stage)
	return err
}

func (p *PostgresMetricsStorage) InsertWorkout(ctx context.Context, profileID uuid.UUID, start, end time.Time, label string, caloriesKcal *int) error {
	query := `
		INSERT INTO workouts (profile_id, start, "end", label, calories_kcal, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (profile_id, start, "end", label) DO NOTHING
	`

	_, err := p.pool.Exec(ctx, query, profileID, start, end, label, caloriesKcal)
	return err
}
