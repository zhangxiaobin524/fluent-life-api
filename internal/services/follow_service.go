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
func (s *FollowService) FollowUser(followerID, followingID uuid.UUID) error {
	// 不能关注自己
	if followerID == followingID {
		return gorm.ErrInvalidTransaction
	}

	// 检查是否已经关注
	var existingFollow models.UserFollow
	err := s.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).First(&existingFollow).Error
	if err == nil {
		return nil // 已经关注，直接返回成功
	}

	// 创建关注关系
	follow := models.UserFollow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}

	return s.db.Create(&follow).Error
}

// UnfollowUser 取消关注
func (s *FollowService) UnfollowUser(followerID, followingID uuid.UUID) error {
	return s.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).Delete(&models.UserFollow{}).Error
}

// IsFollowing 检查是否已关注
func (s *FollowService) IsFollowing(followerID, followingID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&models.UserFollow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error
	return count > 0, err
}

// GetFollowers 获取粉丝列表
func (s *FollowService) GetFollowers(userID uuid.UUID, page, pageSize int) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// 计算总数
	s.db.Model(&models.UserFollow{}).Where("following_id = ?", userID).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.Table("users").
		Select("users.*").
		Joins("INNER JOIN user_follows ON users.id = user_follows.follower_id").
		Where("user_follows.following_id = ?", userID).
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
	s.db.Model(&models.UserFollow{}).Where("follower_id = ?", userID).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := s.db.Table("users").
		Select("users.*").
		Joins("INNER JOIN user_follows ON users.id = user_follows.following_id").
		Where("user_follows.follower_id = ?", userID).
		Offset(offset).
		Limit(pageSize).
		Find(&users).Error

	return users, total, err
}

// GetFollowCount 获取关注数和粉丝数
func (s *FollowService) GetFollowCount(userID uuid.UUID) (followersCount, followingCount int64, err error) {
	// 粉丝数
	err = s.db.Model(&models.UserFollow{}).Where("following_id = ?", userID).Count(&followersCount).Error
	if err != nil {
		return 0, 0, err
	}

	// 关注数
	err = s.db.Model(&models.UserFollow{}).Where("follower_id = ?", userID).Count(&followingCount).Error
	return followersCount, followingCount, err
}
