package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ZhX589/UniBlack/backend/internal/domain"
	"github.com/ZhX589/UniBlack/backend/internal/models"
	"github.com/ZhX589/UniBlack/backend/internal/repository"
	"github.com/ZhX589/UniBlack/backend/internal/storage"
)

var ErrSubmissionRestricted = errors.New("submission restricted")

type EventService struct {
	events    *repository.EventRepository
	subjects  *repository.SubjectRepository
	sanctions *repository.SanctionRepository
	users     *repository.UserRepository
	verifier  interface {
		VerifySubmissionValidation(context.Context, string, string, string, string) error
	}
}

func NewEventService(events *repository.EventRepository, subjects *repository.SubjectRepository, sanctions *repository.SanctionRepository, users *repository.UserRepository, verifier interface {
	VerifySubmissionValidation(context.Context, string, string, string, string) error
}) *EventService {
	return &EventService{events: events, subjects: subjects, sanctions: sanctions, users: users, verifier: verifier}
}

type PublishAccountRequest struct {
	Platform         string                 `json:"platform"`
	PlatformLabel    string                 `json:"platform_label"`
	AccountType      string                 `json:"account_type"`
	Username         string                 `json:"username"`
	AccountID        string                 `json:"account_id"`
	CustomAttributes map[string]interface{} `json:"custom_attributes"`
	IsPrimary        bool                   `json:"is_primary"`
}
type PublishEventRequest struct {
	Title        string     `json:"title"`
	Details      string     `json:"details"`
	Severity     int        `json:"severity"`
	OccurredFrom *time.Time `json:"occurred_from"`
	OccurredTo   *time.Time `json:"occurred_to"`
}
type PublishSubjectRequest struct {
	DisplayName      string                       `json:"display_name"`
	Accounts         []PublishAccountRequest      `json:"accounts"`
	Events           []PublishEventRequest        `json:"events"`
	VerificationCode string                       `json:"verification_code"`
	CaptchaToken     string                       `json:"captcha_token"`
	TextEvidence     []PublishTextEvidenceRequest `json:"text_evidence"`
}

type PublishTextEvidenceRequest struct {
	EventIndex int    `json:"event_index"`
	Title      string `json:"title"`
	Text       string `json:"text"`
}

func (r PublishTextEvidenceRequest) Validate(eventCount int) error {
	if r.EventIndex < 0 || r.EventIndex >= eventCount {
		return errors.New("invalid evidence event index")
	}
	if len(r.Text) == 0 || len(r.Text) > MaxTextEvidenceBytes {
		return errors.New("invalid text evidence size")
	}
	return nil
}

func (s *EventService) Publish(ctx context.Context, req PublishSubjectRequest, userID string) (*models.Subject, error) {
	if len(req.Accounts) == 0 || len(req.Events) == 0 {
		return nil, errors.New("at least one account and event required")
	}
	if s.users != nil && s.verifier != nil {
		user, err := s.users.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if err := s.verifier.VerifySubmissionValidation(ctx, user.Email, req.VerificationCode, req.CaptchaToken, userID); err != nil {
			return nil, err
		}
	}
	if s.sanctions != nil {
		blocked, err := s.sanctions.HasActiveSubmissionRestriction(ctx, userID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, ErrSubmissionRestricted
		}
	}
	inputs := make([]domain.AccountInput, 0, len(req.Accounts))
	accounts := make([]models.Account, 0, len(req.Accounts))
	seen := map[string]bool{}
	for _, a := range req.Accounts {
		if strings.TrimSpace(a.Platform) == "" || (strings.TrimSpace(a.Username) == "" && strings.TrimSpace(a.AccountID) == "") {
			return nil, errors.New("invalid account")
		}
		a.Platform = strings.ToLower(strings.TrimSpace(a.Platform))
		a.Username = strings.ToLower(strings.TrimSpace(a.Username))
		a.AccountID = strings.ToLower(strings.TrimSpace(a.AccountID))
		key := domain.AccountDedupKey(a.Platform, a.Username, a.AccountID)
		if seen[key] {
			return nil, errors.New("duplicate account")
		}
		seen[key] = true
		inputs = append(inputs, domain.AccountInput{Platform: a.Platform, Username: a.Username, AccountID: a.AccountID})
		var label, username, accountID *string
		if a.PlatformLabel != "" {
			label = &a.PlatformLabel
		}
		if a.Username != "" {
			username = &a.Username
		}
		if a.AccountID != "" {
			accountID = &a.AccountID
		}
		kind := a.AccountType
		if kind == "" {
			kind = "username"
		}
		attributes := a.CustomAttributes
		if attributes == nil {
			attributes = map[string]interface{}{}
		}
		accounts = append(accounts, models.Account{Platform: strings.ToLower(strings.TrimSpace(a.Platform)), PlatformLabel: label, AccountType: kind, Username: username, AccountID: accountID, CustomAttributes: attributes, IsPrimary: a.IsPrimary})
	}
	name, err := domain.ResolveDisplayName(req.DisplayName, inputs)
	if err != nil {
		return nil, err
	}
	publicID, err := domain.GeneratePublicID()
	if err != nil {
		return nil, err
	}
	// Subject lifecycle remains active/inactive; public visibility belongs to
	// each Event, whose initial status is published under the Phase 13 policy.
	subject := &models.Subject{PublicID: publicID, DisplayName: name, Status: "active", CreatedBy: &userID}
	events := make([]models.Event, 0, len(req.Events))
	for _, e := range req.Events {
		if strings.TrimSpace(e.Title) == "" || strings.TrimSpace(e.Details) == "" {
			return nil, errors.New("event title and details required")
		}
		severity := e.Severity
		if severity < 1 {
			severity = 1
		}
		if severity > 5 {
			severity = 5
		}
		if e.OccurredFrom != nil && e.OccurredTo != nil && e.OccurredTo.Before(*e.OccurredFrom) {
			return nil, errors.New("event time range invalid")
		}
		events = append(events, models.Event{Title: e.Title, Details: e.Details, Severity: severity, Status: "published", OccurredFrom: e.OccurredFrom, OccurredTo: e.OccurredTo, SubmittedBy: &userID})
	}
	// Store text blobs first. Their DB metadata is inserted by the same
	// transaction as the subject/accounts/events; failed DB writes compensate.
	evidence := make([]repository.EventEvidence, 0, len(req.TextEvidence))
	keys := make([]string, 0, len(req.TextEvidence))
	for number, item := range req.TextEvidence {
		if err := item.Validate(len(events)); err != nil {
			return nil, err
		}
		key := storage.BuildEvidenceKey(publicID, item.EventIndex+1, number+1, ".txt")
		body := []byte(item.Text)
		sum := sha256.Sum256(body)
		if _, err := s.events.StoreText(ctx, key, bytes.NewReader(body)); err != nil {
			return nil, err
		}
		keys = append(keys, key)
		size := int64(len(body))
		hash := hex.EncodeToString(sum[:])
		mime := "text/plain"
		original := key[strings.LastIndex(key, "/")+1:]
		evidence = append(evidence, repository.EventEvidence{
			EventIndex: item.EventIndex,
			Evidence:   models.Evidence{Type: "text", Title: &item.Title, StorageKey: &key, OriginalFilename: &original, FileSize: &size, SHA256: &hash, MimeType: &mime, UploadedBy: &userID},
		})
	}
	audit := &models.AuditLog{UserID: &userID, Action: "publish", ResourceType: "subject", Changes: map[string]interface{}{"public_id": publicID, "event_count": len(events)}}
	if err := s.events.PublishWithEvidence(ctx, subject, accounts, events, evidence, audit); err != nil {
		for _, key := range keys {
			_ = s.events.DeleteStored(ctx, key)
		}
		return nil, err
	}
	subject.Accounts = accounts
	subject.Events = events
	return subject, nil
}

func (s *EventService) Get(ctx context.Context, id string) (*models.Event, error) {
	return s.events.GetByID(ctx, id)
}

// CanManageEvent reports whether a requester may attach evidence to an event.
func (s *EventService) CanManageEvent(ctx context.Context, eventID, userID string, roles []string) (*models.Event, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role == "admin" || role == "moderator" {
			return event, nil
		}
	}
	if event.SubmittedBy != nil && *event.SubmittedBy == userID {
		return event, nil
	}
	return nil, errors.New("event evidence access denied")
}

func (s *EventService) CanReadEvent(ctx context.Context, eventID, userID string, roles []string) (*models.Event, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event.Status == "published" || event.Status == "corrected" {
		return event, nil
	}
	return s.CanManageEvent(ctx, eventID, userID, roles)
}

func (s *EventService) SubjectPublicID(ctx context.Context, eventID string) (string, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		return "", err
	}
	subject, err := s.subjects.GetSubjectByID(ctx, event.SubjectID)
	if err != nil {
		return "", err
	}
	return subject.PublicID, nil
}
