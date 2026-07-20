package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"
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
	ErrSMTPRequired       = errors.New("smtp not configured")
)

func appEnv() string {
	if v := os.Getenv("APP_ENV"); v != "" {
		return strings.ToLower(v)
	}
	if v := os.Getenv("GO_ENV"); v != "" {
		return strings.ToLower(v)
	}
	return "development"
}

func isDevelopment() bool {
	e := appEnv()
	return e == "" || e == "development" || e == "dev"
}

// SettingReader reads system options (DB or OptionMap cache).
type SettingReader interface {
	GetSettingValue(ctx context.Context, key string, dest interface{}) error
}

// AuthService handles authentication logic
type AuthService struct {
	userRepo       *repository.UserRepository
	settings       SettingReader
	accessListRepo *repository.AccessListRepository
	verifyRepo     *repository.VerificationRepository
	provider       auth.AuthProvider
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo *repository.UserRepository,
	settings SettingReader,
	accessListRepo *repository.AccessListRepository,
	verifyRepo *repository.VerificationRepository,
	provider auth.AuthProvider,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		settings:       settings,
		accessListRepo: accessListRepo,
		verifyRepo:     verifyRepo,
		provider:       provider,
	}
}

func (s *AuthService) opt(ctx context.Context, key string, dest interface{}) {
	if s.settings != nil {
		_ = s.settings.GetSettingValue(ctx, key, dest)
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
	s.opt(ctx, "security.captcha_enabled", &cfg.Enabled)
	s.opt(ctx, "security.captcha_provider", &cfg.Provider)
	s.opt(ctx, "security.captcha_site_key", &cfg.SiteKey)
	s.opt(ctx, "security.captcha_secret_key", &cfg.Secret)
	return cfg
}

func (s *AuthService) loadMailer(ctx context.Context) mailer.Mailer {
	var host, user, pass, from string
	var port int
	var ssl, skipVerify bool
	s.opt(ctx, "security.smtp_host", &host)
	s.opt(ctx, "security.smtp_port", &port)
	s.opt(ctx, "security.smtp_username", &user)
	s.opt(ctx, "security.smtp_password", &pass)
	s.opt(ctx, "security.smtp_from", &from)
	s.opt(ctx, "security.smtp_ssl", &ssl)
	s.opt(ctx, "security.smtp_insecure_skip_verify", &skipVerify)
	return mailer.New(mailer.Config{
		Host:               host,
		Port:               port,
		Username:           user,
		Password:           pass,
		From:               from,
		SSL:                ssl,
		InsecureSkipVerify: skipVerify,
	})
}

// Register creates a new user
func (s *AuthService) Register(ctx context.Context, req RegisterRequest, ip string) (*models.User, error) {
	var registrationEnabled bool
	s.opt(ctx, "auth.registration_enabled", &registrationEnabled)
	// default open when unset
	if s.settings != nil {
		var probe interface{}
		if err := s.settings.GetSettingValue(ctx, "auth.registration_enabled", &probe); err == nil {
			if !registrationEnabled {
				return nil, ErrRegistrationClosed
			}
		}
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

	// Captcha: runtime is always demo when enabled (no third-party network).
	capCfg := s.loadCaptchaConfig(ctx)
	if capCfg.Enabled {
		if err := captcha.NewProvider(capCfg).Verify(ctx, req.CaptchaToken, ip); err != nil {
			return nil, ErrInvalidCaptcha
		}
	}

	// Email verification code
	var emailVerificationEnabled bool
	s.opt(ctx, "security.email_verification", &emailVerificationEnabled)
	if emailVerificationEnabled {
		if err := s.VerifyEmailCode(ctx, req.Email, "register", req.VerificationCode); err != nil {
			return nil, err
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

// SendVerificationCode sends a verification code for the given purpose (default register).
func (s *AuthService) SendVerificationCode(ctx context.Context, email string) error {
	return s.SendVerificationCodeForPurpose(ctx, email, "register")
}

// SendVerificationCodeForPurpose issues a code for register/submission/appeal.
func (s *AuthService) SendVerificationCodeForPurpose(ctx context.Context, email, purpose string) error {
	if purpose == "" {
		purpose = "register"
	}
	var emailVerificationEnabled bool
	s.opt(ctx, "security.email_verification", &emailVerificationEnabled)
	if !emailVerificationEnabled {
		return fmt.Errorf("email verification is disabled")
	}
	if s.verifyRepo == nil {
		return fmt.Errorf("verification store unavailable")
	}

	// Development: accept fixed 123456; do not send mail or store random codes.
	if isDevelopment() {
		return nil
	}

	var host string
	s.opt(ctx, "security.smtp_host", &host)
	if strings.TrimSpace(host) == "" {
		return ErrSMTPRequired
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return err
	}
	_ = s.verifyRepo.InvalidateEmail(ctx, email, purpose)
	row := &models.VerificationCode{
		Email:     email,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := s.verifyRepo.Create(ctx, row); err != nil {
		return err
	}

	siteName := "UniBlack"
	s.opt(ctx, "site.name", &siteName)

	m := s.loadMailer(ctx)
	return m.Send(ctx, mailer.Message{
		To:      []string{email},
		Subject: fmt.Sprintf("[%s] 验证码", siteName),
		Body:    fmt.Sprintf("您的验证码是 %s，10 分钟内有效。\n\n如非本人操作请忽略。", code),
	})
}

// VerifyEmailCode validates a code for purpose. Development accepts only 123456.
func (s *AuthService) VerifyEmailCode(ctx context.Context, email, purpose, code string) error {
	if strings.TrimSpace(code) == "" {
		return ErrInvalidCode
	}
	if purpose == "" {
		purpose = "register"
	}
	if isDevelopment() {
		if code != "123456" {
			return ErrInvalidCode
		}
		return nil
	}
	if s.verifyRepo == nil {
		return ErrInvalidCode
	}
	if err := s.verifyRepo.Consume(ctx, email, purpose, code); err != nil {
		return ErrInvalidCode
	}
	return nil
}

// VerifyEmail verifies an email with a code (standalone, register purpose).
func (s *AuthService) VerifyEmail(ctx context.Context, email, code string) error {
	return s.VerifyEmailCode(ctx, email, "register", code)
}

// VerifySubmissionValidation checks the configured email and demo-captcha
// requirements before a subject/event record may be published.
func (s *AuthService) VerifySubmissionValidation(ctx context.Context, email, code, captchaToken string) error {
	var emailVerificationEnabled bool
	s.opt(ctx, "security.email_verification", &emailVerificationEnabled)
	if emailVerificationEnabled {
		if err := s.VerifyEmailCode(ctx, email, "submission", code); err != nil {
			return err
		}
	}
	capCfg := s.loadCaptchaConfig(ctx)
	if capCfg.Enabled {
		if err := captcha.NewProvider(capCfg).Verify(ctx, captchaToken, ""); err != nil {
			return ErrInvalidCaptcha
		}
	}
	return nil
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
