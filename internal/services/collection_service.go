package services

import (
	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CollectionService struct {
	db *gorm.DB
}

func NewCollectionService(db *gorm.DB) *CollectionService {
	return &CollectionService{db: db}
}

// CollectPost 收藏帖子
func (s *CollectionService) CollectPost(userID, postID uuid.UUID) error {
	// 检查是否已经收藏
	var existingCollection models.PostCollection
	err := s.db.Where("user_id = ? AND post_id = ?", userID, postID).First(&existingCollection).Error
	if err == nil {
		return nil // 已经收藏，直接返回成功
	}

	// 创建收藏记录
	collection := models.PostCollection{
		UserID: userID,
		PostID: postID,
	}

	return s.db.Create(&collection).Error
}

// UncollectPost 取消收藏
func (s *CollectionService) UncollectPost(userID, postID uuid.UUID) error {
	return s.db.Where("user_id = ? AND post_id = ?", userID, postID).Delete(&models.PostCollection{}).Error
}

// IsCollected 检查是否已收藏
func (s *CollectionService) IsCollected(userID, postID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&models.PostCollection{}).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Count(&count).Error
	return count > 0, err
}

// GetCollectedPosts 获取用户收藏的帖子列表
func (s *CollectionService) GetCollectedPosts(userID uuid.UUID, page, pageSize int) ([]models.Post, int64, error) {
	var posts []models.Post
	var total int64

	// 计算总数
	s.db.Model(&models.PostCollection{}).Where("user_id = ?", userID).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.Table("posts").
		Select("posts.*").
		Joins("INNER JOIN post_collections ON posts.id = post_collections.post_id").
		Where("post_collections.user_id = ?", userID).
		Preload("User").
		Order("post_collections.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts).Error

	return posts, total, err
}

// GetPostCollectionCount 获取帖子收藏数
func (s *CollectionService) GetPostCollectionCount(postID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.Model(&models.PostCollection{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}
