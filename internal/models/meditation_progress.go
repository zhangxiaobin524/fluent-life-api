package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MeditationProgress struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_meditation_progress_user_stage" json:"user_id"`
	Stage         int       `gorm:"not null;uniqueIndex:idx_meditation_progress_user_stage" json:"stage"` // 1-3
	CompletedDays int       `gorm:"not null;default:0" json:"completed_days"`
	Unlocked      bool      `gorm:"not null;default:false" json:"unlocked"`
	TotalTime     int       `gorm:"not null;default:0" json:"total_time"` // 新增字段：总冥想时长（秒）
	UpdatedAt     time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (m *MeditationProgress) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}






