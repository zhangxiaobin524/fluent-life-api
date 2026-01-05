package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Follow represents a follow relationship between two users
type Follow struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FollowerID uuid.UUID `gorm:"type:uuid;not null;index" json:"follower_id"` // ID of the user who initiated the follow
	FolloweeID uuid.UUID `gorm:"type:uuid;not null;index" json:"followee_id"` // ID of the user being followed
	CreatedAt  time.Time `json:"created_at"`
}

// BeforeCreate hook to set UUID if not already set
func (f *Follow) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name for the Follow model
func (Follow) TableName() string {
	return "follows"
}
