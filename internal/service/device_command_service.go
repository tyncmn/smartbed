// Package service – IoT Device Command Service.
// Issues, tracks, and acknowledges commands to dispenser devices over MQTT.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smartbed/internal/domain"
	mqttclient "smartbed/internal/mqtt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// DeviceCommandService manages the full lifecycle of device commands.
type DeviceCommandService struct {
	db         *sqlx.DB
	mqtt       *mqttclient.Client
	ackTimeout time.Duration
}

// NewDeviceCommandService creates a new DeviceCommandService.
func NewDeviceCommandService(db *sqlx.DB, mqtt *mqttclient.Client, ackTimeoutSec int) *DeviceCommandService {
	return &DeviceCommandService{
		db:         db,
		mqtt:       mqtt,
		ackTimeout: time.Duration(ackTimeoutSec) * time.Second,
	}
}

// IssueCommand creates a command, publishes it to MQTT, and starts ACK tracking.
func (s *DeviceCommandService) IssueCommand(
	ctx context.Context,
	deviceID uuid.UUID,
	protocolID *uuid.UUID,
	issuedBy uuid.UUID,
	commandType string,
	payload interface{},
) (*domain.DeviceCommand, error) {
	// Verify device is online
	var device domain.DispenserDevice
	if err := s.db.GetContext(ctx, &device, `SELECT * FROM dispenser_devices WHERE id=$1`, deviceID); err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	if !device.IsOnline {
		return nil, fmt.Errorf("device %s is not online", device.DeviceCode)
	}

	payloadJSON, _ := json.Marshal(payload)
	now := time.Now().UTC()
	timeoutAt := now.Add(s.ackTimeout)

	cmd := &domain.DeviceCommand{
		ID:          uuid.New(),
		DeviceID:    deviceID,
		ProtocolID:  protocolID,
		IssuedBy:    issuedBy,
		CommandType: commandType,
		Payload:     string(payloadJSON),
		Status:      domain.CommandQueued,
		TimeoutAt:   &timeoutAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Persist in queued state
	if err := s.persistCommand(ctx, cmd); err != nil {
		return nil, fmt.Errorf("persist command: %w", err)
	}

	// Publish to MQTT
	topic := mqttclient.CommandTopic(device.DeviceCode)
	mqttPayload := map[string]interface{}{
		"command_id":   cmd.ID,
		"command_type": commandType,
		"payload":      payload,
		"issued_at":    now,
	}
	if err := s.mqtt.Publish(topic, mqttPayload); err != nil {
		// Mark as failed if publish fails
		_ = s.updateStatus(ctx, cmd.ID, domain.CommandFailed, nil)
		return nil, fmt.Errorf("mqtt publish: %w", err)
	}

	// Update to published
	publishedAt := time.Now().UTC()
	_ = s.updateStatus(ctx, cmd.ID, domain.CommandPublished, &publishedAt)

	log.Info().
		Str("command_id", cmd.ID.String()).
		Str("device", device.DeviceCode).
		Str("topic", topic).
		Msg("device command published")

	return cmd, nil
}

// HandleACK processes an incoming ACK from the MQTT broker and updates command lifecycle.
func (s *DeviceCommandService) HandleACK(ctx context.Context, ack mqttclient.ACKPayload) {
	cmdID, err := uuid.Parse(ack.CommandID)
	if err != nil {
		log.Warn().Str("command_id", ack.CommandID).Msg("invalid command_id in ACK")
		return
	}

	// Persist the acknowledgement record
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO device_acknowledgements (id, device_command_id, device_code, status, message, received_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		uuid.New(), cmdID, ack.DeviceID, ack.Status, ack.Message, ack.Timestamp,
	)
	if err != nil {
		log.Error().Err(err).Msg("persist ACK failed")
	}

	// Update command status
	now := time.Now().UTC()
	var newStatus domain.CommandStatus
	switch ack.Status {
	case "executed":
		newStatus = domain.CommandExecuted
	case "failed":
		newStatus = domain.CommandFailed
	default:
		newStatus = domain.CommandAcknowledged
	}
	_ = s.updateStatus(ctx, cmdID, newStatus, &now)
	log.Info().Str("command_id", cmdID.String()).Str("status", string(newStatus)).Msg("device command ACK processed")
}

// GetDeviceStatus returns current status info for a device and its latest command.
func (s *DeviceCommandService) GetDeviceStatus(ctx context.Context, deviceID uuid.UUID) (*domain.DispenserDevice, *domain.DeviceCommand, error) {
	var device domain.DispenserDevice
	if err := s.db.GetContext(ctx, &device, `SELECT * FROM dispenser_devices WHERE id=$1`, deviceID); err != nil {
		return nil, nil, fmt.Errorf("device not found: %w", err)
	}
	var latestCmd domain.DeviceCommand
	err := s.db.GetContext(ctx, &latestCmd, `
		SELECT * FROM device_commands WHERE device_id=$1 ORDER BY created_at DESC LIMIT 1`, deviceID)
	if err != nil {
		return &device, nil, nil // device exists, just no commands yet
	}
	return &device, &latestCmd, nil
}

// SweepTimedOutCommands sets status=timeout for commands past their timeout_at.
// Called by the background worker on a schedule.
func (s *DeviceCommandService) SweepTimedOutCommands(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		UPDATE device_commands
		SET status='timeout', updated_at=now()
		WHERE status IN ('queued','published')
		  AND timeout_at IS NOT NULL
		  AND timeout_at < now()`,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func (s *DeviceCommandService) persistCommand(ctx context.Context, cmd *domain.DeviceCommand) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO device_commands
			(id, device_id, protocol_id, issued_by, command_type, payload, status, timeout_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		cmd.ID, cmd.DeviceID, cmd.ProtocolID, cmd.IssuedBy,
		cmd.CommandType, cmd.Payload, string(cmd.Status),
		cmd.TimeoutAt, cmd.CreatedAt, cmd.UpdatedAt,
	)
	return err
}

func (s *DeviceCommandService) updateStatus(ctx context.Context, cmdID uuid.UUID, status domain.CommandStatus, ts *time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE device_commands SET status=$1, updated_at=now(),
		published_at = CASE WHEN $2 = 'published' THEN $3 ELSE published_at END,
		acknowledged_at = CASE WHEN $2 IN ('acknowledged','executed','failed') THEN $3 ELSE acknowledged_at END,
		executed_at = CASE WHEN $2 = 'executed' THEN $3 ELSE executed_at END
		WHERE id = $4`,
		string(status), string(status), ts, cmdID,
	)
	return err
}
