// Package middleware – audit logging for all clinical/safety decisions.
package middleware

import (
	"context"
	"encoding/json"
	"time"

	"smartbed/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// AuditLogger provides methods to record clinical and security events.
type AuditLogger struct {
	db *sqlx.DB
}

// NewAuditLogger creates a new AuditLogger backed by the given DB.
func NewAuditLogger(db *sqlx.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// Log records an audit event. It is intentionally non-blocking on failure.
func (a *AuditLogger) Log(ctx context.Context, entry domain.AuditLog) {
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now().UTC()

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, actor_id, actor_role, user_id, action, entity_type, entity_id, detail, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		entry.ID, entry.ActorID, entry.ActorRole, entry.UserID,
		entry.Action, entry.EntityType, entry.EntityID,
		entry.Detail, entry.IPAddress, entry.CreatedAt,
	)
	if err != nil {
		log.Error().Err(err).Str("action", entry.Action).Msg("audit log write failed")
	}
}

// LogFromGin is a convenience wrapper that extracts user info from the Gin context.
func (a *AuditLogger) LogFromGin(c *gin.Context, action, entityType string, entityID *uuid.UUID, detail interface{}) {
	actorID, _ := GetCurrentUserID(c)
	actorRole, _ := GetCurrentUserRole(c)

	var detailJSON string
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			detailJSON = string(b)
		}
	}

	entry := domain.AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Detail:     detailJSON,
		IPAddress:  c.ClientIP(),
	}
	if actorID != uuid.Nil {
		entry.ActorID = &actorID
	}
	if actorRole != "" {
		entry.ActorRole = &actorRole
	}

	go a.Log(context.Background(), entry)
}
