// Package domain defines core domain types used across all SmartBed modules.
// This package has zero external dependencies — it is pure business logic types.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ─── Enums ────────────────────────────────────────────────────────────────────

// Role represents a user's access role.
type Role string

const (
	RoleAdmin     Role = "admin"
	RoleDoctor    Role = "doctor"
	RoleCaregiver Role = "caregiver"
	RoleOperator  Role = "operator"
	RolePatient   Role = "patient"
)

// Sex represents biological sex for baseline lookups.
type Sex string

const (
	SexMale   Sex = "male"
	SexFemale Sex = "female"
)

// MetricType represents a biometric measurement type.
type MetricType string

const (
	MetricHeartRate        MetricType = "heart_rate"
	MetricSpO2             MetricType = "spo2"
	MetricStressLevel      MetricType = "stress_level"
	MetricSleepDuration    MetricType = "sleep_duration"
	MetricSkinTemperature  MetricType = "skin_temperature"
	MetricMovementScore    MetricType = "movement_score"
	MetricSleepStage       MetricType = "sleep_stage"
	MetricRespiration      MetricType = "respiration"
	MetricBloodPressureSys MetricType = "blood_pressure_systolic"
	MetricBloodPressureDia MetricType = "blood_pressure_diastolic"
)

// RiskLevel represents a clinical risk classification.
type RiskLevel string

const (
	RiskNormal   RiskLevel = "normal"
	RiskMild     RiskLevel = "mild"     // 5-10%
	RiskHigh     RiskLevel = "high"     // 10-20%
	RiskCritical RiskLevel = "critical" // >20%
)

// AlertType represents the kind of health alert generated.
type AlertType string

const (
	AlertAbnormalHeartRate   AlertType = "abnormal_heart_rate"
	AlertLowOxygen           AlertType = "low_oxygen"
	AlertElevatedTemperature AlertType = "elevated_temperature"
	AlertDisturbedSleep      AlertType = "disturbed_sleep"
	AlertHighRisk            AlertType = "high_risk_health_event"
	AlertCritical            AlertType = "critical_health_event"
)

// AlertAction represents the routing action taken for an alert.
type AlertAction string

const (
	ActionInApp           AlertAction = "in_app_alert"
	ActionDashboard       AlertAction = "dashboard_alert"
	ActionCaregiverNotify AlertAction = "caregiver_notification"
	ActionDoctorNotify    AlertAction = "doctor_notification"
	ActionEmergencyNotify AlertAction = "emergency_notification"
)

// ProtocolState represents the lifecycle state of a doctor protocol.
type ProtocolState string

const (
	ProtocolDraft     ProtocolState = "draft"
	ProtocolApproved  ProtocolState = "approved"
	ProtocolActive    ProtocolState = "active"
	ProtocolSuspended ProtocolState = "suspended"
)

// TriggerOutcome represents what action a triggered protocol takes.
type TriggerOutcome string

const (
	OutcomeNotifyOnly            TriggerOutcome = "notify_only"
	OutcomeSuggestAction         TriggerOutcome = "suggest_action"
	OutcomeRequireManualApproval TriggerOutcome = "require_manual_approval"
	OutcomeSendDispenserCommand  TriggerOutcome = "send_dispenser_command"
)

// CommandStatus represents the lifecycle state of a device command.
type CommandStatus string

const (
	CommandQueued       CommandStatus = "queued"
	CommandPublished    CommandStatus = "published"
	CommandAcknowledged CommandStatus = "acknowledged"
	CommandExecuted     CommandStatus = "executed"
	CommandFailed       CommandStatus = "failed"
	CommandTimeout      CommandStatus = "timeout"
)

// SleepStage represents a named sleep cycle stage.
type SleepStage string

const (
	SleepStageLight SleepStage = "light"
	SleepStageDeep  SleepStage = "deep"
	SleepStageREM   SleepStage = "rem"
	SleepStageAwake SleepStage = "awake"
)

// AgeGroup maps an age to a canonical string used for baseline lookups.
type AgeGroup string

const (
	AgeGroup20s AgeGroup = "20-29"
	AgeGroup30s AgeGroup = "30-39"
	AgeGroup40s AgeGroup = "40-49"
	AgeGroup50s AgeGroup = "50-59"
	AgeGroup60s AgeGroup = "60-69"
	AgeGroup70s AgeGroup = "70-79"
	AgeGroup80p AgeGroup = "80+"
)

// AgeGroupFromAge returns the AgeGroup for a given age.
func AgeGroupFromAge(age int) AgeGroup {
	switch {
	case age < 30:
		return AgeGroup20s
	case age < 40:
		return AgeGroup30s
	case age < 50:
		return AgeGroup40s
	case age < 60:
		return AgeGroup50s
	case age < 70:
		return AgeGroup60s
	case age < 80:
		return AgeGroup70s
	default:
		return AgeGroup80p
	}
}

// ─── Core Models ──────────────────────────────────────────────────────────────

// User represents an authenticated user account.
type User struct {
	ID           uuid.UUID `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Role         Role      `db:"role"`
	IsActive     bool      `db:"is_active"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// UserProfile holds health and personal profile data for a patient.
type UserProfile struct {
	ID                 uuid.UUID  `db:"id"`
	UserID             uuid.UUID  `db:"user_id"`
	FullName           string     `db:"full_name"`
	Age                int        `db:"age"`
	Sex                Sex        `db:"sex"`
	WeightKg           float64    `db:"weight_kg"`
	ExistingConditions []string   `db:"existing_conditions"` // stored as JSONB
	DoctorUserID       *uuid.UUID `db:"doctor_user_id"`
	CaregiverUserID    *uuid.UUID `db:"caregiver_user_id"`
	BaselineProfileID  *uuid.UUID `db:"baseline_profile_id"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// EmergencyContact is a person to notify in a critical event.
type EmergencyContact struct {
	ID           uuid.UUID `db:"id"`
	UserID       uuid.UUID `db:"user_id"`
	Name         string    `db:"name"`
	Relationship string    `db:"relationship"`
	Phone        string    `db:"phone"`
	Email        string    `db:"email"`
	Priority     int       `db:"priority"` // lower = higher priority
	CreatedAt    time.Time `db:"created_at"`
}

// BaselineRange defines normal health ranges for an age group and sex.
type BaselineRange struct {
	ID          uuid.UUID  `db:"id"`
	AgeGroup    AgeGroup   `db:"age_group"`
	Sex         Sex        `db:"sex"`
	MetricType  MetricType `db:"metric_type"`
	MinValue    float64    `db:"min_value"`
	MaxValue    float64    `db:"max_value"`
	NormalValue float64    `db:"normal_value"` // midpoint reference
	Unit        string     `db:"unit"`
	CreatedAt   time.Time  `db:"created_at"`
}

// UserBaselineOverride allows a doctor to adjust a patient's personal baseline.
type UserBaselineOverride struct {
	ID          uuid.UUID  `db:"id"`
	UserID      uuid.UUID  `db:"user_id"`
	MetricType  MetricType `db:"metric_type"`
	MinValue    float64    `db:"min_value"`
	MaxValue    float64    `db:"max_value"`
	NormalValue float64    `db:"normal_value"`
	SetByDoctor uuid.UUID  `db:"set_by_doctor"`
	ValidFrom   time.Time  `db:"valid_from"`
	ValidUntil  *time.Time `db:"valid_until"`
	CreatedAt   time.Time  `db:"created_at"`
}

// VitalEvent is a single biometric reading, normalized into the time-series table.
type VitalEvent struct {
	ID              uuid.UUID  `db:"id"`
	UserID          uuid.UUID  `db:"user_id"`
	DeviceID        string     `db:"device_id"`
	SourceTimestamp time.Time  `db:"source_timestamp"`
	IngestionID     uuid.UUID  `db:"ingestion_id"`
	MetricType      MetricType `db:"metric_type"`
	Value           float64    `db:"value"`
	Unit            string     `db:"unit"`
	CreatedAt       time.Time  `db:"created_at"`
}

// RawIngestionLog stores raw JSON payloads before normalization.
type RawIngestionLog struct {
	ID         uuid.UUID `db:"id"`
	UserID     uuid.UUID `db:"user_id"`
	DeviceID   string    `db:"device_id"`
	RawPayload string    `db:"raw_payload"` // JSONB
	ReceivedAt time.Time `db:"received_at"`
}

// SleepSession represents a single sleep period for a user.
type SleepSession struct {
	ID               uuid.UUID  `db:"id"`
	UserID           uuid.UUID  `db:"user_id"`
	StartTime        time.Time  `db:"start_time"`
	EndTime          *time.Time `db:"end_time"`
	DurationMinutes  int        `db:"duration_minutes"`
	QualityScore     float64    `db:"quality_score"` // 0–100
	DisturbanceCount int        `db:"disturbance_count"`
	IsDisturbed      bool       `db:"is_disturbed"`
	AvgHeartRate     *float64   `db:"avg_heart_rate"`
	AvgSpO2          *float64   `db:"avg_spo2"`
	AvgMovementScore *float64   `db:"avg_movement_score"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
}

// SleepStageEvent stores individual sleep stage windows within a session.
type SleepStageEvent struct {
	ID             uuid.UUID  `db:"id"`
	SleepSessionID uuid.UUID  `db:"sleep_session_id"`
	StageType      SleepStage `db:"stage_type"`
	StartTime      time.Time  `db:"start_time"`
	EndTime        time.Time  `db:"end_time"`
	DurationMins   int        `db:"duration_mins"`
	CreatedAt      time.Time  `db:"created_at"`
}

// RiskEvaluation stores the computed risk for a single vital event.
type RiskEvaluation struct {
	ID             uuid.UUID  `db:"id"`
	UserID         uuid.UUID  `db:"user_id"`
	VitalEventID   uuid.UUID  `db:"vital_event_id"`
	MetricType     MetricType `db:"metric_type"`
	MeasuredValue  float64    `db:"measured_value"`
	NormalValue    float64    `db:"normal_value"`
	Deviation      float64    `db:"deviation"`
	RiskPercentage float64    `db:"risk_percentage"`
	RiskLevel      RiskLevel  `db:"risk_level"`
	EvaluatedAt    time.Time  `db:"evaluated_at"`
	CreatedAt      time.Time  `db:"created_at"`
}

// Alert represents a health alert generated by the risk or sleep engine.
type Alert struct {
	ID             uuid.UUID   `db:"id"`
	UserID         uuid.UUID   `db:"user_id"`
	AlertType      AlertType   `db:"alert_type"`
	RiskLevel      RiskLevel   `db:"risk_level"`
	Message        string      `db:"message"`
	MetricType     *MetricType `db:"metric_type"`
	MetricValue    *float64    `db:"metric_value"`
	IsAcknowledged bool        `db:"is_acknowledged"`
	AcknowledgedBy *uuid.UUID  `db:"acknowledged_by"`
	AcknowledgedAt *time.Time  `db:"acknowledged_at"`
	CreatedAt      time.Time   `db:"created_at"`
}

// DoctorProtocol is a doctor-configured medication / response workflow.
type DoctorProtocol struct {
	ID             uuid.UUID      `db:"id"`
	PatientID      uuid.UUID      `db:"patient_id"`
	DoctorID       uuid.UUID      `db:"doctor_id"`
	Name           string         `db:"name"`
	Description    string         `db:"description"`
	State          ProtocolState  `db:"state"`
	TriggerOutcome TriggerOutcome `db:"trigger_outcome"`
	DoctorApproved bool           `db:"doctor_approved"`
	IsLocked       bool           `db:"is_locked"`
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

// ProtocolRule defines a condition within a doctor protocol.
type ProtocolRule struct {
	ID             uuid.UUID  `db:"id"`
	ProtocolID     uuid.UUID  `db:"protocol_id"`
	MetricType     MetricType `db:"metric_type"`
	Operator       string     `db:"operator"` // gt, lt, gte, lte, eq
	ThresholdValue float64    `db:"threshold_value"`
	RiskLevel      RiskLevel  `db:"risk_level"` // minimum risk to trigger
	CreatedAt      time.Time  `db:"created_at"`
}

// DispenserDevice represents a registered IoT pill dispenser.
type DispenserDevice struct {
	ID          uuid.UUID  `db:"id"`
	PatientID   uuid.UUID  `db:"patient_id"`
	DeviceCode  string     `db:"device_code"` // MQTT identifier
	Model       string     `db:"model"`
	FirmwareVer string     `db:"firmware_ver"`
	IsOnline    bool       `db:"is_online"`
	LastSeenAt  *time.Time `db:"last_seen_at"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

// DeviceCommand is an issued command to a dispenser device.
type DeviceCommand struct {
	ID             uuid.UUID     `db:"id"`
	DeviceID       uuid.UUID     `db:"device_id"`
	ProtocolID     *uuid.UUID    `db:"protocol_id"`
	IssuedBy       uuid.UUID     `db:"issued_by"`
	CommandType    string        `db:"command_type"` // dispense | hold | reset
	Payload        string        `db:"payload"`      // JSONB
	Status         CommandStatus `db:"status"`
	PublishedAt    *time.Time    `db:"published_at"`
	AcknowledgedAt *time.Time    `db:"acknowledged_at"`
	ExecutedAt     *time.Time    `db:"executed_at"`
	TimeoutAt      *time.Time    `db:"timeout_at"`
	CreatedAt      time.Time     `db:"created_at"`
	UpdatedAt      time.Time     `db:"updated_at"`
}

// DeviceAcknowledgement stores the raw ACK payload from a device.
type DeviceAcknowledgement struct {
	ID              uuid.UUID `db:"id"`
	DeviceCommandID uuid.UUID `db:"device_command_id"`
	DeviceCode      string    `db:"device_code"`
	Status          string    `db:"status"`
	Message         string    `db:"message"`
	ReceivedAt      time.Time `db:"received_at"`
}

// Notification records every outbound notification sent.
type Notification struct {
	ID          uuid.UUID  `db:"id"`
	UserID      uuid.UUID  `db:"user_id"`      // patient
	RecipientID uuid.UUID  `db:"recipient_id"` // who received it
	Channel     string     `db:"channel"`      // sms | email | push | webhook
	Subject     string     `db:"subject"`
	Body        string     `db:"body"`
	Status      string     `db:"status"` // sent | failed | pending
	SentAt      *time.Time `db:"sent_at"`
	FailedAt    *time.Time `db:"failed_at"`
	ErrorMsg    string     `db:"error_msg"`
	CreatedAt   time.Time  `db:"created_at"`
}

// AuditLog records every clinical / safety decision for compliance.
type AuditLog struct {
	ID         uuid.UUID  `db:"id"`
	ActorID    *uuid.UUID `db:"actor_id"`
	ActorRole  *Role      `db:"actor_role"`
	UserID     *uuid.UUID `db:"user_id"` // patient affected
	Action     string     `db:"action"`  // e.g. "protocol.activated", "alert.created"
	EntityType string     `db:"entity_type"`
	EntityID   *uuid.UUID `db:"entity_id"`
	Detail     string     `db:"detail"` // JSONB
	IPAddress  string     `db:"ip_address"`
	CreatedAt  time.Time  `db:"created_at"`
}
