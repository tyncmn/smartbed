// Package service – Sleep Dashboard Service.
// Aggregates sleep summary, nightly timeline, stage distribution, health alerts,
// AI analysis, and predictive metrics into a single dashboard response.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"smartbed/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SleepDashboardService aggregates all sleep analytics data.
type SleepDashboardService struct {
	db       *sqlx.DB
	sleepSvc *SleepAnalyticsService
	alertSvc *AlertService
	aiSvc    *AIAnalysisService // optional; nil if OPENAI_API_KEY is not set
}

// NewSleepDashboardService creates a new SleepDashboardService.
// aiSvc may be nil — AI analysis is skipped when not configured.
func NewSleepDashboardService(
	db *sqlx.DB,
	sleepSvc *SleepAnalyticsService,
	alertSvc *AlertService,
	aiSvc *AIAnalysisService,
) *SleepDashboardService {
	return &SleepDashboardService{db: db, sleepSvc: sleepSvc, alertSvc: alertSvc, aiSvc: aiSvc}
}

// ─── Response DTOs ────────────────────────────────────────────────────────────

// DashboardPeriod describes the time window covered by the dashboard.
type DashboardPeriod struct {
	Days int       `json:"days"`
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// DailyTimelineEntry holds one night's aggregated sleep data.
type DailyTimelineEntry struct {
	Date             string   `json:"date"` // YYYY-MM-DD (local date of session start)
	DurationMins     float64  `json:"duration_mins"`
	QualityScore     float64  `json:"quality_score"`
	IsDisturbed      bool     `json:"is_disturbed"`
	DisturbanceCount int      `json:"disturbance_count"`
	AvgHeartRate     *float64 `json:"avg_heart_rate"`
	AvgSpO2          *float64 `json:"avg_spo2"`
	AvgMovement      *float64 `json:"avg_movement"`
}

// SleepStageShare is the inferred distribution of sleep stages.
type SleepStageShare struct {
	Stage      string  `json:"stage"`       // deep | rem | light | awake
	Percentage float64 `json:"percentage"`  // 0–100
	AvgMinutes float64 `json:"avg_minutes"` // average minutes per night in this stage
}

// SleepPrediction contains trend analysis and a 7-day quality forecast.
type SleepPrediction struct {
	NextNightQuality   float64   `json:"next_night_quality"`   // 0–100
	TrendDirection     string    `json:"trend_direction"`      // improving | declining | stable
	PredictedRiskLevel string    `json:"predicted_risk_level"` // normal | mild | high | critical
	QualityForecast    []float64 `json:"quality_forecast"`     // next 7 nights (0–100 each)
	HealthRisks        []string  `json:"health_risks"`
}

// SleepDashboardResponse is the complete payload for the dashboard endpoint.
type SleepDashboardResponse struct {
	UserID            string               `json:"user_id"`
	Period            DashboardPeriod      `json:"period"`
	Summary           *SleepSummary        `json:"summary"`
	Timeline          []DailyTimelineEntry `json:"timeline"`
	StageDistribution []SleepStageShare    `json:"stage_distribution"`
	HealthAlerts      []domain.Alert       `json:"health_alerts"`
	AIAnalysis        *AIAnalysisResult    `json:"ai_analysis,omitempty"`
	Predictions       SleepPrediction      `json:"predictions"`
	GeneratedAt       time.Time            `json:"generated_at"`
}

// ─── Public API ───────────────────────────────────────────────────────────────

// GetDashboard fetches all dashboard data concurrently and returns the response.
func (s *SleepDashboardService) GetDashboard(ctx context.Context, userID uuid.UUID, days int) (*SleepDashboardResponse, error) {
	if days < 1 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -days)

	// ── Concurrent fetch ──────────────────────────────────────────────────────
	var (
		mu       sync.Mutex
		wg       sync.WaitGroup
		fetchErr error

		summary  *SleepSummary
		timeline []DailyTimelineEntry
		stages   []SleepStageShare
		alerts   []domain.Alert
		profile  userProfileRow
	)

	launch := func(fn func() error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				mu.Lock()
				if fetchErr == nil {
					fetchErr = err
				}
				mu.Unlock()
			}
		}()
	}

	launch(func() error {
		s, err := s.sleepSvc.GetSleepSummary(ctx, userID, days)
		if err != nil {
			return fmt.Errorf("sleep summary: %w", err)
		}
		mu.Lock()
		summary = s
		mu.Unlock()
		return nil
	})

	launch(func() error {
		t, err := s.getSleepTimeline(ctx, userID, from)
		if err != nil {
			return fmt.Errorf("timeline: %w", err)
		}
		mu.Lock()
		timeline = t
		mu.Unlock()
		return nil
	})

	launch(func() error {
		st, err := s.getStageDistribution(ctx, userID, from)
		if err != nil {
			return fmt.Errorf("stage distribution: %w", err)
		}
		mu.Lock()
		stages = st
		mu.Unlock()
		return nil
	})

	launch(func() error {
		a, err := s.alertSvc.GetAlerts(ctx, userID, false, 20, 0)
		if err != nil {
			return fmt.Errorf("alerts: %w", err)
		}
		mu.Lock()
		alerts = a
		mu.Unlock()
		return nil
	})

	launch(func() error {
		p, err := s.getUserProfile(ctx, userID)
		if err != nil {
			// not fatal — AI degrades gracefully without profile
			return nil
		}
		mu.Lock()
		profile = p
		mu.Unlock()
		return nil
	})

	wg.Wait()
	if fetchErr != nil {
		return nil, fetchErr
	}

	// Ensure slices are never null in JSON
	if alerts == nil {
		alerts = []domain.Alert{}
	}
	if timeline == nil {
		timeline = []DailyTimelineEntry{}
	}
	if stages == nil {
		stages = []SleepStageShare{}
	}

	// ── Predictions ───────────────────────────────────────────────────────────
	predictions := computePredictions(timeline)

	// ── AI Analysis (optional) ────────────────────────────────────────────────
	var aiResult *AIAnalysisResult
	if s.aiSvc != nil {
		alertMsgs := make([]string, 0, len(alerts))
		for _, a := range alerts {
			alertMsgs = append(alertMsgs, a.Message)
		}
		aiResult, _ = s.aiSvc.Analyze(ctx, AnalyzeInput{
			UserID:          userID,
			Age:             profile.Age,
			Sex:             profile.Sex,
			Conditions:      profile.ExistingConditions,
			Summary:         summary,
			Timeline:        timeline,
			RecentAlertMsgs: alertMsgs,
		})
		// AI failure is non-fatal — dashboard still returns without it
	}

	return &SleepDashboardResponse{
		UserID: userID.String(),
		Period: DashboardPeriod{
			Days: days,
			From: from,
			To:   now,
		},
		Summary:           summary,
		Timeline:          timeline,
		StageDistribution: stages,
		HealthAlerts:      alerts,
		AIAnalysis:        aiResult,
		Predictions:       predictions,
		GeneratedAt:       now,
	}, nil
}

// ─── Timeline ─────────────────────────────────────────────────────────────────

type sleepSessionRow struct {
	StartTime        time.Time `db:"start_time"`
	DurationMinutes  int       `db:"duration_minutes"`
	QualityScore     float64   `db:"quality_score"`
	IsDisturbed      bool      `db:"is_disturbed"`
	DisturbanceCount int       `db:"disturbance_count"`
	AvgHeartRate     *float64  `db:"avg_heart_rate"`
	AvgSpO2          *float64  `db:"avg_spo2"`
	AvgMovementScore *float64  `db:"avg_movement_score"`
}

func (s *SleepDashboardService) getSleepTimeline(ctx context.Context, userID uuid.UUID, from time.Time) ([]DailyTimelineEntry, error) {
	var rows []sleepSessionRow
	err := s.db.SelectContext(ctx, &rows, `
		SELECT start_time, duration_minutes, quality_score, is_disturbed,
		       disturbance_count, avg_heart_rate, avg_spo2, avg_movement_score
		FROM sleep_sessions
		WHERE user_id = $1
		  AND start_time >= $2
		ORDER BY start_time ASC`, userID, from)
	if err != nil {
		return nil, err
	}

	entries := make([]DailyTimelineEntry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, DailyTimelineEntry{
			Date:             r.StartTime.Format("2006-01-02"),
			DurationMins:     float64(r.DurationMinutes),
			QualityScore:     r.QualityScore,
			IsDisturbed:      r.IsDisturbed,
			DisturbanceCount: r.DisturbanceCount,
			AvgHeartRate:     r.AvgHeartRate,
			AvgSpO2:          r.AvgSpO2,
			AvgMovement:      r.AvgMovementScore,
		})
	}
	return entries, nil
}

// ─── Stage Distribution ───────────────────────────────────────────────────────

// getStageDistribution infers sleep stage ratios from movement_score vital_events.
// Thresholds (heuristic, based on actigraphy research):
//
//	< 0.15   → deep sleep
//	0.15–0.35 → REM
//	0.35–0.60 → light sleep
//	≥ 0.60   → awake
func (s *SleepDashboardService) getStageDistribution(ctx context.Context, userID uuid.UUID, from time.Time) ([]SleepStageShare, error) {
	type mvRow struct {
		Value float64 `db:"value"`
	}
	var rows []mvRow
	err := s.db.SelectContext(ctx, &rows, `
		SELECT value
		FROM vital_events
		WHERE user_id      = $1
		  AND metric_type  = 'movement_score'
		  AND source_timestamp >= $2
		  AND EXTRACT(HOUR FROM source_timestamp AT TIME ZONE 'UTC') IN
		      (21,22,23,0,1,2,3,4,5,6,7)
		ORDER BY source_timestamp ASC`, userID, from)
	if err != nil {
		return nil, err
	}

	counts := map[string]int{"deep": 0, "rem": 0, "light": 0, "awake": 0}
	for _, r := range rows {
		switch {
		case r.Value < 0.15:
			counts["deep"]++
		case r.Value < 0.35:
			counts["rem"]++
		case r.Value < 0.60:
			counts["light"]++
		default:
			counts["awake"]++
		}
	}

	total := len(rows)
	if total == 0 {
		return []SleepStageShare{}, nil
	}

	// Estimate minutes: assume one reading per 5 minutes during night
	const minutesPerReading = 5.0

	order := []string{"deep", "rem", "light", "awake"}
	result := make([]SleepStageShare, 0, 4)
	for _, stage := range order {
		n := counts[stage]
		result = append(result, SleepStageShare{
			Stage:      stage,
			Percentage: roundTo2(float64(n) * 100.0 / float64(total)),
			AvgMinutes: roundTo2(float64(n) * minutesPerReading),
		})
	}
	return result, nil
}

// ─── User Profile ─────────────────────────────────────────────────────────────

type userProfileRow struct {
	Age                int      `db:"age"`
	Sex                string   `db:"sex"`
	ExistingConditions []string // unmarshalled separately
}

func (s *SleepDashboardService) getUserProfile(ctx context.Context, userID uuid.UUID) (userProfileRow, error) {
	type rawRow struct {
		Age                int    `db:"age"`
		Sex                string `db:"sex"`
		ExistingConditions []byte `db:"existing_conditions"`
	}
	var raw rawRow
	err := s.db.GetContext(ctx, &raw, `
		SELECT age, sex, existing_conditions
		FROM user_profiles
		WHERE user_id = $1`, userID)
	if err != nil {
		return userProfileRow{}, err
	}

	var conditions []string
	if len(raw.ExistingConditions) > 0 {
		// stored as jsonb array '["hypertension","diabetes"]'
		_ = json.Unmarshal(raw.ExistingConditions, &conditions)
	}

	return userProfileRow{
		Age:                raw.Age,
		Sex:                raw.Sex,
		ExistingConditions: conditions,
	}, nil
}

// ─── Predictions ──────────────────────────────────────────────────────────────

// computePredictions uses simple linear regression on historical quality scores
// to predict trend direction and project the next 7 nights.
func computePredictions(timeline []DailyTimelineEntry) SleepPrediction {
	if len(timeline) == 0 {
		return SleepPrediction{
			NextNightQuality:   0,
			TrendDirection:     "stable",
			PredictedRiskLevel: "normal",
			QualityForecast:    make([]float64, 7),
			HealthRisks:        []string{},
		}
	}

	n := float64(len(timeline))
	var sumX, sumY, sumXY, sumX2 float64
	for i, e := range timeline {
		x := float64(i)
		y := e.QualityScore
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Linear regression: y = a + b*x
	denom := n*sumX2 - sumX*sumX
	var slope float64
	if denom != 0 {
		slope = (n*sumXY - sumX*sumY) / denom
	}
	intercept := (sumY - slope*sumX) / n

	// Project next 7 nights
	forecast := make([]float64, 7)
	for i := range forecast {
		x := n + float64(i)
		v := intercept + slope*x
		forecast[i] = roundTo2(clampQuality(v))
	}

	nextQuality := forecast[0]

	trend := "stable"
	switch {
	case slope > 1.5:
		trend = "improving"
	case slope < -1.5:
		trend = "declining"
	}

	// Risk level based on next-night projection and average quality
	avgQuality := sumY / n
	riskLevel := "normal"
	var healthRisks []string

	switch {
	case nextQuality < 30 || avgQuality < 35:
		riskLevel = "critical"
		healthRisks = append(healthRisks, "Severe sleep deprivation — consult a physician immediately")
	case nextQuality < 50 || avgQuality < 50:
		riskLevel = "high"
		healthRisks = append(healthRisks, "Consistently poor sleep quality may impair cognitive function and immune response")
	case nextQuality < 65:
		riskLevel = "mild"
		healthRisks = append(healthRisks, "Sleep quality below optimal — consider sleep hygiene improvements")
	}

	if trend == "declining" {
		healthRisks = append(healthRisks, "Declining sleep trend detected — monitor for underlying causes")
	}

	if healthRisks == nil {
		healthRisks = []string{}
	}

	return SleepPrediction{
		NextNightQuality:   nextQuality,
		TrendDirection:     trend,
		PredictedRiskLevel: riskLevel,
		QualityForecast:    forecast,
		HealthRisks:        healthRisks,
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func clampQuality(v float64) float64 {
	return math.Max(0, math.Min(100, v))
}

func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}
