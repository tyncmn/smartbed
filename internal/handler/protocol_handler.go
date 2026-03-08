// Package handler – Protocol and Device Command handlers.
package handler

import (
	"net/http"

	"smartbed/internal/domain"
	"smartbed/internal/middleware"
	"smartbed/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ─── Protocol Handler ──────────────────────────────────────────────────────────

// ProtocolHandler handles doctor protocol management.
type ProtocolHandler struct {
	protocolSvc *service.ProtocolService
	auditLogger *middleware.AuditLogger
}

// NewProtocolHandler creates a new ProtocolHandler.
func NewProtocolHandler(svc *service.ProtocolService, audit *middleware.AuditLogger) *ProtocolHandler {
	return &ProtocolHandler{protocolSvc: svc, auditLogger: audit}
}

// CreateProtocolRequest is the DTO for protocol creation.
type CreateProtocolRequest struct {
	PatientID      string `json:"patient_id"      validate:"required,uuid"`
	Name           string `json:"name"             validate:"required"`
	Description    string `json:"description"`
	TriggerOutcome string `json:"trigger_outcome"  validate:"required,oneof=notify_only suggest_action require_manual_approval send_dispenser_command"`
	Rules          []struct {
		MetricType     string  `json:"metric_type"      validate:"required"`
		Operator       string  `json:"operator"         validate:"required,oneof=gt lt gte lte eq"`
		ThresholdValue float64 `json:"threshold_value"  validate:"required"`
		RiskLevel      string  `json:"risk_level"       validate:"required"`
	} `json:"rules"`
}

// CreateProtocol godoc
// @Summary      Create a doctor protocol
// @Tags         protocols
// @Accept       json
// @Produce      json
// @Param        body body CreateProtocolRequest true "Protocol definition"
// @Success      201  {object} domain.DoctorProtocol
// @Router       /api/v1/protocols [post]
func (h *ProtocolHandler) CreateProtocol(c *gin.Context) {
	var req CreateProtocolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	doctorID, _ := middleware.GetCurrentUserID(c)
	patientID, _ := uuid.Parse(req.PatientID)

	p := domain.DoctorProtocol{
		PatientID:      patientID,
		DoctorID:       doctorID,
		Name:           req.Name,
		Description:    req.Description,
		TriggerOutcome: domain.TriggerOutcome(req.TriggerOutcome),
	}
	var rules []domain.ProtocolRule
	for _, r := range req.Rules {
		rules = append(rules, domain.ProtocolRule{
			MetricType:     domain.MetricType(r.MetricType),
			Operator:       r.Operator,
			ThresholdValue: r.ThresholdValue,
			RiskLevel:      domain.RiskLevel(r.RiskLevel),
		})
	}

	created, err := h.protocolSvc.CreateProtocol(c.Request.Context(), p, rules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.auditLogger.LogFromGin(c, "protocol.created", "doctor_protocols", &created.ID, req)
	c.JSON(http.StatusCreated, created)
}

// UpdateProtocolState godoc
// @Summary      Transition a protocol's state
// @Tags         protocols
// @Param        id    path string true "Protocol ID"
// @Param        state body map[string]string true "Target state"
// @Success      200
// @Router       /api/v1/protocols/{id}/state [put]
func (h *ProtocolHandler) UpdateProtocolState(c *gin.Context) {
	protocolID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid protocol id"})
		return
	}
	var body struct {
		State string `json:"state" validate:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, _ := middleware.GetCurrentUserID(c)
	if err := h.protocolSvc.TransitionState(c.Request.Context(), protocolID, actorID, domain.ProtocolState(body.State)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "state updated"})
}

// ─── Device Command Handler ─────────────────────────────────────────────────────

// DeviceHandler handles dispenser device commands and status.
type DeviceHandler struct {
	deviceSvc   *service.DeviceCommandService
	auditLogger *middleware.AuditLogger
}

// NewDeviceHandler creates a new DeviceHandler.
func NewDeviceHandler(svc *service.DeviceCommandService, audit *middleware.AuditLogger) *DeviceHandler {
	return &DeviceHandler{deviceSvc: svc, auditLogger: audit}
}

// ExecuteCommand godoc
// @Summary      Issue a command to a dispenser device
// @Tags         devices
// @Accept       json
// @Produce      json
// @Param        deviceId path string true "Device ID (UUID)"
// @Param        body body map[string]interface{} true "Command payload"
// @Success      202  {object} domain.DeviceCommand
// @Router       /api/v1/device-commands/{deviceId}/execute [post]
func (h *DeviceHandler) ExecuteCommand(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}
	var body struct {
		CommandType string      `json:"command_type" validate:"required"`
		Payload     interface{} `json:"payload"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	actorID, _ := middleware.GetCurrentUserID(c)
	cmd, err := h.deviceSvc.IssueCommand(c.Request.Context(), deviceID, nil, actorID, body.CommandType, body.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.auditLogger.LogFromGin(c, "device_command.issued", "device_commands", &cmd.ID, body)
	c.JSON(http.StatusAccepted, cmd)
}

// GetDeviceStatus godoc
// @Summary      Get device status and latest command
// @Tags         devices
// @Produce      json
// @Param        deviceId path string true "Device ID (UUID)"
// @Success      200  {object} map[string]interface{}
// @Router       /api/v1/devices/{deviceId}/status [get]
func (h *DeviceHandler) GetDeviceStatus(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}
	device, latestCmd, err := h.deviceSvc.GetDeviceStatus(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"device":         device,
		"latest_command": latestCmd,
	})
}
