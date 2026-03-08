// Package handler – Dashboard and alert handlers.
package handler

import (
	"net/http"
	"strconv"

	"smartbed/internal/middleware"
	"smartbed/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ─── Alert Handler ─────────────────────────────────────────────────────────────

// AlertHandler handles alert listing and acknowledgement.
type AlertHandler struct {
	alertSvc *service.AlertService
}

// NewAlertHandler creates a new AlertHandler.
func NewAlertHandler(alertSvc *service.AlertService) *AlertHandler {
	return &AlertHandler{alertSvc: alertSvc}
}

// GetAlerts godoc
// @Summary      List alerts for a user
// @Tags         alerts
// @Produce      json
// @Param        user_id query string true "Patient user ID"
// @Param        unread  query bool   false "Filter unread only"
// @Success      200  {array} domain.Alert
// @Router       /api/v1/alerts [get]
func (h *AlertHandler) GetAlerts(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	onlyUnread := c.Query("unread") == "true"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	alerts, err := h.alertSvc.GetAlerts(c.Request.Context(), userID, onlyUnread, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts, "count": len(alerts)})
}

// AcknowledgeAlert godoc
// @Summary Acknowledge an alert
// @Tags    alerts
// @Param   id path string true "Alert ID"
// @Success 200
// @Router  /api/v1/alerts/:id/acknowledge [put]
func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	alertID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}
	actorID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	if err := h.alertSvc.AcknowledgeAlert(c.Request.Context(), alertID, actorID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "alert acknowledged"})
}

// ─── Dashboard Handler ──────────────────────────────────────────────────────────

// DashboardHandler serves read-only dashboard data.
type DashboardHandler struct {
	db interface {
		GetContext(ctx interface{}, dest, query interface{}, args ...interface{}) error
	}
	sleepSvc *service.SleepAnalyticsService
	alertSvc *service.AlertService
}

// DashboardHandlerWithDeps creates a dashboard handler using real services.
func DashboardHandlerWithDeps(sleepSvc *service.SleepAnalyticsService, alertSvc *service.AlertService) *DashboardHandler {
	return &DashboardHandler{sleepSvc: sleepSvc, alertSvc: alertSvc}
}

// GetUserCurrentStatus godoc
// @Summary      Get current health status for a user
// @Tags         dashboard
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200  {object} map[string]interface{}
// @Router       /api/v1/users/{id}/current-status [get]
func (h *DashboardHandler) GetUserCurrentStatus(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	// Return unacknowledged alerts count + 7d sleep summary as status proxy
	alerts, err := h.alertSvc.GetAlerts(c.Request.Context(), userID, true, 10, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	summary, _ := h.sleepSvc.GetSleepSummary(c.Request.Context(), userID, 7)

	c.JSON(http.StatusOK, gin.H{
		"user_id":          userID,
		"unread_alerts":    len(alerts),
		"recent_alerts":    alerts,
		"sleep_7d_summary": summary,
	})
}

// GetSleepSummary godoc
// @Summary      Get sleep summary for a user
// @Tags         dashboard
// @Produce      json
// @Param        id   path string true  "User ID"
// @Param        days query int  false "Days (7 or 30, default 7)"
// @Success      200  {object} service.SleepSummary
// @Router       /api/v1/users/{id}/sleep-summary [get]
func (h *DashboardHandler) GetSleepSummary(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days != 7 && days != 30 {
		days = 7
	}
	summary, err := h.sleepSvc.GetSleepSummary(c.Request.Context(), userID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}
