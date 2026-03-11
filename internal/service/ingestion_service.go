// Package service – Vitals Ingestion Service.
// Accepts raw payloads, validates, deduplicates, normalizes, and triggers risk evaluation.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smartbed/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// IngestionPayload is the inbound DTO from Apple Watch / iPhone bridge.
type IngestionPayload struct {
	UserID    string         `json:"user_id"    validate:"required"`
	DeviceID  string         `json:"device_id"  validate:"required"`
	Timestamp time.Time      `json:"timestamp"  validate:"required"`
	Metrics   MetricsPayload `json:"metrics"    validate:"required"`
}

// MetricsPayload contains all optional biometric fields.
type MetricsPayload struct {
	HeartRate         *float64 `json:"heart_rate"`
	SpO2              *float64 `json:"spo2"`
	StressLevel       *float64 `json:"stress_level"`
	SleepDurationMins *float64 `json:"sleep_duration_minutes"`
	SkinTemperature   *float64 `json:"skin_temperature"`
	MovementScore     *float64 `json:"movement_score"`
	SleepStage        *string  `json:"sleep_stage"`
}

// IngestionResult summarizes what was created after ingestion.
type IngestionResult struct {
	IngestionID   uuid.UUID   `json:"ingestion_id"`
	VitalEventIDs []uuid.UUID `json:"vital_event_ids"`
	Duplicates    int         `json:"duplicates"`
}

// IngestionService normalizes inbound payloads into time-series vital_events.
type IngestionService struct {
	db         *sqlx.DB
	riskEngine *RiskEngine
}

// NewIngestionService creates a new IngestionService.
func NewIngestionService(db *sqlx.DB, riskEngine *RiskEngine) *IngestionService {
	return &IngestionService{db: db, riskEngine: riskEngine}
}

// Ingest processes one ingestion payload end-to-end.
func (s *IngestionService) Ingest(ctx context.Context, payload IngestionPayload) (*IngestionResult, error) {
	userID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	// 1. Store raw payload
	rawJSON, _ := json.Marshal(payload)
	ingestionID := uuid.New()
	receivedAt := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO raw_ingestion_log (id, user_id, device_id, raw_payload, received_at)
		VALUES ($1,$2,$3,$4,$5)`,
		ingestionID, userID, payload.DeviceID, rawJSON, receivedAt,
	); err != nil {
		return nil, fmt.Errorf("store raw ingestion: %w", err)
	}

	// 2. Normalize each present metric into vital_events
	metricMap := s.extractMetrics(payload.Metrics)
	result := &IngestionResult{IngestionID: ingestionID}

	for metricType, value := range metricMap {
		eventID, isDup, err := s.upsertVitalEvent(ctx, userID, payload.DeviceID, payload.Timestamp, ingestionID, metricType, value)
		if err != nil {
			log.Error().Err(err).Str("metric", string(metricType)).Msg("vital event upsert failed")
			continue
		}
		if isDup {
			result.Duplicates++
			continue
		}
		result.VitalEventIDs = append(result.VitalEventIDs, eventID)

		// 3. Trigger asynchronous risk evaluation for non-duplicate events
		go func(eid uuid.UUID, mt domain.MetricType, v float64) {
			if _, err := s.riskEngine.EvaluateVitalRisk(context.Background(), userID, eid, mt, v, payload.Timestamp); err != nil {
				log.Error().Err(err).Str("metric", string(mt)).Msg("risk evaluation failed")
			}
		}(eventID, metricType, value)
	}

	// 4. Composite sleep disturbance check (if both HR and movement are present)
	if payload.Metrics.HeartRate != nil && payload.Metrics.MovementScore != nil {
		var stage domain.SleepStage
		if payload.Metrics.SleepStage != nil {
			stage = domain.SleepStage(*payload.Metrics.SleepStage)
		}
		go func() {
			_, err := s.riskEngine.EvaluateCompositeSleepDisturbance(
				context.Background(), userID,
				*payload.Metrics.HeartRate, *payload.Metrics.MovementScore,
				SleepDisturbanceContext{SleepStage: stage},
			)
			if err != nil {
				log.Warn().Err(err).Msg("composite sleep disturbance eval failed")
			}
		}()
	}

	return result, nil
}

// upsertVitalEvent inserts a vital event, returning (eventID, isDuplicate, error).
func (s *IngestionService) upsertVitalEvent(
	ctx context.Context,
	userID uuid.UUID,
	deviceID string,
	sourceTs time.Time,
	ingestionID uuid.UUID,
	metricType domain.MetricType,
	value float64,
) (uuid.UUID, bool, error) {
	id := uuid.New()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO vital_events (id, user_id, device_id, source_timestamp, ingestion_id, metric_type, value, unit, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'',$8)
		ON CONFLICT (device_id, source_timestamp, metric_type, user_id) DO NOTHING`,
		id, userID, deviceID, sourceTs, ingestionID, string(metricType), value, time.Now().UTC(),
	)
	if err != nil {
		return uuid.Nil, false, err
	}

	// Check if the ID was actually inserted or it was a no-op (duplicate)
	var insertedID uuid.UUID
	err = s.db.GetContext(ctx, &insertedID, `
		SELECT id FROM vital_events
		WHERE device_id=$1 AND source_timestamp=$2 AND metric_type=$3 AND user_id=$4`,
		deviceID, sourceTs, string(metricType), userID,
	)
	if err != nil {
		return uuid.Nil, false, err
	}

	isDuplicate := insertedID != id
	return insertedID, isDuplicate, nil
}

// LatestVitals holds the most recent value per metric type for a user.
type LatestVitals struct {
	UserID    uuid.UUID             `json:"user_id"`
	Metrics   map[string]VitalPoint `json:"metrics"`
	FetchedAt time.Time             `json:"fetched_at"`
}

// VitalPoint is a single metric reading with its timestamp.
type VitalPoint struct {
	Value      float64   `json:"value"`
	RecordedAt time.Time `json:"recorded_at"`
}

// GetLatestVitals returns the most recent value for each metric type for the given user.
func (s *IngestionService) GetLatestVitals(ctx context.Context, userID uuid.UUID) (*LatestVitals, error) {
	type row struct {
		MetricType string    `db:"metric_type"`
		Value      float64   `db:"value"`
		SourceTS   time.Time `db:"source_timestamp"`
	}
	var rows []row
	err := s.db.SelectContext(ctx, &rows, `
		SELECT DISTINCT ON (metric_type)
			metric_type, value, source_timestamp
		FROM vital_events
		WHERE user_id = $1
		ORDER BY metric_type, source_timestamp DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("get latest vitals: %w", err)
	}
	metrics := make(map[string]VitalPoint, len(rows))
	for _, r := range rows {
		metrics[r.MetricType] = VitalPoint{Value: r.Value, RecordedAt: r.SourceTS}
	}
	return &LatestVitals{
		UserID:    userID,
		Metrics:   metrics,
		FetchedAt: time.Now().UTC(),
	}, nil
}

// extractMetrics converts the payload DTO into a typed map.
func (s *IngestionService) extractMetrics(m MetricsPayload) map[domain.MetricType]float64 {
	result := map[domain.MetricType]float64{}
	if m.HeartRate != nil {
		result[domain.MetricHeartRate] = *m.HeartRate
	}
	if m.SpO2 != nil {
		result[domain.MetricSpO2] = *m.SpO2
	}
	if m.StressLevel != nil {
		result[domain.MetricStressLevel] = *m.StressLevel
	}
	if m.SleepDurationMins != nil {
		result[domain.MetricSleepDuration] = *m.SleepDurationMins
	}
	if m.SkinTemperature != nil {
		result[domain.MetricSkinTemperature] = *m.SkinTemperature
	}
	if m.MovementScore != nil {
		result[domain.MetricMovementScore] = *m.MovementScore
	}
	return result
}
