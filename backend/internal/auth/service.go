package auth

import (
	"errors"
	"os"
	"time"

	"iafarma/pkg/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication logic
type Service struct {
	userRepo UserRepository
}

// UserRepository interface for user data access
type UserRepository interface {
	GetByEmail(email string) (*models.User, error)
	GetByID(id uuid.UUID) (*models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
	// Password reset methods
	CreatePasswordResetToken(token *models.PasswordResetToken) error
	GetPasswordResetToken(token string) (*models.PasswordResetToken, error)
	MarkPasswordResetTokenAsUsed(tokenID uuid.UUID) error
	InvalidateUserPasswordResetTokens(userID uuid.UUID) error
}

// NewService creates a new auth service
func NewService(userRepo UserRepository) *Service {
	return &Service{
		userRepo: userRepo,
	}
}

// LoginRequest represents login request data
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents login response data
type LoginResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         models.User `json:"user"`
	ExpiresIn    int64       `json:"expires_in"`
}

// TokenClaims represents JWT token claims
type TokenClaims struct {
	UserID   uuid.UUID  `json:"user_id"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	Email    string     `json:"email"`
	Role     string     `json:"role"`
	Type     string     `json:"type"` // access or refresh
	jwt.RegisteredClaims
}

// Login authenticates a user and returns tokens
func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	if !s.verifyPassword(req.Password, user.Password) {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	s.userRepo.Update(user)

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	accessDuration, _ := time.ParseDuration(getEnvOrDefault("JWT_ACCESS_DURATION", "15m"))

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
		ExpiresIn:    int64(accessDuration.Seconds()),
	}, nil
}

// RefreshToken generates new tokens from refresh token
func (s *Service) RefreshToken(tokenString string) (*LoginResponse, error) {
	claims, err := s.validateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.Type != "refresh" {
		return nil, errors.New("invalid token type")
	}

	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("user account is disabled")
	}

	// Generate new tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	accessDuration, _ := time.ParseDuration(getEnvOrDefault("JWT_ACCESS_DURATION", "15m"))

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
		ExpiresIn:    int64(accessDuration.Seconds()),
	}, nil
}

// ValidateToken validates and parses a JWT token
func (s *Service) ValidateToken(tokenString string) (*TokenClaims, error) {
	return s.validateToken(tokenString)
}

// HashPassword hashes a password using Argon2
func (s *Service) HashPassword(password string) (string, error) {
	// Use bcrypt for simplicity in this example
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// UpdateProfile updates user profile information
func (s *Service) UpdateProfile(userID string, req models.UpdateProfileRequest) (*models.User, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Update user fields
	user.Name = req.Name
	user.Email = req.Email
	user.Phone = req.Phone

	if err := s.userRepo.Update(user); err != nil {
		return nil, errors.New("failed to update user profile")
	}

	return user, nil
}

// ChangePassword changes user password
func (s *Service) ChangePassword(userID string, currentPassword, newPassword string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return errors.New("user not found")
	}

	// Verify current password
	if !s.verifyPassword(currentPassword, user.Password) {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	// Update password
	user.Password = hashedPassword
	if err := s.userRepo.Update(user); err != nil {
		return errors.New("failed to update password")
	}

	return nil
}

// generateAccessToken generates an access token
func (s *Service) generateAccessToken(user *models.User) (string, error) {
	duration, err := time.ParseDuration(getEnvOrDefault("JWT_ACCESS_DURATION", "15m"))
	if err != nil {
		duration = 15 * time.Minute
	}

	claims := TokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Role:     user.Role,
		Type:     "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "iafarma",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// generateRefreshToken generates a refresh token
func (s *Service) generateRefreshToken(user *models.User) (string, error) {
	duration, err := time.ParseDuration(getEnvOrDefault("JWT_REFRESH_DURATION", "24h"))
	if err != nil {
		duration = 24 * time.Hour
	}

	claims := TokenClaims{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Role:     user.Role,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "iafarma",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

// validateToken validates and parses a JWT token
func (s *Service) validateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// verifyPassword verifies a password against its hash
func (s *Service) verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// getEnvOrDefault gets an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RequestPasswordReset creates a password reset token and returns it for email sending
func (s *Service) RequestPasswordReset(email string) (*models.PasswordResetToken, error) {
	// Check if user exists
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		// Don't reveal if email exists or not for security
		return nil, errors.New("if the email exists, a reset link has been sent")
	}

	// Invalidate any existing tokens for this user
	if err := s.userRepo.InvalidateUserPasswordResetTokens(user.ID); err != nil {
		return nil, errors.New("failed to process password reset request")
	}

	// Generate new token
	token := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour) // Token expires in 1 hour

	resetToken := &models.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
		IsUsed:    false,
		User:      user,
	}

	// Save token
	if err := s.userRepo.CreatePasswordResetToken(resetToken); err != nil {
		return nil, errors.New("failed to create password reset token")
	}

	return resetToken, nil
}

// ResetPassword resets user password using a valid token
func (s *Service) ResetPassword(token, newPassword string) error {
	// Get and validate token
	resetToken, err := s.userRepo.GetPasswordResetToken(token)
	if err != nil {
		return errors.New("invalid or expired reset token")
	}

	// Check if token is expired
	if time.Now().After(resetToken.ExpiresAt) {
		return errors.New("reset token has expired")
	}

	// Hash new password
	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to process new password")
	}

	// Update user password
	resetToken.User.Password = hashedPassword
	if err := s.userRepo.Update(resetToken.User); err != nil {
		return errors.New("failed to update password")
	}

	// Mark token as used
	if err := s.userRepo.MarkPasswordResetTokenAsUsed(resetToken.ID); err != nil {
		return errors.New("failed to invalidate reset token")
	}

	return nil
}
