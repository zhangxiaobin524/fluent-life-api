package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PracticeRoom 对练房模型
type PracticeRoom struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index:idx_rooms_user_id" json:"user_id"`
	Title       string    `gorm:"type:varchar(100);not null" json:"title"`
	Theme       string    `gorm:"type:varchar(50);not null;index:idx_rooms_theme" json:"theme"` // 房间主题：日常对话、商务沟通、演讲练习等
	Type        string    `gorm:"type:varchar(50);not null;index:idx_rooms_type" json:"type"`   // 房间类型：公开房间、私密房间、限时房间、练习模式
	Description string    `gorm:"type:text" json:"description,omitempty"`
	MaxMembers  int       `gorm:"not null;default:2" json:"max_members"` // 最大成员数
	CurrentMembers int    `gorm:"not null;default:1" json:"current_members"` // 当前成员数
	IsActive    bool      `gorm:"not null;default:true;index:idx_rooms_active" json:"is_active"`
	CreatedAt   time.Time `gorm:"index:idx_rooms_created_at" json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	User    User              `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Members []PracticeRoomMember `gorm:"foreignKey:RoomID" json:"members,omitempty"`
}

// PracticeRoomMember 对练房成员
type PracticeRoomMember struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	RoomID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_room_member;index:idx_member_room_id" json:"room_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_room_member;index:idx_member_user_id" json:"user_id"`
	JoinedAt  time.Time `json:"joined_at"`
	IsHost    bool      `gorm:"not null;default:false" json:"is_host"` // 是否是房主

	Room PracticeRoom `gorm:"foreignKey:RoomID" json:"-"`
	User User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (pr *PracticeRoom) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	return nil
}

func (prm *PracticeRoomMember) BeforeCreate(tx *gorm.DB) error {
	if prm.ID == uuid.Nil {
		prm.ID = uuid.New()
	}
	if prm.JoinedAt.IsZero() {
		prm.JoinedAt = time.Now()
	}
	return nil
}




