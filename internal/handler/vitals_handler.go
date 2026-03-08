// Package handler – Vitals ingestion handler.
package handler

import (
	"net/http"

	"smartbed/internal/service"

	"github.com/gin-gonic/gin"
)

// VitalsHandler handles biometric data ingestion.
type VitalsHandler struct {
	ingestionSvc *service.IngestionService
}

// NewVitalsHandler creates a new VitalsHandler.
func NewVitalsHandler(ingestionSvc *service.IngestionService) *VitalsHandler {
	return &VitalsHandler{ingestionSvc: ingestionSvc}
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
