package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ZhX589/UniBlack/backend/internal/auth"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive      = errors.New("user is inactive")
)

// AuthService handles authentication logic
type AuthService struct {
	userRepo *repository.UserRepository
	provider auth.AuthProvider
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *repository.UserRepository, provider auth.AuthProvider) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		provider: provider,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// Register creates a new user
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*models.User, error) {
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
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		AuthProvider: "local",
		IsActive:     true,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Assign default user role
	// TODO: Get default role ID from database
	// s.userRepo.AssignRole(ctx, user.ID, defaultRoleID)

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
