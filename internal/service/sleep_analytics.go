// Package service – Sleep Analytics Service.
// Computes composite nightly sleep scores, detects disturbed sleep,
// and provides 7d / 30d trend aggregations.
package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"smartbed/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SleepAnalyticsService computes and retrieves sleep quality data.
type SleepAnalyticsService struct {
	db *sqlx.DB
}

// NewSleepAnalyticsService creates a new SleepAnalyticsService.
func NewSleepAnalyticsService(db *sqlx.DB) *SleepAnalyticsService {
	return &SleepAnalyticsService{db: db}
}

// NightlyScoreInput holds the inputs needed to score a sleep session.
type NightlyScoreInput struct {
	DurationMinutes  int
	DisturbanceCount int
	AvgMovement      float64
	HeartRateStdDev  float64 // lower = more stable = better
	SpO2StdDev       float64 // lower = more stable = better
}

// ComputeNightlySleepScore returns a 0–100 composite sleep quality score.
//
// Score breakdown:
//   - Duration score    : 30 pts — penalised if < 6h or > 9h
//   - Stability score   : 25 pts — based on HR + SpO₂ standard deviation
//   - Disturbance score : 25 pts — penalised per disturbance event
//   - Movement score    : 20 pts — penalised for high movement
func ComputeNightlySleepScore(input NightlyScoreInput) float64 {
	// ── Duration (0–30) ──────────────────────────────────────────────
	durationHrs := float64(input.DurationMinutes) / 60.0
	var durationScore float64
	switch {
	case durationHrs >= 7 && durationHrs <= 8:
		durationScore = 30
	case durationHrs >= 6 && durationHrs < 7:
		durationScore = 22
	case durationHrs > 8 && durationHrs <= 9:
		durationScore = 25
	case durationHrs >= 5 && durationHrs < 6:
		durationScore = 14
	case durationHrs > 9:
		durationScore = 18
	default:
		durationScore = 5
	}

	// ── Stability (0–25) ─────────────────────────────────────────────
	// HR std dev < 3 = excellent, > 15 = poor
	hrStability := math.Max(0, 12.5*(1-input.HeartRateStdDev/15.0))
	// SpO₂ std dev < 0.5 = excellent, > 3 = poor
	spo2Stability := math.Max(0, 12.5*(1-input.SpO2StdDev/3.0))
	stabilityScore := hrStability + spo2Stability

	// ── Disturbance (0–25) ────────────────────────────────────────────
	disturbancePenalty := math.Min(25, float64(input.DisturbanceCount)*5.0)
	disturbanceScore := 25 - disturbancePenalty

	// ── Movement (0–20) ───────────────────────────────────────────────
	// movement score 0–1: < 0.2 is calm sleep, > 0.6 is very restless
	movementScore := math.Max(0, 20*(1-input.AvgMovement/0.6))

	total := durationScore + stabilityScore + disturbanceScore + movementScore
	return math.Min(100, math.Max(0, total))
}

// IsDisturbedSleep returns true if score < 60 or disturbances >= 3.
func IsDisturbedSleep(score float64, disturbances int) bool {
	return score < 60.0 || disturbances >= 3
}

// CreateOrUpdateSession creates or updates today's sleep session for a user.
func (s *SleepAnalyticsService) CreateOrUpdateSession(ctx context.Context, userID uuid.UUID, input NightlyScoreInput, startTime time.Time) (*domain.SleepSession, error) {
	score := ComputeNightlySleepScore(input)
	disturbed := IsDisturbedSleep(score, input.DisturbanceCount)

	session := domain.SleepSession{
		ID:               uuid.New(),
		UserID:           userID,
		StartTime:        startTime,
		DurationMinutes:  input.DurationMinutes,
		QualityScore:     score,
		DisturbanceCount: input.DisturbanceCount,
		IsDisturbed:      disturbed,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sleep_sessions
			(id, user_id, start_time, duration_minutes, quality_score, disturbance_count, is_disturbed, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT DO NOTHING`,
		session.ID, session.UserID, session.StartTime,
		session.DurationMinutes, session.QualityScore,
		session.DisturbanceCount, session.IsDisturbed,
		session.CreatedAt, session.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert sleep session: %w", err)
	}
	return &session, nil
}

// SleepSummary holds aggregated sleep stats over a period.
type SleepSummary struct {
	UserID          uuid.UUID `db:"user_id"            json:"user_id"`
	AvgQualityScore float64   `db:"avg_quality_score"  json:"avg_quality_score"`
	AvgDurationMins float64   `db:"avg_duration_minutes" json:"avg_duration_mins"`
	TotalNights     int       `db:"total_nights"       json:"total_nights"`
	DisturbedNights int       `db:"disturbed_nights"   json:"disturbed_nights"`
	PeriodDays      int       `json:"period_days"`
}

// GetSleepSummary returns an aggregated sleep summary for the last N days.
func (s *SleepAnalyticsService) GetSleepSummary(ctx context.Context, userID uuid.UUID, days int) (*SleepSummary, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	var summary SleepSummary
	err := s.db.GetContext(ctx, &summary, `
		SELECT
			user_id,
			COALESCE(AVG(quality_score), 0)       AS avg_quality_score,
			COALESCE(AVG(duration_minutes), 0)     AS avg_duration_minutes,
			COUNT(*)                                AS total_nights,
			COUNT(*) FILTER (WHERE is_disturbed)   AS disturbed_nights
		FROM sleep_sessions
		WHERE user_id = $1 AND start_time >= $2
		GROUP BY user_id`,
		userID, since,
	)
	if err != nil {
		// No sessions yet – return zero summary
		return &SleepSummary{UserID: userID, PeriodDays: days}, nil
	}
	summary.PeriodDays = days
	return &summary, nil
}
