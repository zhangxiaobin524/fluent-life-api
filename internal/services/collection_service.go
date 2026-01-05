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
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查是否已经收藏
		var existingCollection models.PostCollection
		err := tx.Where("user_id = ? AND post_id = ?", userID, postID).First(&existingCollection).Error
		if err == nil {
			return nil // 已经收藏，直接返回成功
		}

		// 创建收藏记录
		collection := models.PostCollection{
			UserID: userID,
			PostID: postID,
		}
		if err := tx.Create(&collection).Error; err != nil {
			return err
		}

		// 更新帖子的收藏数量
		if err := tx.Model(&models.Post{}).Where("id = ?", postID).
			UpdateColumn("favorites_count", gorm.Expr("favorites_count + ?", 1)).Error; err != nil {
			return err
		}

		return nil
	})
}

// UncollectPost 取消收藏
func (s *CollectionService) UncollectPost(userID, postID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除收藏记录
		result := tx.Where("user_id = ? AND post_id = ?", userID, postID).Delete(&models.PostCollection{})
		if result.Error != nil {
			return result.Error
		}
		// 如果没有记录被删除，则直接返回成功
		if result.RowsAffected == 0 {
			return nil
		}

		// 更新帖子的收藏数量
		if err := tx.Model(&models.Post{}).Where("id = ?", postID).
			UpdateColumn("favorites_count", gorm.Expr("favorites_count - ?", 1)).Error; err != nil {
			return err
		}

		return nil
	})
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

	if err != nil {
		return posts, total, err
	}

	// 由于这些帖子都是从收藏表中获取的，所以它们都是已收藏的
	for i := range posts {
		posts[i].IsCollected = true
	}

	return posts, total, err
}

// GetPostCollectionCount 获取帖子收藏数
func (s *CollectionService) GetPostCollectionCount(postID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.Model(&models.PostCollection{}).Where("post_id = ?", postID).Count(&count).Error
	return count, err
}
