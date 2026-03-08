// Package service – Alert Service.
// Creates, routes, and persists health alerts.
package service

import (
	"context"
	"fmt"
	"time"

	"smartbed/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AlertService creates and manages health alerts.
type AlertService struct {
	db              *sqlx.DB
	notificationSvc *NotificationService
}

// NewAlertService creates a new AlertService.
func NewAlertService(db *sqlx.DB, notificationSvc *NotificationService) *AlertService {
	return &AlertService{db: db, notificationSvc: notificationSvc}
}

// CreateAlert persists an alert and triggers notification routing.
func (s *AlertService) CreateAlert(
	ctx context.Context,
	userID uuid.UUID,
	alertType domain.AlertType,
	riskLevel domain.RiskLevel,
	metricType domain.MetricType,
	metricValue float64,
	message string,
) error {
	alert := domain.Alert{
		ID:          uuid.New(),
		UserID:      userID,
		AlertType:   alertType,
		RiskLevel:   riskLevel,
		Message:     message,
		MetricType:  &metricType,
		MetricValue: &metricValue,
		CreatedAt:   time.Now().UTC(),
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO alerts (id, user_id, alert_type, risk_level, message, metric_type, metric_value, is_acknowledged, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,FALSE,$8)`,
		alert.ID, alert.UserID, string(alert.AlertType), string(alert.RiskLevel),
		alert.Message, string(metricType), metricValue, alert.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}

	// Route notifications based on risk level
	go s.routeNotifications(context.Background(), userID, alert)

	return nil
}

// routeNotifications dispatches notifications based on alert severity.
func (s *AlertService) routeNotifications(ctx context.Context, userID uuid.UUID, alert domain.Alert) {
	switch alert.RiskLevel {
	case domain.RiskCritical:
		// Critical: notify doctor + caregiver + emergency contacts
		s.notificationSvc.NotifyDoctor(ctx, userID, alert)
		s.notificationSvc.NotifyCaregiver(ctx, userID, alert)
		s.notificationSvc.NotifyEmergencyContacts(ctx, userID, alert)
	case domain.RiskHigh:
		// High: notify doctor + caregiver
		s.notificationSvc.NotifyDoctor(ctx, userID, alert)
		s.notificationSvc.NotifyCaregiver(ctx, userID, alert)
	case domain.RiskMild:
		// Mild: notify caregiver only
		s.notificationSvc.NotifyCaregiver(ctx, userID, alert)
	}
}

// GetAlerts fetches paginated alerts for a user, optionally filtered by acknowledged state.
func (s *AlertService) GetAlerts(ctx context.Context, userID uuid.UUID, onlyUnread bool, limit, offset int) ([]domain.Alert, error) {
	query := `
		SELECT id, user_id, alert_type, risk_level, message, metric_type, metric_value,
		       is_acknowledged, acknowledged_by, acknowledged_at, created_at
		FROM alerts
		WHERE user_id = $1`
	args := []interface{}{userID}

	if onlyUnread {
		query += " AND is_acknowledged = FALSE"
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d OFFSET %d", limit, offset)

	var alerts []domain.Alert
	if err := s.db.SelectContext(ctx, &alerts, query, args...); err != nil {
		return nil, err
	}
	return alerts, nil
}

// AcknowledgeAlert marks an alert as acknowledged by a user.
func (s *AlertService) AcknowledgeAlert(ctx context.Context, alertID, acknowledgedBy uuid.UUID) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		UPDATE alerts SET is_acknowledged=$1, acknowledged_by=$2, acknowledged_at=$3
		WHERE id=$4`,
		true, acknowledgedBy, now, alertID,
	)
	return err
}
