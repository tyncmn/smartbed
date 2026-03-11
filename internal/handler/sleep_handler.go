// Package handler – Sleep Dashboard Handler.
// Serves GET /api/v1/users/:id/sleep-dashboard
package handler

import (
	"net/http"
	"strconv"

	"smartbed/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SleepHandler serves sleep dashboard endpoints.
type SleepHandler struct {
	dashboardSvc *service.SleepDashboardService
}

// NewSleepHandler creates a new SleepHandler.
func NewSleepHandler(dashboardSvc *service.SleepDashboardService) *SleepHandler {
	return &SleepHandler{dashboardSvc: dashboardSvc}
}

// GetSleepDashboard handles GET /api/v1/users/:id/sleep-dashboard?days=7
// Returns aggregated sleep analytics, health alerts, AI analysis, and predictions.
func (h *SleepHandler) GetSleepDashboard(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	days := 7
	if dStr := c.Query("days"); dStr != "" {
		if d, err := strconv.Atoi(dStr); err == nil && d > 0 {
			days = d
		}
	}

	resp, err := h.dashboardSvc.GetDashboard(c.Request.Context(), userID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build dashboard"})
		return
	}

	c.JSON(http.StatusOK, resp)
}
