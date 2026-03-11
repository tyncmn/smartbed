// Package handler – Vitals ingestion handler.
package handler

import (
	"net/http"

	"smartbed/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VitalsHandler handles biometric data ingestion.
type VitalsHandler struct {
	ingestionSvc *service.IngestionService
	sleepSvc     *service.SleepAnalyticsService
}

// NewVitalsHandler creates a new VitalsHandler.
func NewVitalsHandler(ingestionSvc *service.IngestionService, sleepSvc *service.SleepAnalyticsService) *VitalsHandler {
	return &VitalsHandler{ingestionSvc: ingestionSvc, sleepSvc: sleepSvc}
}

// Ingest godoc
// @Summary      Ingest biometric data from Apple Watch / iPhone bridge
// @Tags         vitals
// @Accept       json
// @Produce      json
// @Param        body body service.IngestionPayload true "Biometric payload"
// @Success      202  {object} service.IngestionResult
// @Failure      400  {object} map[string]string
// @Failure      422  {object} map[string]string
// @Router       /api/v1/vitals/ingest [post]
func (h *VitalsHandler) Ingest(c *gin.Context) {
	var payload service.IngestionPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validate.Struct(payload); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	result, err := h.ingestionSvc.Ingest(c.Request.Context(), payload)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, result)
}

// GetLatestVitals godoc
// @Summary      Get latest vital reading per metric for a user (poll every 5s for live display)
// @Tags         vitals
// @Produce      json
// @Param        user_id query string true "Patient user ID"
// @Success      200  {object} service.LatestVitals
// @Router       /api/v1/vitals/latest [get]
func (h *VitalsHandler) GetLatestVitals(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	result, err := h.ingestionSvc.GetLatestVitals(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// GetVitalsSleepSummary godoc
// @Summary      Get aggregated sleep summary for a user
// @Tags         vitals
// @Produce      json
// @Param        user_id query string true  "Patient user ID"
// @Param        days    query int    false "Lookback days (7 or 30, default 7)"
// @Success      200  {object} service.SleepSummary
// @Router       /api/v1/vitals/sleep-summary [get]
func (h *VitalsHandler) GetVitalsSleepSummary(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	days := 7
	if d := c.Query("days"); d == "30" {
		days = 30
	}
	summary, err := h.sleepSvc.GetSleepSummary(c.Request.Context(), userID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}
