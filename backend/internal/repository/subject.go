package repository

import (
	"context"
	"errors"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/ZhX589/UniBlack/backend/internal/models"
)

var (
	ErrSubjectNotFound      = errors.New("subject not found")
	ErrSubjectAlreadyExists = errors.New("subject already exists")
	ErrIdentifierExists     = errors.New("identifier already exists")
	ErrIdentifierNotFound   = errors.New("identifier not found")
)

// SubjectRepository handles subject database operations
type SubjectRepository struct {
	db *gorm.DB
}

// NewSubjectRepository creates a new subject repository
func NewSubjectRepository(db *gorm.DB) *SubjectRepository {
	return &SubjectRepository{db: db}
}

// CreateSubject creates a new subject
func (r *SubjectRepository) CreateSubject(ctx context.Context, subject *models.Subject) error {
	return r.db.WithContext(ctx).Create(subject).Error
}

// GetSubjectByID retrieves a subject by ID with identifiers
func (r *SubjectRepository) GetSubjectByID(ctx context.Context, id string) (*models.Subject, error) {
	var subject models.Subject
	query := r.db.WithContext(ctx).
		Preload("Identifiers").
		Preload("Accounts").
		Preload("Events")
	if strings.HasPrefix(id, "UBS_") {
		query = query.Where("public_id = ?", id)
	} else {
		query = query.Where("id = ?", id)
	}
	err := query.First(&subject).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubjectNotFound
		}
		return nil, err
	}
	return &subject, nil
}

// AccountConflicts returns normalized account identities that already exist.
func (r *SubjectRepository) AccountConflicts(ctx context.Context, accounts []models.Account) ([]string, error) {
	conflicts := make([]string, 0)
	for _, account := range accounts {
		platform := strings.ToLower(strings.TrimSpace(account.Platform))
		var count int64
		query := r.db.WithContext(ctx).Model(&models.Account{}).Where("lower(btrim(platform)) = ?", platform)
		if account.AccountID != nil && strings.TrimSpace(*account.AccountID) != "" {
			query = query.Where("lower(btrim(account_id)) = ?", strings.ToLower(strings.TrimSpace(*account.AccountID)))
		} else if account.Username != nil && strings.TrimSpace(*account.Username) != "" {
			query = query.Where("(account_id IS NULL OR btrim(account_id) = '') AND lower(btrim(username)) = ?", strings.ToLower(strings.TrimSpace(*account.Username)))
		} else {
			continue
		}
		if err := query.Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			conflicts = append(conflicts, platform)
		}
	}
	return conflicts, nil
}

// GetSubjectByIdentifier retrieves a subject by platform and value
func (r *SubjectRepository) GetSubjectByIdentifier(ctx context.Context, platform, value string) (*models.Subject, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	value = strings.ToLower(strings.TrimSpace(value))
	var subject models.Subject
	err := r.db.WithContext(ctx).
		Preload("Identifiers").
		Preload("Accounts").
		Joins("JOIN accounts ON accounts.subject_id = subjects.id").
		Where("subjects.status = 'active' AND lower(btrim(accounts.platform)) = ? AND (lower(btrim(accounts.username)) = ? OR lower(btrim(accounts.account_id)) = ?)", platform, value, value).
		First(&subject).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = r.db.WithContext(ctx).
			Preload("Identifiers").
			Preload("Accounts").
			Joins("JOIN identifiers ON identifiers.subject_id = subjects.id").
			Where("subjects.status = 'active' AND lower(btrim(identifiers.platform)) = ? AND lower(btrim(identifiers.value)) = ?", platform, value).
			First(&subject).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSubjectNotFound
		}
		return nil, err
	}
	return &subject, nil
}

// ListSubjects lists all subjects with pagination
func (r *SubjectRepository) ListSubjects(ctx context.Context, offset, limit int, status string) ([]models.Subject, int64, error) {
	var subjects []models.Subject
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Subject{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.
		Preload("Identifiers").
		Preload("Accounts").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&subjects).Error

	return subjects, total, err
}

// UpdateSubject updates a subject
func (r *SubjectRepository) UpdateSubject(ctx context.Context, subject *models.Subject) error {
	return r.db.WithContext(ctx).Save(subject).Error
}

// DeleteSubject deletes a subject and its identifiers
func (r *SubjectRepository) DeleteSubject(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete identifiers first
		if err := tx.Where("subject_id = ?", id).Delete(&models.Identifier{}).Error; err != nil {
			return err
		}
		// Delete subject
		return tx.Where("id = ?", id).Delete(&models.Subject{}).Error
	})
}

// IncrementCaseCount increments the case count for a subject
func (r *SubjectRepository) IncrementCaseCount(ctx context.Context, subjectID string) error {
	return r.db.WithContext(ctx).
		Model(&models.Subject{}).
		Where("id = ?", subjectID).
		Update("case_count", gorm.Expr("case_count + 1")).Error
}

// AddIdentifier adds an identifier to a subject
func (r *SubjectRepository) AddIdentifier(ctx context.Context, identifier *models.Identifier) error {
	// Check if identifier already exists
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Identifier{}).
		Where("platform = ? AND value = ?", identifier.Platform, identifier.Value).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrIdentifierExists
	}

	return r.db.WithContext(ctx).Create(identifier).Error
}

// RemoveIdentifier removes an identifier
func (r *SubjectRepository) RemoveIdentifier(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Identifier{})
	if result.RowsAffected == 0 {
		return ErrIdentifierNotFound
	}
	return result.Error
}

// GetIdentifiersBySubjectID retrieves all identifiers for a subject
func (r *SubjectRepository) GetIdentifiersBySubjectID(ctx context.Context, subjectID string) ([]models.Identifier, error) {
	var identifiers []models.Identifier
	err := r.db.WithContext(ctx).
		Where("subject_id = ?", subjectID).
		Order("is_primary DESC, created_at ASC").
		Find(&identifiers).Error
	return identifiers, err
}

// SearchSubjects searches subjects by identifier value
func (r *SubjectRepository) SearchSubjects(ctx context.Context, query string) ([]models.Subject, error) {
	var subjects []models.Subject
	pattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Preload("Identifiers").
		Preload("Accounts").
		Joins("LEFT JOIN accounts ON accounts.subject_id = subjects.id").
		Joins("LEFT JOIN identifiers ON identifiers.subject_id = subjects.id").
		Where("subjects.status = 'active' AND (subjects.display_name ILIKE ? OR accounts.username ILIKE ? OR accounts.account_id ILIKE ? OR identifiers.value ILIKE ?)", pattern, pattern, pattern, pattern).
		Group("subjects.id").
		Order("subjects.created_at DESC").
		Limit(50).
		Find(&subjects).Error
	if err == nil {
		needle := strings.ToLower(query)
		sort.SliceStable(subjects, func(i, j int) bool {
			return accountMatches(subjects[i].Accounts, needle) && !accountMatches(subjects[j].Accounts, needle)
		})
	}
	return subjects, err
}

func accountMatches(accounts []models.Account, needle string) bool {
	for _, account := range accounts {
		if (account.Username != nil && strings.Contains(strings.ToLower(*account.Username), needle)) ||
			(account.AccountID != nil && strings.Contains(strings.ToLower(*account.AccountID), needle)) {
			return true
		}
	}
	return false
}

// PublicStatistics is intentionally Event-first; legacy CaseCount is not a public metric.
// Event counts include published and corrected public records.
func (r *SubjectRepository) PublicStatistics(ctx context.Context) (int64, int64, error) {
	var subjects, events int64
	if err := r.db.WithContext(ctx).Model(&models.Subject{}).Where("status = 'active'").Count(&subjects).Error; err != nil {
		return 0, 0, err
	}
	if err := r.db.WithContext(ctx).Model(&models.Event{}).Where("status IN ?", []string{"published", "corrected"}).Count(&events).Error; err != nil {
		return 0, 0, err
	}
	return subjects, events, nil
}
