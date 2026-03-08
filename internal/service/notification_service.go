// Package service – Notification Service (mock providers).
// Sends notifications through caregiver, doctor, and emergency channels.
// Designed for easy swap-in with real providers (SendGrid, Twilio, FCM).
package service

import (
	"context"
	"fmt"
	"time"

	"smartbed/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// NotificationService manages outbound notifications.
type NotificationService struct {
	db *sqlx.DB
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(db *sqlx.DB) *NotificationService {
	return &NotificationService{db: db}
}

// NotifyCaregiver sends an alert notification to the patient's caregiver.
func (s *NotificationService) NotifyCaregiver(ctx context.Context, patientID uuid.UUID, alert domain.Alert) {
	caregiverID, err := s.fetchCaregiverID(ctx, patientID)
	if err != nil || caregiverID == uuid.Nil {
		log.Warn().Err(err).Str("patient", patientID.String()).Msg("caregiver not found; notification skipped")
		return
	}
	s.send(ctx, domain.Notification{
		UserID:      patientID,
		RecipientID: caregiverID,
		Channel:     "push",
		Subject:     fmt.Sprintf("SmartBed Alert: %s", alert.AlertType),
		Body:        alert.Message,
	})
}

// NotifyDoctor sends an alert notification to the patient's doctor.
func (s *NotificationService) NotifyDoctor(ctx context.Context, patientID uuid.UUID, alert domain.Alert) {
	doctorID, err := s.fetchDoctorID(ctx, patientID)
	if err != nil || doctorID == uuid.Nil {
		log.Warn().Err(err).Str("patient", patientID.String()).Msg("doctor not found; notification skipped")
		return
	}
	s.send(ctx, domain.Notification{
		UserID:      patientID,
		RecipientID: doctorID,
		Channel:     "email",
		Subject:     fmt.Sprintf("[DOCTOR ALERT] %s – Patient %s", alert.AlertType, patientID),
		Body:        alert.Message,
	})
}

// NotifyEmergencyContacts sends critical alerts to all emergency contacts by priority.
func (s *NotificationService) NotifyEmergencyContacts(ctx context.Context, patientID uuid.UUID, alert domain.Alert) {
	var contacts []struct {
		ID    uuid.UUID `db:"id"`
		Phone string    `db:"phone"`
		Email string    `db:"email"`
	}
	if err := s.db.SelectContext(ctx, &contacts, `
		SELECT id, phone, email FROM emergency_contacts
		WHERE user_id = $1 ORDER BY priority ASC`, patientID,
	); err != nil {
		log.Error().Err(err).Msg("fetch emergency contacts failed")
		return
	}
	for _, c := range contacts {
		s.send(ctx, domain.Notification{
			UserID:      patientID,
			RecipientID: c.ID,
			Channel:     "sms",
			Subject:     "EMERGENCY: SmartBed Critical Alert",
			Body:        fmt.Sprintf("CRITICAL ALERT for your contact: %s", alert.Message),
		})
	}
}

// send dispatches a notification through the appropriate mock provider and persists the result.
func (s *NotificationService) send(ctx context.Context, n domain.Notification) {
	n.ID = uuid.New()
	n.CreatedAt = time.Now().UTC()

	// MOCK SEND: In production, replace with SendGrid/Twilio/FCM SDK calls
	log.Info().
		Str("channel", n.Channel).
		Str("recipient", n.RecipientID.String()).
		Str("subject", n.Subject).
		Msg("[MOCK] notification dispatched")

	now := time.Now().UTC()
	n.Status = "sent"
	n.SentAt = &now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, recipient_id, channel, subject, body, status, sent_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		n.ID, n.UserID, n.RecipientID, n.Channel, n.Subject, n.Body, n.Status, n.SentAt, n.CreatedAt,
	)
	if err != nil {
		log.Error().Err(err).Msg("persist notification failed")
	}
}

func (s *NotificationService) fetchCaregiverID(ctx context.Context, patientID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.GetContext(ctx, &id, `SELECT caregiver_user_id FROM user_profiles WHERE user_id=$1`, patientID)
	return id, err
}

func (s *NotificationService) fetchDoctorID(ctx context.Context, patientID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.db.GetContext(ctx, &id, `SELECT doctor_user_id FROM user_profiles WHERE user_id=$1`, patientID)
	return id, err
}
