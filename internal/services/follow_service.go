package services

import (
	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FollowService struct {
	db *gorm.DB
}

func NewFollowService(db *gorm.DB) *FollowService {
	return &FollowService{db: db}
}

// FollowUser 关注用户
func (s *FollowService) FollowUser(followerID, followeeID uuid.UUID) error {
	// 不能关注自己
	if followerID == followeeID {
		return gorm.ErrInvalidTransaction
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查是否已经关注
		var existingFollow models.Follow
		err := tx.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).First(&existingFollow).Error
		if err == nil {
			return nil // 已经关注，直接返回成功
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}

		// 创建关注关系
		follow := models.Follow{
			FollowerID:  followerID,
			FolloweeID: followeeID,
		}
		if err := tx.Create(&follow).Error; err != nil {
			return err
		}

		// 更新关注者的 following_count
		if err := tx.Model(&models.User{}).Where("id = ?", followerID).Update("following_count", gorm.Expr("following_count + ?", 1)).Error; err != nil {
			return err
		}

		// 更新被关注者的 followers_count
		if err := tx.Model(&models.User{}).Where("id = ?", followeeID).Update("followers_count", gorm.Expr("followers_count + ?", 1)).Error; err != nil {
			return err
		}

		return nil
	})
}

// UnfollowUser 取消关注
func (s *FollowService) UnfollowUser(followerID, followeeID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查是否已经关注
		var existingFollow models.Follow
		err := tx.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).First(&existingFollow).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil // 未关注，直接返回成功
			}
			return err
		}

		// 删除关注关系
		if err := tx.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).Delete(&models.Follow{}).Error; err != nil {
			return err
		}

		// 更新关注者的 following_count
		if err := tx.Model(&models.User{}).Where("id = ?", followerID).Update("following_count", gorm.Expr("following_count - ?", 1)).Error; err != nil {
			return err
		}

		// 更新被关注者的 followers_count
		if err := tx.Model(&models.User{}).Where("id = ?", followeeID).Update("followers_count", gorm.Expr("followers_count - ?", 1)).Error; err != nil {
			return err
		}

		return nil
	})
}

// IsFollowing 检查是否已关注
func (s *FollowService) IsFollowing(followerID, followeeID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&models.Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Count(&count).Error
	return count > 0, err
}

// GetFollowers 获取粉丝列表
func (s *FollowService) GetFollowers(userID uuid.UUID, page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// 计算总数
	s.db.Model(&models.Follow{}).Where("followee_id = ?", userID).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.Table("users").
		Select("users.*").
		Joins("INNER JOIN follows ON users.id = follows.follower_id").
		Where("follows.followee_id = ?", userID).
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

// GetFollowing 获取关注列表
func (s *FollowService) GetFollowing(userID uuid.UUID, page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// 计算总数
	s.db.Model(&models.Follow{}).Where("follower_id = ?", userID).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.Table("users").
		Select("users.*").
		Joins("INNER JOIN follows ON users.id = follows.followee_id").
		Where("follows.follower_id = ?", userID).
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

// GetFollowCount 获取关注数和粉丝数
func (s *FollowService) GetFollowCount(userID uuid.UUID) (followersCount, followingCount int64, err error) {
	// 粉丝数
	err = s.db.Model(&models.Follow{}).Where("followee_id = ?", userID).Count(&followersCount).Error
	if err != nil {
		return 0, 0, err
	}

	// 关注数
	err = s.db.Model(&models.Follow{}).Where("follower_id = ?", userID).Count(&followingCount).Error
	return followersCount, followingCount, err
}
