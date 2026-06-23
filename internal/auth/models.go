package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey"`
	FullName          string    `gorm:"size:200;not null"`
	PhoneNumber       string    `gorm:"size:30;not null;uniqueIndex"`
	PasswordHash      string    `gorm:"type:text;not null"`
	District          string    `gorm:"size:100"`
	PreferredLanguage string    `gorm:"size:20;not null;default:english"`
	Role              string    `gorm:"size:20;not null;default:farmer"`
	IsActive          bool      `gorm:"not null;default:true"`
	CreatedAt         time.Time `gorm:"not null;default:now()"`
	UpdatedAt         time.Time `gorm:"not null;default:now()"`
}

func (User) TableName() string { return "users" }

func (u *User) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"type:text;not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	RevokedAt *time.Time `gorm:"default:null"`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

func (rt *RefreshToken) BeforeCreate(_ *gorm.DB) error {
	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	return nil
}

type DiagnosisReview struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey"`
	DiagnosisID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	OfficerID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	ReviewStatus       string     `gorm:"size:30;not null;default:pending"`
	ConfirmedCondition string     `gorm:"size:255"`
	OfficerComment     string     `gorm:"type:text"`
	Recommendation     string     `gorm:"type:text"`
	Urgency            string     `gorm:"size:20"`
	RequiresFieldVisit bool       `gorm:"not null;default:false"`
	IsAccepted         bool       `gorm:"not null;default:false"`
	IsHidden           bool       `gorm:"not null;default:false"`
	CreatedAt          time.Time  `gorm:"not null;default:now()"`
	UpdatedAt          time.Time  `gorm:"not null;default:now()"`
}

func (DiagnosisReview) TableName() string { return "diagnosis_reviews" }

func (r *DiagnosisReview) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type Notification struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index"`
	Title            string    `gorm:"size:255;not null"`
	Message          string    `gorm:"type:text;not null"`
	NotificationType string    `gorm:"size:50;not null"`
	IsRead           bool      `gorm:"not null;default:false"`
	EntityType       string    `gorm:"size:100"`
	EntityID         *uuid.UUID `gorm:"type:uuid"`
	CreatedAt        time.Time `gorm:"not null;default:now()"`
}

func (Notification) TableName() string { return "notifications" }

func (n *Notification) BeforeCreate(_ *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

type AuditLog struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey"`
	ActorUserID *uuid.UUID     `gorm:"type:uuid;index"`
	Action      string         `gorm:"size:100;not null"`
	EntityType  string         `gorm:"size:100;not null"`
	EntityID    *uuid.UUID     `gorm:"type:uuid"`
	Metadata    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'::jsonb"`
	CreatedAt   time.Time      `gorm:"not null;default:now()"`
}

func (AuditLog) TableName() string { return "audit_logs" }

func (a *AuditLog) BeforeCreate(_ *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

type TranscriptionFeedback struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null;index"`
	LanguageHint    string     `gorm:"size:20;not null"`
	Rating          string     `gorm:"size:30;not null"`
	CorrectionLength int       `gorm:"default:null"`
	CreatedAt       time.Time  `gorm:"not null;default:now()"`
}

func (TranscriptionFeedback) TableName() string { return "transcription_feedback" }

func (f *TranscriptionFeedback) BeforeCreate(_ *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
