package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
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
	FileEvidence     []PublishFileEvidenceRequest `json:"file_evidence"`
	LinkEvidence     []PublishLinkEvidenceRequest `json:"link_evidence"`
}

// PublishLinkEvidenceRequest stores publisher-supplied link metadata only.
type PublishLinkEvidenceRequest struct {
	EventIndex  int    `json:"event_index"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type PublishTextEvidenceRequest struct {
	EventIndex int    `json:"event_index"`
	Title      string `json:"title"`
	Text       string `json:"text"`
}

// PublishFileEvidenceRequest carries binary content for multipart publish.
// Content is not JSON-serialized from the browser; the handler fills it.
type PublishFileEvidenceRequest struct {
	EventIndex int    `json:"event_index"`
	Title      string `json:"title"`
	Filename   string `json:"filename"`
	Content    []byte `json:"-"`
	Field      string `json:"field,omitempty"`
}

const MaxPublishFileBytes = 20 << 20

var allowedPublishFileExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".pdf": true, ".txt": true, ".bin": true, ".zip": true, ".webm": true,
	".mp4": true, ".md": true,
}

func (r PublishTextEvidenceRequest) Validate(eventCount int) error {
	if r.EventIndex < 0 || r.EventIndex >= eventCount {
		return errors.New("invalid evidence event index")
	}
	return validateTextEvidence(r.Text)
}

func (r PublishFileEvidenceRequest) Validate(eventCount int) error {
	if r.EventIndex < 0 || r.EventIndex >= eventCount {
		return errors.New("invalid evidence event index")
	}
	if len(r.Content) == 0 {
		return errors.New("file evidence content required")
	}
	if len(r.Content) > MaxPublishFileBytes {
		return fmt.Errorf("file evidence exceeds %d bytes", MaxPublishFileBytes)
	}
	ext := strings.ToLower(filepath.Ext(r.Filename))
	if ext == "" || !allowedPublishFileExt[ext] {
		return fmt.Errorf("file extension not allowed: %s", ext)
	}
	return nil
}

func (r PublishLinkEvidenceRequest) Validate(eventCount int) error {
	if r.EventIndex < 0 || r.EventIndex >= eventCount {
		return errors.New("invalid evidence event index")
	}
	return domain.ValidateLinkEvidence(r.Title, r.URL)
}

// nextEvidenceKey assigns per-event T### / F### sequence numbers.
func nextEvidenceKey(publicID string, eventIndex int, ext string, textN, fileN []int) string {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	eventNumber := eventIndex + 1
	if strings.EqualFold(ext, ".txt") {
		textN[eventIndex]++
		return storage.BuildEvidenceKey(publicID, eventNumber, textN[eventIndex], ext)
	}
	fileN[eventIndex]++
	return storage.BuildEvidenceKey(publicID, eventNumber, fileN[eventIndex], ext)
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
	// Store text/file blobs first. Link evidence is metadata-only.
	// DB metadata is inserted in the same transaction as subject/accounts/events;
	// failed DB writes compensate storage.
	evidence := make([]repository.EventEvidence, 0, len(req.TextEvidence)+len(req.FileEvidence)+len(req.LinkEvidence))
	keys := make([]string, 0, len(req.TextEvidence)+len(req.FileEvidence))
	textN := make([]int, len(events))
	fileN := make([]int, len(events))
	cleanup := func() {
		for _, key := range keys {
			_ = s.events.DeleteStored(ctx, key)
		}
	}

	for _, item := range req.LinkEvidence {
		if err := item.Validate(len(events)); err != nil {
			return nil, err
		}
		title := item.Title
		description := item.Description
		linkURL := item.URL
		evidence = append(evidence, repository.EventEvidence{
			EventIndex: item.EventIndex,
			Evidence:   models.Evidence{Type: "link", Title: &title, Description: &description, URL: &linkURL, UploadedBy: &userID},
		})
	}

	for _, item := range req.TextEvidence {
		if err := item.Validate(len(events)); err != nil {
			return nil, err
		}
		key := nextEvidenceKey(publicID, item.EventIndex, ".txt", textN, fileN)
		body := []byte(item.Text)
		sum := sha256.Sum256(body)
		if _, err := s.events.StoreBlob(ctx, key, bytes.NewReader(body), "text/plain; charset=utf-8"); err != nil {
			cleanup()
			return nil, err
		}
		keys = append(keys, key)
		size := int64(len(body))
		hash := hex.EncodeToString(sum[:])
		mime := "text/plain"
		title := item.Title
		if title == "" {
			title = "text evidence"
		}
		original := filepath.Base(key)
		evidence = append(evidence, repository.EventEvidence{
			EventIndex: item.EventIndex,
			Evidence:   models.Evidence{Type: "text", Title: &title, StorageKey: &key, OriginalFilename: &original, FileSize: &size, SHA256: &hash, MimeType: &mime, UploadedBy: &userID},
		})
	}

	for _, item := range req.FileEvidence {
		if err := item.Validate(len(events)); err != nil {
			cleanup()
			return nil, err
		}
		ext := strings.ToLower(filepath.Ext(item.Filename))
		key := nextEvidenceKey(publicID, item.EventIndex, ext, textN, fileN)
		sum := sha256.Sum256(item.Content)
		mime := getContentType(ext)
		if _, err := s.events.StoreBlob(ctx, key, bytes.NewReader(item.Content), mime); err != nil {
			cleanup()
			return nil, err
		}
		keys = append(keys, key)
		size := int64(len(item.Content))
		hash := hex.EncodeToString(sum[:])
		title := item.Title
		if title == "" {
			title = filepath.Base(item.Filename)
		}
		original := filepath.Base(item.Filename)
		evidence = append(evidence, repository.EventEvidence{
			EventIndex: item.EventIndex,
			Evidence:   models.Evidence{Type: "file", Title: &title, StorageKey: &key, OriginalFilename: &original, FileSize: &size, SHA256: &hash, MimeType: &mime, UploadedBy: &userID},
		})
	}

	audit := &models.AuditLog{UserID: &userID, Action: "publish", ResourceType: "subject", Changes: map[string]interface{}{
		"public_id": publicID, "event_count": len(events), "evidence_count": len(evidence),
	}}
	if err := s.events.PublishWithEvidence(ctx, subject, accounts, events, evidence, audit); err != nil {
		cleanup()
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
