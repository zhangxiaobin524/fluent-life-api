package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VerificationCode struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Identifier string    `gorm:"type:varchar(255);not null;index:idx_verification_codes_identifier" json:"identifier"`
	Code       string    `gorm:"type:varchar(6);not null" json:"-"`
	Type       string    `gorm:"type:varchar(20);not null" json:"type"` // 'register' | 'login'
	ExpiresAt  time.Time `gorm:"not null;index:idx_verification_codes_expires_at" json:"expires_at"`
	Used       bool      `gorm:"default:false" json:"used"`
	CreatedAt  time.Time `json:"created_at"`
}

func (v *VerificationCode) BeforeCreate(tx *gorm.DB) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	return nil
}






