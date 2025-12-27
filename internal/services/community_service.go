package services

import (
	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommunityService struct {
	db *gorm.DB
}

func NewCommunityService(db *gorm.DB) *CommunityService {
	return &CommunityService{db: db}
}

func (s *CommunityService) CreatePost(userID uuid.UUID, content, tag string) (*models.Post, error) {
	post := models.Post{
		UserID:        userID,
		Content:       content,
		Tag:           tag,
		LikesCount:    0,
		CommentsCount: 0,
	}

	if err := s.db.Create(&post).Error; err != nil {
		return nil, err
	}

	return &post, nil
}

func (s *CommunityService) GetPosts(page, pageSize int, userID *uuid.UUID) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	s.db.Model(&models.Post{}).Count(&total)

	offset := (page - 1) * pageSize
	if err := s.db.Preload("User").Preload("Likes").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&posts).Error; err != nil {
		return nil, 0, err
	}

	// 如果提供了用户ID，标记用户是否已点赞
	if userID != nil {
		for i := range posts {
			for _, like := range posts[i].Likes {
				if like.UserID == *userID {
					// 这里我们需要一个临时字段来存储点赞状态
					// 但由于Post模型没有这个字段，我们可以在handler层处理
				}
			}
		}
	}

	return posts, total, nil
}

func (s *CommunityService) GetPost(postID uuid.UUID) (*models.Post, error) {
	var post models.Post
	if err := s.db.Preload("User").Preload("Likes").Preload("Comments.User").First(&post, postID).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

func (s *CommunityService) ToggleLike(userID, postID uuid.UUID) (bool, error) {
	var like models.PostLike
	err := s.db.Where("user_id = ? AND post_id = ?", userID, postID).First(&like).Error

	if err == gorm.ErrRecordNotFound {
		// 点赞
		like = models.PostLike{
			UserID: userID,
			PostID: postID,
		}
		s.db.Create(&like)
		s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("likes_count", gorm.Expr("likes_count + 1"))
		return true, nil
	} else if err == nil {
		// 取消点赞
		s.db.Delete(&like)
		s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("likes_count", gorm.Expr("likes_count - 1"))
		return false, nil
	}

	return false, err
}

func (s *CommunityService) CreateComment(userID, postID uuid.UUID, content string) (*models.Comment, error) {
	comment := models.Comment{
		UserID:  userID,
		PostID:  postID,
		Content: content,
	}

	if err := s.db.Create(&comment).Error; err != nil {
		return nil, err
	}

	s.db.Model(&models.Post{}).Where("id = ?", postID).UpdateColumn("comments_count", gorm.Expr("comments_count + 1"))

	return &comment, nil
}

func (s *CommunityService) GetComments(postID uuid.UUID) ([]models.Comment, error) {
	var comments []models.Comment
	if err := s.db.Preload("User").Where("post_id = ?", postID).Order("created_at ASC").Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

