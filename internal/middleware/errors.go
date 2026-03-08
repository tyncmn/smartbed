// Package middleware provides Gin middleware for the SmartBed API.
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// AppError is a structured API error returned to clients.
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *AppError) Error() string { return e.Message }

// Standard errors
var (
	ErrUnauthorized   = &AppError{Code: http.StatusUnauthorized, Message: "unauthorized"}
	ErrForbidden      = &AppError{Code: http.StatusForbidden, Message: "forbidden"}
	ErrNotFound       = &AppError{Code: http.StatusNotFound, Message: "not found"}
	ErrBadRequest     = &AppError{Code: http.StatusBadRequest, Message: "bad request"}
	ErrConflict       = &AppError{Code: http.StatusConflict, Message: "conflict"}
	ErrInternalServer = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
)

// Wrap returns a new AppError with additional detail.
func Wrap(base *AppError, detail string) *AppError {
	return &AppError{Code: base.Code, Message: base.Message, Detail: detail}
}

// ErrorHandler is a centralized Gin error-handling middleware.
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		var appErr *AppError
		if errors.As(err, &appErr) {
			c.JSON(appErr.Code, gin.H{
				"error":  appErr.Message,
				"detail": appErr.Detail,
			})
			return
		}

		// Log unexpected errors and return generic 500
		log.Error().Err(err).Str("path", c.Request.URL.Path).Msg("unhandled error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// RequestLogger logs each incoming request with zerolog.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Str("ip", c.ClientIP()).
			Str("user-agent", c.Request.UserAgent()).
			Msg("request")
	}
}

// Pagination extracts page/limit query params and stores them in context.
func Pagination() gin.HandlerFunc {
	return func(c *gin.Context) {
		page := strings.TrimSpace(c.DefaultQuery("page", "1"))
		limit := strings.TrimSpace(c.DefaultQuery("limit", "50"))
		c.Set("page", page)
		c.Set("limit", limit)
		c.Next()
	}
}
