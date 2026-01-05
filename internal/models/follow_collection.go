package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Follow 用户关注关系
type Follow struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	FollowerID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_follow_follower_following;index:idx_user_follow_follower" json:"follower_id"`  // 关注者
	FolloweeID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_follow_follower_following;index:idx_user_follow_following" json:"followee_id"` // 被关注者
	CreatedAt   time.Time `json:"created_at"`

	Follower  User `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Followee User `gorm:"foreignKey:FolloweeID" json:"followee,omitempty"`
}

// TableName specifies the table name for the Follow model
func (Follow) TableName() string {
	return "follows"
}

// PostCollection 帖子收藏
type PostCollection struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_post_collection_user_post;index:idx_post_collection_user" json:"user_id"`
	PostID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_post_collection_user_post;index:idx_post_collection_post" json:"post_id"`
	CreatedAt time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Post Post `gorm:"foreignKey:PostID" json:"post,omitempty"`
}

func (uf *Follow) BeforeCreate(tx *gorm.DB) error {
	if uf.ID == uuid.Nil {
		uf.ID = uuid.New()
	}
	return nil
}

func (pc *PostCollection) BeforeCreate(tx *gorm.DB) error {
	if pc.ID == uuid.Nil {
		pc.ID = uuid.New()
	}
	return nil
}
