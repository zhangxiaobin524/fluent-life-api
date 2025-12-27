package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index:idx_posts_user_id" json:"user_id"`
	Content       string    `gorm:"type:text;not null" json:"content"`
	Tag           string    `gorm:"type:varchar(50);index:idx_posts_tag" json:"tag"`
	LikesCount    int       `gorm:"not null;default:0" json:"likes_count"`
	CommentsCount int       `gorm:"not null;default:0" json:"comments_count"`
	CreatedAt     time.Time `gorm:"index:idx_posts_created_at" json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	User     User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Likes    []PostLike `gorm:"foreignKey:PostID" json:"likes,omitempty"`
	Comments []Comment  `gorm:"foreignKey:PostID" json:"comments,omitempty"`
}

type PostLike struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PostID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_post_likes_post_user;index:idx_post_likes_post_id" json:"post_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_post_likes_post_user;index:idx_post_likes_user_id" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`

	Post Post `gorm:"foreignKey:PostID" json:"-"`
	User User `gorm:"foreignKey:UserID" json:"-"`
}

type Comment struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	PostID    uuid.UUID `gorm:"type:uuid;not null;index:idx_comments_post_id" json:"post_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_comments_user_id" json:"user_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Post Post `gorm:"foreignKey:PostID" json:"-"`
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (p *Post) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (pl *PostLike) BeforeCreate(tx *gorm.DB) error {
	if pl.ID == uuid.Nil {
		pl.ID = uuid.New()
	}
	return nil
}

func (c *Comment) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}






