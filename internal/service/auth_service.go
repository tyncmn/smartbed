// Package service – Auth Service.
// Handles user authentication and JWT token management.
package service

import (
	"context"
	"fmt"
	"time"

	"smartbed/internal/domain"
	"smartbed/internal/middleware"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles user authentication.
type AuthService struct {
	db         *sqlx.DB
	jwtService *middleware.JWTService
}

// NewAuthService creates a new AuthService.
func NewAuthService(db *sqlx.DB, jwtService *middleware.JWTService) *AuthService {
	return &AuthService{db: db, jwtService: jwtService}
}

// LoginRequest is the DTO for login.
type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// TokenPair holds both access and refresh tokens.
type TokenPair struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresAt    time.Time   `json:"expires_at"`
	UserID       uuid.UUID   `json:"user_id"`
	Role         domain.Role `json:"role"`
}

// Login validates credentials and returns a JWT token pair.
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*TokenPair, error) {
	var user domain.User
	if err := s.db.GetContext(ctx, &user, `
		SELECT id, email, password_hash, role, is_active FROM users WHERE email=$1`, req.Email,
	); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is deactivated")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID, user.Role)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(time.Hour),
		UserID:       user.ID,
		Role:         user.Role,
	}, nil
}

// HashPassword creates a bcrypt hash of a plaintext password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CreateUser creates a new user account (admin only).
func (s *AuthService) CreateUser(ctx context.Context, email, password string, role domain.Role) (*domain.User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, role, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		user.ID, user.Email, user.PasswordHash, string(user.Role), user.IsActive, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &user, nil
}
