// Package service – Risk Engine.
// Implements deviation calculation, risk percentage, risk classification,
// per-metric evaluation, and composite sleep disturbance detection.
package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"smartbed/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// ─── Risk Math ────────────────────────────────────────────────────────────────

// CalculateDeviation returns current - normal.
func CalculateDeviation(current, normal float64) float64 {
	return current - normal
}

// CalculateRiskPercentage returns abs(deviation / normal) * 100.
// Returns 0 if normal is zero to avoid division by zero.
func CalculateRiskPercentage(current, normal float64) float64 {
	if normal == 0 {
		return 0
	}
	return math.Abs((current-normal)/normal) * 100
}

// ClassifyRisk maps a risk percentage to a RiskLevel.
// Thresholds (from spec):
//   - Normal   : 0–5%
//   - Mild     : 5–10%
//   - High     : 10–20%
//   - Critical : > 20%
func ClassifyRisk(riskPct float64) domain.RiskLevel {
	switch {
	case riskPct > 20:
		return domain.RiskCritical
	case riskPct > 10:
		return domain.RiskHigh
	case riskPct > 5:
		return domain.RiskMild
	default:
		return domain.RiskNormal
	}
}

// SleepDisturbanceContext carries contextual information for composite evaluation.
type SleepDisturbanceContext struct {
	SleepSessionID  uuid.UUID
	SleepStage      domain.SleepStage
	DurationMinutes int
}

// ─── Risk Engine Service ──────────────────────────────────────────────────────

// RiskEngine evaluates biometric risk for a given user.
type RiskEngine struct {
	db          *sqlx.DB
	baselineSvc *BaselineService
	alertSvc    *AlertService
}

// NewRiskEngine creates a new RiskEngine.
func NewRiskEngine(db *sqlx.DB, baselineSvc *BaselineService, alertSvc *AlertService) *RiskEngine {
	return &RiskEngine{db: db, baselineSvc: baselineSvc, alertSvc: alertSvc}
}

// EvaluateVitalRisk performs full risk analysis for one metric value:
// 1. Fetch applicable baseline
// 2. Calculate deviation + risk %
// 3. Classify risk level
// 4. Persist risk evaluation
// 5. Generate alert if above normal
func (e *RiskEngine) EvaluateVitalRisk(
	ctx context.Context,
	userID uuid.UUID,
	vitalEventID uuid.UUID,
	metricType domain.MetricType,
	value float64,
	ts time.Time,
) (*domain.RiskEvaluation, error) {
	baseline, err := e.baselineSvc.GetApplicableBaseline(ctx, userID, metricType, ts)
	if err != nil {
		// Log but do not fail ingestion — some metrics may lack baseline data
		log.Warn().Err(err).Str("metric", string(metricType)).Msg("baseline not found; skipping risk eval")
		return nil, nil
	}

	normalVal := baseline.NormalValue
	deviation := CalculateDeviation(value, normalVal)
	riskPct := CalculateRiskPercentage(value, normalVal)
	riskLevel := ClassifyRisk(riskPct)

	// Apply additional metric-specific hard-threshold checks
	riskLevel = e.applyHardThresholds(metricType, value, riskLevel, baseline)

	eval := &domain.RiskEvaluation{
		ID:             uuid.New(),
		UserID:         userID,
		VitalEventID:   vitalEventID,
		MetricType:     metricType,
		MeasuredValue:  value,
		NormalValue:    normalVal,
		Deviation:      deviation,
		RiskPercentage: riskPct,
		RiskLevel:      riskLevel,
		EvaluatedAt:    ts,
		CreatedAt:      time.Now().UTC(),
	}

	if err := e.persistEvaluation(ctx, eval); err != nil {
		return nil, fmt.Errorf("persist risk eval: %w", err)
	}

	// Generate an alert for anything above normal
	if riskLevel != domain.RiskNormal {
		alertType := e.alertTypeFor(metricType, value, baseline)
		if err := e.alertSvc.CreateAlert(ctx, userID, alertType, riskLevel, metricType, value, e.buildAlertMessage(metricType, value, riskLevel, deviation)); err != nil {
			log.Warn().Err(err).Msg("alert creation failed; continuing")
		}
	}

	return eval, nil
}

// applyHardThresholds enforces metric-specific clinical thresholds that
// override the percentage-based classification (e.g., SpO₂ < 92 is always critical).
func (e *RiskEngine) applyHardThresholds(metricType domain.MetricType, value float64, current domain.RiskLevel, b *domain.BaselineRange) domain.RiskLevel {
	switch metricType {
	case domain.MetricSpO2:
		if value < 90 {
			return domain.RiskCritical
		}
		if value < 92 {
			return domain.RiskHigh
		}
		if value < 95 {
			return better(current, domain.RiskMild)
		}
	case domain.MetricHeartRate:
		if value > b.MaxValue*1.25 || value < b.MinValue*0.7 {
			return domain.RiskCritical
		}
	case domain.MetricSkinTemperature:
		if value >= 39.0 {
			return domain.RiskCritical
		}
		if value >= 38.0 {
			return better(current, domain.RiskHigh)
		}
	}
	return current
}

// better returns the more severe of two risk levels.
func better(a, b domain.RiskLevel) domain.RiskLevel {
	order := map[domain.RiskLevel]int{
		domain.RiskNormal:   0,
		domain.RiskMild:     1,
		domain.RiskHigh:     2,
		domain.RiskCritical: 3,
	}
	if order[a] >= order[b] {
		return a
	}
	return b
}

// EvaluateCompositeSleepDisturbance detects disturbed sleep from combined signals.
// Rule: High movement score AND elevated heart rate during sleep = disturbed.
func (e *RiskEngine) EvaluateCompositeSleepDisturbance(
	ctx context.Context,
	userID uuid.UUID,
	heartRate float64,
	movementScore float64,
	sleepCtx SleepDisturbanceContext,
) (bool, error) {
	hrBaseline, err := e.baselineSvc.GetApplicableBaseline(ctx, userID, domain.MetricHeartRate, time.Now())
	if err != nil || hrBaseline == nil {
		return false, nil
	}
	mvBaseline, err := e.baselineSvc.GetApplicableBaseline(ctx, userID, domain.MetricMovementScore, time.Now())
	if err != nil || mvBaseline == nil {
		return false, nil
	}

	hrElevated := heartRate > hrBaseline.NormalValue*1.10   // >10% above normal sleeping HR
	movElevated := movementScore > mvBaseline.MaxValue*0.75 // movement >75% of the upper bound

	disturbed := hrElevated && movElevated

	if disturbed {
		msg := fmt.Sprintf("Disturbed sleep detected: HR=%.1f bpm (normal %.1f), movement=%.2f", heartRate, hrBaseline.NormalValue, movementScore)
		if err := e.alertSvc.CreateAlert(ctx, userID, domain.AlertDisturbedSleep, domain.RiskMild, domain.MetricMovementScore, movementScore, msg); err != nil {
			log.Warn().Err(err).Msg("disturbed sleep alert failed")
		}
	}

	return disturbed, nil
}

func (e *RiskEngine) persistEvaluation(ctx context.Context, eval *domain.RiskEvaluation) error {
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO risk_evaluations
			(id, user_id, vital_event_id, metric_type, measured_value, normal_value, deviation, risk_percentage, risk_level, evaluated_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		eval.ID, eval.UserID, eval.VitalEventID, string(eval.MetricType),
		eval.MeasuredValue, eval.NormalValue, eval.Deviation,
		eval.RiskPercentage, string(eval.RiskLevel), eval.EvaluatedAt, eval.CreatedAt,
	)
	return err
}

func (e *RiskEngine) alertTypeFor(metricType domain.MetricType, value float64, b *domain.BaselineRange) domain.AlertType {
	switch metricType {
	case domain.MetricHeartRate:
		return domain.AlertAbnormalHeartRate
	case domain.MetricSpO2:
		return domain.AlertLowOxygen
	case domain.MetricSkinTemperature:
		if value > b.MaxValue {
			return domain.AlertElevatedTemperature
		}
	}
	return domain.AlertHighRisk
}

func (e *RiskEngine) buildAlertMessage(metricType domain.MetricType, value float64, level domain.RiskLevel, deviation float64) string {
	dir := "above"
	if deviation < 0 {
		dir = "below"
	}
	return fmt.Sprintf("%s: %.2f (%s normal, risk=%s, deviation=%.2f)", metricType, value, dir, level, math.Abs(deviation))
}
