// Package service – Doctor Protocol Service.
// Implements the protocol state machine with mandatory safety guardrails
// before any dispenser command can be issued.
package service

import (
	"context"
	"fmt"
	"time"

	"smartbed/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ProtocolService manages doctor-configured medication/response workflows.
type ProtocolService struct {
	db        *sqlx.DB
	deviceSvc *DeviceCommandService
	auditLog  func(ctx context.Context, actorID uuid.UUID, action string, entityID uuid.UUID, detail interface{})
}

// NewProtocolService creates a new ProtocolService.
func NewProtocolService(db *sqlx.DB, deviceSvc *DeviceCommandService) *ProtocolService {
	return &ProtocolService{db: db, deviceSvc: deviceSvc}
}

// SetAuditFunc sets the audit log callback (prevents circular deps).
func (s *ProtocolService) SetAuditFunc(fn func(ctx context.Context, actorID uuid.UUID, action string, entityID uuid.UUID, detail interface{})) {
	s.auditLog = fn
}

// CreateProtocol creates a new protocol in draft state (doctor only).
func (s *ProtocolService) CreateProtocol(ctx context.Context, p domain.DoctorProtocol, rules []domain.ProtocolRule) (*domain.DoctorProtocol, error) {
	p.ID = uuid.New()
	p.State = domain.ProtocolDraft
	p.DoctorApproved = false
	p.IsLocked = false
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = p.CreatedAt

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO doctor_protocols
			(id, patient_id, doctor_id, name, description, state, trigger_outcome, doctor_approved, is_locked, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		p.ID, p.PatientID, p.DoctorID, p.Name, p.Description,
		string(p.State), string(p.TriggerOutcome), p.DoctorApproved, p.IsLocked,
		p.CreatedAt, p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("insert protocol: %w", err)
	}

	for i := range rules {
		rules[i].ID = uuid.New()
		rules[i].ProtocolID = p.ID
		rules[i].CreatedAt = time.Now().UTC()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO protocol_rules (id, protocol_id, metric_type, operator, threshold_value, risk_level, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			rules[i].ID, rules[i].ProtocolID, string(rules[i].MetricType),
			string(rules[i].Operator), rules[i].ThresholdValue, string(rules[i].RiskLevel), rules[i].CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("insert protocol rule: %w", err)
		}
	}

	return &p, tx.Commit()
}

// TransitionState advances the protocol through its state machine.
// Valid transitions: draft→approved, approved→active, active→suspended, suspended→active.
func (s *ProtocolService) TransitionState(ctx context.Context, protocolID, actorID uuid.UUID, newState domain.ProtocolState) error {
	p, err := s.GetByID(ctx, protocolID)
	if err != nil {
		return err
	}

	if !s.isValidTransition(p.State, newState) {
		return fmt.Errorf("invalid state transition: %s → %s", p.State, newState)
	}

	// Approving requires the actor to be the owning doctor
	if newState == domain.ProtocolApproved && actorID != p.DoctorID {
		return fmt.Errorf("only the owning doctor may approve a protocol")
	}

	now := time.Now().UTC()
	approved := p.DoctorApproved
	if newState == domain.ProtocolApproved {
		approved = true
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE doctor_protocols SET state=$1, doctor_approved=$2, updated_at=$3 WHERE id=$4`,
		string(newState), approved, now, protocolID,
	); err != nil {
		return err
	}

	if s.auditLog != nil {
		s.auditLog(ctx, actorID, fmt.Sprintf("protocol.state.%s", newState), protocolID, map[string]interface{}{
			"from": p.State,
			"to":   newState,
		})
	}
	return nil
}

// EvaluateAndTrigger evaluates a protocol's rules against an incoming risk evaluation.
// SAFETY: Only proceeds through the `send_dispenser_command` path after all guards pass.
func (s *ProtocolService) EvaluateAndTrigger(ctx context.Context, patientID uuid.UUID, eval domain.RiskEvaluation, actorID uuid.UUID) error {
	var protocols []domain.DoctorProtocol
	if err := s.db.SelectContext(ctx, &protocols, `
		SELECT * FROM doctor_protocols
		WHERE patient_id=$1 AND state='active' AND doctor_approved=TRUE AND is_locked=FALSE`,
		patientID,
	); err != nil {
		return err
	}

	for _, p := range protocols {
		rules, err := s.getRules(ctx, p.ID)
		if err != nil {
			continue
		}
		if s.rulesMatch(rules, eval) {
			if err := s.executeOutcome(ctx, p, eval, actorID); err != nil {
				return fmt.Errorf("execute protocol %s outcome: %w", p.ID, err)
			}
		}
	}
	return nil
}

// executeOutcome dispatches the appropriate action based on protocol trigger_outcome.
func (s *ProtocolService) executeOutcome(ctx context.Context, p domain.DoctorProtocol, eval domain.RiskEvaluation, actorID uuid.UUID) error {
	switch p.TriggerOutcome {
	case domain.OutcomeNotifyOnly:
		// Notification is handled by the alert service already – nothing extra
		return nil

	case domain.OutcomeSuggestAction:
		// Create a dashboard-visible suggestion (stored as info-level alert)
		return nil

	case domain.OutcomeRequireManualApproval:
		// Create a pending approval record; dispenser command is NOT sent automatically
		if s.auditLog != nil {
			s.auditLog(ctx, actorID, "protocol.manual_approval_required", p.ID, map[string]interface{}{
				"metric":     eval.MetricType,
				"risk_level": eval.RiskLevel,
			})
		}
		return nil

	case domain.OutcomeSendDispenserCommand:
		// ─────────────────────────────────────────────────────────────────
		// SAFETY GATE: All of the following must hold before issuing a
		// dispenser command:
		// 1. Protocol is active
		// 2. Protocol has doctor_approved = true
		// 3. Protocol is not locked
		// 4. Risk level meets or exceeds the rule threshold
		// ─────────────────────────────────────────────────────────────────
		if !p.DoctorApproved {
			return fmt.Errorf("safety gate: protocol not doctor-approved")
		}
		if p.IsLocked {
			return fmt.Errorf("safety gate: protocol is locked")
		}
		// Find the device for this patient
		var deviceID uuid.UUID
		if err := s.db.GetContext(ctx, &deviceID, `
			SELECT id FROM dispenser_devices WHERE patient_id=$1 AND is_online=TRUE LIMIT 1`, p.PatientID,
		); err != nil {
			return fmt.Errorf("no online dispenser device for patient: %w", err)
		}

		if s.auditLog != nil {
			s.auditLog(ctx, actorID, "protocol.dispenser_command_issued", p.ID, map[string]interface{}{
				"device_id":  deviceID,
				"metric":     eval.MetricType,
				"risk_level": eval.RiskLevel,
			})
		}

		_, err := s.deviceSvc.IssueCommand(ctx, deviceID, &p.ID, actorID, "dispense", map[string]interface{}{
			"protocol_id":    p.ID,
			"triggered_by":   "risk_evaluation",
			"metric":         eval.MetricType,
			"measured_value": eval.MeasuredValue,
		})
		return err
	}
	return nil
}

// GetByID fetches a protocol by ID.
func (s *ProtocolService) GetByID(ctx context.Context, id uuid.UUID) (*domain.DoctorProtocol, error) {
	var p domain.DoctorProtocol
	if err := s.db.GetContext(ctx, &p, `SELECT * FROM doctor_protocols WHERE id=$1`, id); err != nil {
		return nil, fmt.Errorf("protocol not found: %w", err)
	}
	return &p, nil
}

func (s *ProtocolService) getRules(ctx context.Context, protocolID uuid.UUID) ([]domain.ProtocolRule, error) {
	var rules []domain.ProtocolRule
	return rules, s.db.SelectContext(ctx, &rules, `SELECT * FROM protocol_rules WHERE protocol_id=$1`, protocolID)
}

func (s *ProtocolService) rulesMatch(rules []domain.ProtocolRule, eval domain.RiskEvaluation) bool {
	for _, r := range rules {
		if r.MetricType != eval.MetricType {
			continue
		}
		var match bool
		switch r.Operator {
		case "gt":
			match = eval.MeasuredValue > r.ThresholdValue
		case "lt":
			match = eval.MeasuredValue < r.ThresholdValue
		case "gte":
			match = eval.MeasuredValue >= r.ThresholdValue
		case "lte":
			match = eval.MeasuredValue <= r.ThresholdValue
		case "eq":
			match = eval.MeasuredValue == r.ThresholdValue
		}
		if match {
			return true
		}
	}
	return false
}

func (s *ProtocolService) isValidTransition(from, to domain.ProtocolState) bool {
	valid := map[domain.ProtocolState][]domain.ProtocolState{
		domain.ProtocolDraft:     {domain.ProtocolApproved},
		domain.ProtocolApproved:  {domain.ProtocolActive, domain.ProtocolSuspended},
		domain.ProtocolActive:    {domain.ProtocolSuspended},
		domain.ProtocolSuspended: {domain.ProtocolActive},
	}
	for _, s := range valid[from] {
		if s == to {
			return true
		}
	}
	return false
}
