// Package middleware – JWT auth and RBAC for SmartBed.
package middleware

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"strings"
	"time"

	"smartbed/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims are the JWT payload fields used by SmartBed.
type Claims struct {
	UserID uuid.UUID   `json:"uid"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// JWTService handles token generation and validation.
type JWTService struct {
	privateKey         *rsa.PrivateKey
	publicKey          *rsa.PublicKey
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
}

// NewJWTService creates a JWTService from PEM key bytes.
func NewJWTService(privateKeyPEM, publicKeyPEM []byte, accessMins, refreshDays int) (*JWTService, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse RSA private key: %w", err)
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse RSA public key: %w", err)
	}
	return &JWTService{
		privateKey:         privateKey,
		publicKey:          publicKey,
		accessTokenExpiry:  time.Duration(accessMins) * time.Minute,
		refreshTokenExpiry: time.Duration(refreshDays) * 24 * time.Hour,
	}, nil
}

// GenerateAccessToken creates a signed access JWT.
func (s *JWTService) GenerateAccessToken(userID uuid.UUID, role domain.Role) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "smartbed",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// GenerateRefreshToken creates a long-lived signed refresh JWT.
func (s *JWTService) GenerateRefreshToken(userID uuid.UUID, role domain.Role) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "smartbed-refresh",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// ParseToken validates a token string and returns the embedded claims.
func (s *JWTService) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token claims")
}

// Authenticate is a Gin middleware that validates the Bearer JWT.
func (s *JWTService) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		claims, err := s.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// RequireRoles is a Gin middleware that enforces role-based access.
func RequireRoles(allowed ...domain.Role) gin.HandlerFunc {
	allowed_set := make(map[domain.Role]struct{}, len(allowed))
	for _, r := range allowed {
		allowed_set[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
			return
		}
		r, ok := role.(domain.Role)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "role type error"})
			return
		}
		if _, allowed := allowed_set[r]; !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}

// GetCurrentUserID extracts the authenticated user's UUID from Gin context.
func GetCurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

// GetCurrentUserRole extracts the authenticated user's role from Gin context.
func GetCurrentUserRole(c *gin.Context) (domain.Role, bool) {
	v, exists := c.Get("user_role")
	if !exists {
		return "", false
	}
	r, ok := v.(domain.Role)
	return r, ok
}
