package diagnosis

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type CropDiagnosis struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	Crop              string  `gorm:"size:100;not null"`
	District          string  `gorm:"size:100"`
	PreferredLanguage string  `gorm:"size:20;not null;default:english"`
	PlantPart         string  `gorm:"size:100"`
	SymptomDescription string `gorm:"type:text;not null"`
	SymptomsStartedAt *time.Time `gorm:"type:date"`
	AffectedPercentage *float64  `gorm:"type:numeric(5,2)"`

	RecentWeather     string `gorm:"type:text"`
	FertiliserHistory string `gorm:"type:text"`
	PesticideHistory  string `gorm:"type:text"`

	ImageStoragePath  string `gorm:"type:text;not null"`
	ImageOriginalName string `gorm:"size:255"`
	ImageContentType  string `gorm:"size:100;not null"`
	ImageSizeBytes    int64  `gorm:"not null"`
	ImageSHA256       string `gorm:"size:64"`

	ProbableCondition  string         `gorm:"size:255"`
	Confidence         float64        `gorm:"type:numeric(5,2)"`
	ConfidenceLabel    string         `gorm:"size:20"`
	Description        string         `gorm:"type:text"`
	ObservedSigns      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb"`
	PossibleAlternatives datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb"`
	RecommendedActions datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb"`
	PreventionTips     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb"`

	Urgency             string `gorm:"size:20"`
	RequiresExpertReview bool  `gorm:"not null;default:true"`
	Disclaimer          string `gorm:"type:text"`

	RawAIResult datatypes.JSON `gorm:"type:jsonb"`
	Model       string         `gorm:"size:150"`

	Status       string `gorm:"size:30;not null;default:processing"`
	ErrorMessage string `gorm:"type:text"`

	CreatedAt time.Time `gorm:"not null;default:now();index"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (CropDiagnosis) TableName() string {
	return "crop_diagnoses"
}

func (d *CropDiagnosis) BeforeCreate(_ interface{}) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
