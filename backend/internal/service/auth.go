package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/auth"
	"github.com/ZhX589/UniBlack/backend/internal/captcha"
	"github.com/ZhX589/UniBlack/backend/internal/mailer"
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
	ErrCaptchaNotReady    = errors.New("captcha enabled but secret not configured")
)

// AuthService handles authentication logic
type AuthService struct {
	userRepo       *repository.UserRepository
	settingRepo    *repository.SystemSettingRepository
	accessListRepo *repository.AccessListRepository
	verifyRepo     *repository.VerificationRepository
	provider       auth.AuthProvider
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo *repository.UserRepository,
	settingRepo *repository.SystemSettingRepository,
	accessListRepo *repository.AccessListRepository,
	verifyRepo *repository.VerificationRepository,
	provider auth.AuthProvider,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		settingRepo:    settingRepo,
		accessListRepo: accessListRepo,
		verifyRepo:     verifyRepo,
		provider:       provider,
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

func (s *AuthService) loadCaptchaConfig(ctx context.Context) captcha.Config {
	var cfg captcha.Config
	_ = s.settingRepo.GetSettingValue(ctx, "security.captcha_enabled", &cfg.Enabled)
	_ = s.settingRepo.GetSettingValue(ctx, "security.captcha_provider", &cfg.Provider)
	_ = s.settingRepo.GetSettingValue(ctx, "security.captcha_site_key", &cfg.SiteKey)
	_ = s.settingRepo.GetSettingValue(ctx, "security.captcha_secret_key", &cfg.Secret)
	return cfg
}

func (s *AuthService) loadMailer(ctx context.Context) mailer.Mailer {
	var host, user, pass, from string
	var port int
	_ = s.settingRepo.GetSettingValue(ctx, "security.smtp_host", &host)
	_ = s.settingRepo.GetSettingValue(ctx, "security.smtp_port", &port)
	_ = s.settingRepo.GetSettingValue(ctx, "security.smtp_username", &user)
	_ = s.settingRepo.GetSettingValue(ctx, "security.smtp_password", &pass)
	_ = s.settingRepo.GetSettingValue(ctx, "security.smtp_from", &from)
	return mailer.New(mailer.Config{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
		From:     from,
	})
}

// Register creates a new user
func (s *AuthService) Register(ctx context.Context, req RegisterRequest, ip string) (*models.User, error) {
	var registrationEnabled bool
	if err := s.settingRepo.GetSettingValue(ctx, "auth.registration_enabled", &registrationEnabled); err == nil && !registrationEnabled {
		return nil, ErrRegistrationClosed
	}

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

	// Captcha (pluggable)
	capCfg := s.loadCaptchaConfig(ctx)
	if capCfg.Enabled {
		if capCfg.Secret == "" {
			return nil, ErrCaptchaNotReady
		}
		if err := captcha.NewProvider(capCfg).Verify(ctx, req.CaptchaToken, ip); err != nil {
			return nil, ErrInvalidCaptcha
		}
	}

	// Email verification code (pluggable via SMTP settings)
	var emailVerificationEnabled bool
	_ = s.settingRepo.GetSettingValue(ctx, "security.email_verification", &emailVerificationEnabled)
	if emailVerificationEnabled {
		if req.VerificationCode == "" {
			return nil, ErrInvalidCode
		}
		if s.verifyRepo == nil {
			return nil, ErrInvalidCode
		}
		if err := s.verifyRepo.Consume(ctx, req.Email, "register", req.VerificationCode); err != nil {
			return nil, ErrInvalidCode
		}
	}

	existing, _ := s.userRepo.GetUserByUsername(ctx, req.Username)
	if existing != nil {
		return nil, fmt.Errorf("username already exists")
	}
	existing, _ = s.userRepo.GetUserByEmail(ctx, req.Email)
	if existing != nil {
		return nil, fmt.Errorf("email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  string(hashedPassword),
		AuthProvider:  "local",
		IsActive:      true,
		EmailVerified: !emailVerificationEnabled || req.VerificationCode != "",
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if role, err := s.userRepo.GetRoleByName(ctx, "user"); err == nil && role != nil {
		_ = s.userRepo.AssignRole(ctx, user.ID, role.ID)
	}

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*auth.TokenPair, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if !user.IsActive {
		return nil, ErrUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = role.Name
	}
	identity := &auth.SubjectIdentity{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Roles:    roles,
		Provider: s.provider.Name(),
	}
	return s.provider.GenerateTokens(ctx, identity)
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	return s.provider.RefreshToken(ctx, refreshToken)
}

func (s *AuthService) ValidateToken(ctx context.Context, accessToken string) (*auth.SubjectIdentity, error) {
	return s.provider.ValidateToken(ctx, accessToken)
}

func (s *AuthService) GetUserPermissions(ctx context.Context, userID string) ([]models.Permission, error) {
	return s.userRepo.GetUserPermissions(ctx, userID)
}

// SendVerificationCode sends a registration verification code to email.
func (s *AuthService) SendVerificationCode(ctx context.Context, email string) error {
	var emailVerificationEnabled bool
	_ = s.settingRepo.GetSettingValue(ctx, "security.email_verification", &emailVerificationEnabled)
	if !emailVerificationEnabled {
		return fmt.Errorf("email verification is disabled")
	}
	if s.verifyRepo == nil {
		return fmt.Errorf("verification store unavailable")
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return err
	}
	_ = s.verifyRepo.InvalidateEmail(ctx, email, "register")
	row := &models.VerificationCode{
		Email:     email,
		Code:      code,
		Purpose:   "register",
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	if err := s.verifyRepo.Create(ctx, row); err != nil {
		return err
	}

	siteName := "UniBlack"
	_ = s.settingRepo.GetSettingValue(ctx, "site.name", &siteName)

	m := s.loadMailer(ctx)
	return m.Send(ctx, mailer.Message{
		To:      []string{email},
		Subject: fmt.Sprintf("[%s] 注册验证码", siteName),
		Body:    fmt.Sprintf("您的验证码是 %s，15 分钟内有效。\n\n如非本人操作请忽略。", code),
	})
}

// VerifyEmail verifies an email with a code (standalone).
func (s *AuthService) VerifyEmail(ctx context.Context, email, code string) error {
	if s.verifyRepo == nil {
		return ErrInvalidCode
	}
	return s.verifyRepo.Consume(ctx, email, "register", code)
}

// SeedAdmin creates the default admin user if it doesn't exist
func (s *AuthService) SeedAdmin(ctx context.Context, password string) error {
	existing, _ := s.userRepo.GetUserByUsername(ctx, "admin")
	if existing != nil {
		return nil
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
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
	adminRole, err := s.userRepo.GetRoleByName(ctx, "admin")
	if err != nil || adminRole == nil {
		return fmt.Errorf("admin role not found")
	}
	return s.userRepo.AssignRole(ctx, admin.ID, adminRole.ID)
}

func generateNumericCode(n int) (string, error) {
	var b []byte
	for i := 0; i < n; i++ {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b = append(b, byte('0'+v.Int64()))
	}
	return string(b), nil
}

func generateToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
