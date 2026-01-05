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
	Image         string    `gorm:"type:text" json:"image"`
	LikesCount    int       `gorm:"not null;default:0" json:"likes_count"`
	CommentsCount int       `gorm:"not null;default:0" json:"comments_count"`
	FavoritesCount int      `gorm:"not null;default:0" json:"favorites_count"`
	CreatedAt     time.Time `gorm:"index:idx_posts_created_at" json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	// Transient field to indicate if the current user has liked the post
	IsLiked bool `gorm:"-" json:"is_liked"`
	// Transient field to indicate if the current user has collected the post
	IsCollected bool `gorm:"-" json:"is_collected"`

	User     User       `gorm:"foreignKey:UserID" json:"user"`
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
	LikesCount int       `gorm:"not null;default:0" json:"likes_count"` // New field for comment likes
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Transient field to indicate if the current user has liked the comment
	IsLiked bool `gorm:"-" json:"is_liked"`

	Post  Post        `gorm:"foreignKey:PostID" json:"-"`
	User  User        `gorm:"foreignKey:UserID" json:"user"`
	Likes []CommentLike `gorm:"foreignKey:CommentID" json:"likes,omitempty"` // New field for comment likes
}

type CommentLike struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CommentID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_comment_likes_comment_user;index:idx_comment_likes_comment_id" json:"comment_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_comment_likes_comment_user;index:idx_comment_likes_user_id" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`

	Comment Comment `gorm:"foreignKey:CommentID" json:"-"`
	User    User    `gorm:"foreignKey:UserID" json:"-"`
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

func (cl *CommentLike) BeforeCreate(tx *gorm.DB) error {
	if cl.ID == uuid.Nil {
		cl.ID = uuid.New()
	}
	return nil
}






