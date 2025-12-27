package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Achievement struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_achievements_user_type;index:idx_achievements_user_id" json:"user_id"`
	AchievementType string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_achievements_user_type" json:"achievement_type"`
	UnlockedAt      time.Time `json:"unlocked_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (a *Achievement) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}






