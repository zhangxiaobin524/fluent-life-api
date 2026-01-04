package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username     string     `gorm:"type:varchar(50);not null;uniqueIndex" json:"username"`
	Email        *string    `gorm:"type:varchar(255);uniqueIndex" json:"email,omitempty"`
	Phone        *string    `gorm:"type:varchar(20);uniqueIndex" json:"phone,omitempty"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	AvatarURL    *string    `gorm:"type:varchar(500)" json:"avatar_url,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// UserProfile 包含用户基本信息和统计数据
type UserProfile struct {
	User
	TotalTrainingDays   int         `json:"total_training_days"`
	TotalTrainingMinutes int         `json:"total_training_minutes"`
	BraveryBadges       []Achievement `json:"bravery_badges"` // 假设 Achievement 是勋章模型
	WeeklyActivity      []int       `json:"weekly_activity"`  // 例如，一周内每天的活跃度
}






