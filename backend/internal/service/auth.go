package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/auth"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
	ErrRegistrationClosed = errors.New("registration is closed")
	ErrInvalidCaptcha     = errors.New("invalid captcha")
	ErrInvalidCode        = errors.New("invalid verification code")
)

// AuthService handles authentication logic
type AuthService struct {
	userRepo      *repository.UserRepository
	settingRepo   *repository.SystemSettingRepository
	accessListRepo *repository.AccessListRepository
	provider      auth.AuthProvider
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo *repository.UserRepository,
	settingRepo *repository.SystemSettingRepository,
	accessListRepo *repository.AccessListRepository,
	provider auth.AuthProvider,
) *AuthService {
	return &AuthService{
		userRepo:      userRepo,
		settingRepo:   settingRepo,
		accessListRepo: accessListRepo,
		provider:      provider,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username         string `json:"username" validate:"required"`
	Email            string `json:"email" validate:"required,email"`
	Password         string `json:"password" validate:"required,min=8"`
	VerificationCode string `json:"verification_code"`
	CaptchaToken     string `json:"captcha_token"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Register creates a new user
func (s *AuthService) Register(ctx context.Context, req RegisterRequest, ip string) (*models.User, error) {
	// Check if registration is enabled
	var registrationEnabled bool
	s.settingRepo.GetSettingValue(ctx, "auth.registration_enabled", &registrationEnabled)
	if !registrationEnabled {
		return nil, ErrRegistrationClosed
	}

	// Check blacklist
	isBlacklisted, _ := s.accessListRepo.IsListed(ctx, "blacklist", "ip", ip)
	if isBlacklisted {
		return nil, fmt.Errorf("access denied")
	}

	isBlacklisted, _ = s.accessListRepo.IsListed(ctx, "blacklist", "email", req.Email)
	if isBlacklisted {
		return nil, fmt.Errorf("email not allowed")
	}

	isBlacklisted, _ = s.accessListRepo.IsListed(ctx, "blacklist", "username", req.Username)
	if isBlacklisted {
		return nil, fmt.Errorf("username not allowed")
	}

	// Check captcha if enabled
	var captchaEnabled bool
	s.settingRepo.GetSettingValue(ctx, "security.captcha_enabled", &captchaEnabled)
	if captchaEnabled && req.CaptchaToken == "" {
		return nil, ErrInvalidCaptcha
	}
	// TODO: Validate captcha token with provider

	// Check email verification if enabled
	var emailVerificationEnabled bool
	s.settingRepo.GetSettingValue(ctx, "security.email_verification", &emailVerificationEnabled)
	if emailVerificationEnabled && req.VerificationCode == "" {
		return nil, ErrInvalidCode
	}
	// TODO: Validate verification code

	// Check if user already exists
	existing, _ := s.userRepo.GetUserByUsername(ctx, req.Username)
	if existing != nil {
		return nil, fmt.Errorf("username already exists")
	}

	existing, _ = s.userRepo.GetUserByEmail(ctx, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		AuthProvider:  "local",
		IsActive:      true,
		EmailVerified: !emailVerificationEnabled, // Auto-verify if email verification is disabled
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign default user role
	// TODO: Get default role ID from database and assign

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*auth.TokenPair, error) {
	// Get user by username
	user, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	s.userRepo.UpdateLastLogin(ctx, user.ID)

	// Get user roles
	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = role.Name
	}

	// Generate tokens
	identity := &auth.SubjectIdentity{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    roles,
		Provider: s.provider.Name(),
	}

	return s.provider.GenerateTokens(ctx, identity)
}

// RefreshToken refreshes an access token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	return s.provider.RefreshToken(ctx, refreshToken)
}

// ValidateToken validates an access token and returns the identity
func (s *AuthService) ValidateToken(ctx context.Context, accessToken string) (*auth.SubjectIdentity, error) {
	return s.provider.ValidateToken(ctx, accessToken)
}

// GetUserPermissions returns all permissions for a user
func (s *AuthService) GetUserPermissions(ctx context.Context, userID string) ([]models.Permission, error) {
	return s.userRepo.GetUserPermissions(ctx, userID)
}

// SendVerificationCode sends a verification code to an email
func (s *AuthService) SendVerificationCode(ctx context.Context, email string) error {
	// Generate 6-digit code
	code := generateVerificationCode()

	// Store code with expiration (15 minutes)
	expiresAt := time.Now().Add(15 * time.Minute)
	// TODO: Store code in database or cache
	_ = code
	_ = expiresAt

	// TODO: Send email via SMTP
	return nil
}

// VerifyEmail verifies an email with a code
func (s *AuthService) VerifyEmail(ctx context.Context, email, code string) error {
	// TODO: Verify code from database/cache
	// TODO: Update user email_verified
	return nil
}

// SeedAdmin creates the default admin user if it doesn't exist
func (s *AuthService) SeedAdmin(ctx context.Context, password string) error {
	// Check if admin exists
	existing, _ := s.userRepo.GetUserByUsername(ctx, "admin")
	if existing != nil {
		return nil // Admin already exists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create admin user
	admin := &models.User{
		Username:      "admin",
		Email:         "admin@uniblack.local",
		PasswordHash:  string(hashedPassword),
		AuthProvider:  "local",
		IsActive:      true,
		EmailVerified: true,
	}

	if err := s.userRepo.CreateUser(ctx, admin); err != nil {
		return fmt.Errorf("failed to create admin: %w", err)
	}

	// Get admin role
	adminRole, err := s.userRepo.GetRoleByName(ctx, "admin")
	if err != nil || adminRole == nil {
		return fmt.Errorf("admin role not found")
	}

	// Assign admin role
	if err := s.userRepo.AssignRole(ctx, admin.ID, adminRole.ID); err != nil {
		return fmt.Errorf("failed to assign admin role: %w", err)
	}

	return nil
}

// generateVerificationCode generates a 6-digit verification code
func generateVerificationCode() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%06d", int(b[0])<<16|int(b[1])<<8|int(b[2])%1000000)
}

// generateToken generates a random token
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
