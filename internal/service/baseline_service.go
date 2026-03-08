// Package service – Baseline Engine.
// Resolves the applicable health range for a user: user override → age/sex population range.
package service

import (
	"context"
	"fmt"
	"time"

	"smartbed/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// baselineRow is a local scan target to avoid scanning directly into domain.BaselineRange.
type baselineRow struct {
	MinValue    float64 `db:"min_value"`
	MaxValue    float64 `db:"max_value"`
	NormalValue float64 `db:"normal_value"`
	Unit        string  `db:"unit"`
}

// BaselineService resolves applicable health baselines for users.
type BaselineService struct {
	db *sqlx.DB
}

// NewBaselineService creates a new BaselineService.
func NewBaselineService(db *sqlx.DB) *BaselineService {
	return &BaselineService{db: db}
}

// GetApplicableBaseline returns the best-matching baseline for a user, metric, and timestamp.
// Priority: active doctor override > age/sex population range.
func (s *BaselineService) GetApplicableBaseline(ctx context.Context, userID uuid.UUID, metricType domain.MetricType, ts time.Time) (*domain.BaselineRange, error) {
	// 1. Try user-specific doctor override
	override, err := s.getUserOverride(ctx, userID, metricType, ts)
	if err != nil {
		return nil, fmt.Errorf("get user override: %w", err)
	}
	if override != nil {
		return override, nil
	}

	// 2. Fall back to population-level age/sex baseline
	return s.getPopulationBaseline(ctx, userID, metricType)
}

// getUserOverride fetches the most recent active doctor-adjusted baseline for a user.
func (s *BaselineService) getUserOverride(ctx context.Context, userID uuid.UUID, metricType domain.MetricType, ts time.Time) (*domain.BaselineRange, error) {
	var row baselineRow
	err := s.db.GetContext(ctx, &row, `
		SELECT min_value, max_value, normal_value, '' AS unit
		FROM user_baseline_overrides
		WHERE user_id     = $1
		  AND metric_type = $2
		  AND valid_from  <= $3
		  AND (valid_until IS NULL OR valid_until >= $3)
		ORDER BY valid_from DESC
		LIMIT 1`,
		userID, string(metricType), ts,
	)
	if err != nil {
		// No override found — return nil (not an error)
		return nil, nil //nolint:nilerr
	}
	return &domain.BaselineRange{
		MetricType:  metricType,
		MinValue:    row.MinValue,
		MaxValue:    row.MaxValue,
		NormalValue: row.NormalValue,
		Unit:        row.Unit,
	}, nil
}

// getPopulationBaseline fetches the age/sex population baseline for a user.
func (s *BaselineService) getPopulationBaseline(ctx context.Context, userID uuid.UUID, metricType domain.MetricType) (*domain.BaselineRange, error) {
	// Fetch the user's age and sex from their profile
	var profile struct {
		Age int        `db:"age"`
		Sex domain.Sex `db:"sex"`
	}
	if err := s.db.GetContext(ctx, &profile, `
		SELECT age, sex FROM user_profiles WHERE user_id = $1`, userID,
	); err != nil {
		return nil, fmt.Errorf("fetch user profile for baseline: %w", err)
	}

	ageGroup := domain.AgeGroupFromAge(profile.Age)

	var row baselineRow
	if err := s.db.GetContext(ctx, &row, `
		SELECT min_value, max_value, normal_value, unit
		FROM baseline_ranges
		WHERE age_group   = $1
		  AND sex         = $2
		  AND metric_type = $3`,
		string(ageGroup), string(profile.Sex), string(metricType),
	); err != nil {
		return nil, fmt.Errorf("no baseline for age_group=%s sex=%s metric=%s: %w", ageGroup, profile.Sex, metricType, err)
	}

	return &domain.BaselineRange{
		AgeGroup:    ageGroup,
		Sex:         profile.Sex,
		MetricType:  metricType,
		MinValue:    row.MinValue,
		MaxValue:    row.MaxValue,
		NormalValue: row.NormalValue,
		Unit:        row.Unit,
	}, nil
}

// SetUserOverride creates a doctor-adjusted baseline override for a patient.
func (s *BaselineService) SetUserOverride(ctx context.Context, override domain.UserBaselineOverride) error {
	override.ID = uuid.New()
	override.ValidFrom = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_baseline_overrides
			(id, user_id, metric_type, min_value, max_value, normal_value, set_by_doctor, valid_from, valid_until)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		override.ID, override.UserID, string(override.MetricType),
		override.MinValue, override.MaxValue, override.NormalValue,
		override.SetByDoctor, override.ValidFrom, override.ValidUntil,
	)
	return err
}
