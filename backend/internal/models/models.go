package models

import (
	"time"
)

// User represents a system user
type User struct {
	ID                      string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Username                string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"username"`
	Email                   string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash            string         `gorm:"type:varchar(255);not null" json:"-"`
	AuthProvider            string         `gorm:"type:varchar(50);default:'local'" json:"auth_provider"`
	ExternalID              *string        `gorm:"type:varchar(255)" json:"external_id,omitempty"`
	DisplayName             *string        `gorm:"type:varchar(255)" json:"display_name,omitempty"`
	AvatarURL               *string        `gorm:"type:varchar(512)" json:"avatar_url,omitempty"`
	IsActive                bool           `gorm:"default:true" json:"is_active"`
	EmailVerified           bool           `gorm:"default:false" json:"email_verified"`
	EmailVerificationToken  *string        `gorm:"type:varchar(255)" json:"-"`
	EmailVerificationExpiresAt *time.Time  `json:"-"`
	LastLoginAt             *time.Time     `json:"last_login_at,omitempty"`
	OrgID                   *string        `gorm:"type:uuid" json:"org_id,omitempty"`
	CreatedAt               time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	Roles                   []Role         `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// Role represents a user role
type Role struct {
	ID          string       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string       `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	Description *string      `gorm:"type:text" json:"description,omitempty"`
	IsSystem    bool         `gorm:"default:false" json:"is_system"`
	CreatedAt   time.Time    `gorm:"autoCreateTime" json:"created_at"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// Permission represents a system permission
type Permission struct {
	ID          string  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string  `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	Resource    string  `gorm:"type:varchar(50)" json:"resource"`
	Action      string  `gorm:"type:varchar(50)" json:"action"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// UserRole represents the user-role association
type UserRole struct {
	UserID string `gorm:"type:uuid;primaryKey"`
	RoleID string `gorm:"type:uuid;primaryKey"`
}

// RolePermission represents the role-permission association
type RolePermission struct {
	RoleID       string `gorm:"type:uuid;primaryKey"`
	PermissionID string `gorm:"type:uuid;primaryKey"`
}

// Subject represents a blacklisted entity
type Subject struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DisplayName string         `gorm:"type:varchar(255);not null" json:"display_name"`
	Notes       *string        `gorm:"type:text" json:"notes,omitempty"`
	RiskLevel   int            `gorm:"default:0" json:"risk_level"`
	CaseCount   int            `gorm:"default:0" json:"case_count"`
	Status      string         `gorm:"type:varchar(20);default:'active'" json:"status"`
	CreatedBy   *string        `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	Identifiers []Identifier   `gorm:"foreignKey:SubjectID" json:"identifiers,omitempty"`
	Cases       []Case         `gorm:"foreignKey:SubjectID" json:"cases,omitempty"`
}

// Identifier represents a subject identifier (QQ, Discord, etc.)
type Identifier struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SubjectID   string    `gorm:"type:uuid;not null" json:"subject_id"`
	Platform    string    `gorm:"type:varchar(50);not null" json:"platform"`
	AccountType string    `gorm:"type:varchar(50);not null;default:'username'" json:"account_type"`
	Value       string    `gorm:"type:varchar(255);not null" json:"value"`
	Label       *string   `gorm:"type:varchar(100)" json:"label,omitempty"`
	IsPrimary   bool      `gorm:"default:false" json:"is_primary"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// Case represents a blacklist case
type Case struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SubjectID   string     `gorm:"type:uuid;not null" json:"subject_id"`
	Title       string     `gorm:"type:varchar(255);not null" json:"title"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	Status      string     `gorm:"type:varchar(20);default:'draft'" json:"status"`
	Severity    int        `gorm:"default:1" json:"severity"`
	Verdict     *string    `gorm:"type:text" json:"verdict,omitempty"`
	SubmittedBy *string    `gorm:"type:uuid" json:"submitted_by,omitempty"`
	ReviewedBy  *string    `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	Evidences   []Evidence `gorm:"foreignKey:CaseID" json:"evidences,omitempty"`
}

// TableName returns the table name for Case
func (Case) TableName() string {
	return "cases"
}

// Evidence represents case evidence
type Evidence struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CaseID      string    `gorm:"type:uuid;not null" json:"case_id"`
	Type        string    `gorm:"type:varchar(20);not null" json:"type"`
	Title       *string   `gorm:"type:varchar(255)" json:"title,omitempty"`
	Description *string   `gorm:"type:text" json:"description,omitempty"`
	URL         *string   `gorm:"type:varchar(512)" json:"url,omitempty"`
	FileSize    *int64    `gorm:"type:bigint" json:"file_size,omitempty"`
	SHA256      *string   `gorm:"type:varchar(64)" json:"sha256,omitempty"`
	MimeType    *string   `gorm:"type:varchar(100)" json:"mime_type,omitempty"`
	UploadedBy  *string   `gorm:"type:uuid" json:"uploaded_by,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName returns the table name for Evidence
func (Evidence) TableName() string {
	return "evidence"
}

// Submission represents a user submission
type Submission struct {
	ID                  string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CaseID              *string    `gorm:"type:uuid" json:"case_id,omitempty"`
	SubjectIdentifiers  string     `gorm:"type:jsonb" json:"subject_identifiers"`
	Reason              string     `gorm:"type:text;not null" json:"reason"`
	Status              string     `gorm:"type:varchar(20);default:'draft'" json:"status"`
	SubmittedBy         *string    `gorm:"type:uuid" json:"submitted_by,omitempty"`
	ReviewedBy          *string    `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewNotes         *string    `gorm:"type:text" json:"review_notes,omitempty"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt           time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// Appeal represents an appeal against a case
type Appeal struct {
	ID          string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CaseID      string     `gorm:"type:uuid;not null" json:"case_id"`
	Reason      string     `gorm:"type:text;not null" json:"reason"`
	Status      string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	SubmittedBy *string    `gorm:"type:uuid" json:"submitted_by,omitempty"`
	ReviewedBy  *string    `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewNotes *string    `gorm:"type:text" json:"review_notes,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           string                 `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       *string                `gorm:"type:uuid" json:"user_id,omitempty"`
	Action       string                 `gorm:"type:varchar(50);not null" json:"action"`
	ResourceType string                 `gorm:"type:varchar(50);not null" json:"resource_type"`
	ResourceID   *string                `gorm:"type:uuid" json:"resource_id,omitempty"`
	Changes      map[string]interface{} `gorm:"type:jsonb" json:"changes,omitempty"`
	IPAddress    *string                `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent    *string                `gorm:"type:text" json:"user_agent,omitempty"`
	CreatedAt    time.Time              `gorm:"autoCreateTime" json:"created_at"`
}

// SystemSetting represents a system configuration
type SystemSetting struct {
	ID          string      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Key         string      `gorm:"type:varchar(100);uniqueIndex;not null" json:"key"`
	Value       string      `gorm:"type:jsonb;not null;default:'{}'" json:"value"`
	Description *string     `gorm:"type:text" json:"description,omitempty"`
	UpdatedBy   *string     `gorm:"type:uuid" json:"updated_by,omitempty"`
	UpdatedAt   time.Time   `gorm:"autoUpdateTime" json:"updated_at"`
}

// AccessList represents a whitelist/blacklist entry
type AccessList struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Type      string    `gorm:"type:varchar(20);not null" json:"type"`
	Target    string    `gorm:"type:varchar(50);not null" json:"target"`
	Value     string    `gorm:"type:varchar(255);not null" json:"value"`
	Reason    *string   `gorm:"type:text" json:"reason,omitempty"`
	CreatedBy *string   `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// VerificationCode is a short-lived code for email verification flows.
type VerificationCode struct {
	ID        string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email     string     `gorm:"type:varchar(255);not null" json:"email"`
	Code      string     `gorm:"type:varchar(32);not null" json:"-"`
	Purpose   string     `gorm:"type:varchar(50);not null;default:'register'" json:"purpose"`
	ExpiresAt time.Time  `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

func (VerificationCode) TableName() string { return "verification_codes" }
