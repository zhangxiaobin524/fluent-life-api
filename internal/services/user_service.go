package services

import (
	"time"

	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	db                 *gorm.DB
	trainingService    *TrainingService
	achievementService *AchievementService
}

// NewUserService 创建一个新的 UserService 实例
func NewUserService(db *gorm.DB, cfg *config.Config) *UserService {
	return &UserService{
		db:                 db,
		trainingService:    NewTrainingService(db, cfg),
		achievementService: NewAchievementService(db),
	}
}

type UserStats struct {
	TotalMinutes    int    `json:"total_minutes"`
	TotalDays       int    `json:"total_days"`
	CurrentLevel    int    `json:"current_level"`
	LevelProgress   int    `json:"level_progress"`
	WeeklyData      []int  `json:"weekly_data"`
}

func (s *UserService) CalculateStats(userID uuid.UUID) (*UserStats, error) {
	// 计算总训练时长（秒）
	var totalDuration int
	s.db.Model(&models.TrainingRecord{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(duration), 0)").
		Scan(&totalDuration)

	totalMinutes := totalDuration / 60

	// 计算总天数（去重）
	var totalDays int
	s.db.Model(&models.TrainingRecord{}).
		Where("user_id = ?", userID).
		Select("COUNT(DISTINCT DATE(timestamp))").
		Scan(&totalDays)

	// 计算等级和进度
	currentLevel := (totalMinutes / 60) + 1
	levelProgress := (totalMinutes % 60) * 100 / 60

	// 计算最近7天的每日训练时长
	weeklyData := make([]int, 7)
	now := time.Now()
	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayDuration int
		s.db.Model(&models.TrainingRecord{}).
			Where("user_id = ? AND timestamp >= ? AND timestamp < ?", userID, startOfDay, endOfDay).
			Select("COALESCE(SUM(duration), 0)").
			Scan(&dayDuration)

		weeklyData[6-i] = dayDuration / 60 // 转换为分钟
	}

	return &UserStats{
		TotalMinutes:  totalMinutes,
		TotalDays:     totalDays,
		CurrentLevel:   currentLevel,
		LevelProgress: levelProgress,
		WeeklyData:    weeklyData,
	}, nil
}

// GetWeeklyActivity 计算用户本周活跃度（例如，过去7天每天的训练时长）
func (s *UserService) GetWeeklyActivity(userID uuid.UUID) ([]int, error) {
	weeklyData := make([]int, 7)
	now := time.Now()
	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dayDuration int
		s.db.Model(&models.TrainingRecord{}).
			Where("user_id = ? AND timestamp >= ? AND timestamp < ?", userID, startOfDay, endOfDay).
			Select("COALESCE(SUM(duration), 0)").
			Scan(&dayDuration)

		weeklyData[6-i] = dayDuration / 60 // 转换为分钟
	}
	return weeklyData, nil
}

func (s *UserService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserProfileWithStats 获取用户资料及统计数据
func (s *UserService) GetUserProfileWithStats(userID uuid.UUID) (*models.UserProfile, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}

	totalTrainingDays, err := s.trainingService.GetTotalTrainingDays(userID)
	if err != nil {
		return nil, err
	}

	totalTrainingSessions, err := s.trainingService.GetTotalTrainingSessions(userID)
	if err != nil {
		return nil, err
	}

	totalTrainingMinutes, err := s.trainingService.GetTotalTrainingMinutes(userID)
	if err != nil {
		return nil, err
	}

	braveryBadges, err := s.achievementService.GetUserBadges(userID)
	if err != nil {
		return nil, err
	}

	weeklyActivity, err := s.GetWeeklyActivity(userID)
	if err != nil {
		return nil, err
	}

	userProfile := &models.UserProfile{
		User:                user,
		TotalTrainingDays:   totalTrainingDays,
		TotalTrainingSessions: totalTrainingSessions,
		TotalTrainingMinutes: totalTrainingMinutes,
		BraveryBadges:       braveryBadges,
		WeeklyActivity:      weeklyActivity,
	}

	return userProfile, nil
}

// FollowUser 处理用户关注逻辑
func (s *UserService) FollowUser(followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return gorm.ErrInvalidData // 用户不能关注自己
	}

	// 检查是否已关注
	var existingFollow models.Follow
	err := s.db.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).First(&existingFollow).Error
	if err == nil {
		return nil // 已经关注，无需重复操作
	}
	if err != gorm.ErrRecordNotFound {
		return err // 其他数据库错误
	}

	// 创建关注记录
	follow := models.Follow{
		FollowerID: followerID,
		FolloweeID: followeeID,
	}
	if err := s.db.Create(&follow).Error; err != nil {
		return err
	}

	// 更新关注者和被关注者的计数
	s.db.Model(&models.User{}).Where("id = ?", followerID).Update("following_count", gorm.Expr("following_count + ?", 1))
	s.db.Model(&models.User{}).Where("id = ?", followeeID).Update("followers_count", gorm.Expr("followers_count + ?", 1))

	return nil
}

// UnfollowUser 处理用户取关逻辑
func (s *UserService) UnfollowUser(followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return gorm.ErrInvalidData // 用户不能取关自己
	}

	// 删除关注记录
	result := s.db.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).Delete(&models.Follow{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil // 没有关注关系，无需操作
	}

	// 更新关注者和被关注者的计数
	s.db.Model(&models.User{}).Where("id = ?", followerID).Update("following_count", gorm.Expr("following_count - ?", 1))
	s.db.Model(&models.User{}).Where("id = ?", followeeID).Update("followers_count", gorm.Expr("followers_count - ?", 1))

	return nil
}

// GetFollowers 获取用户的粉丝列表
func (s *UserService) GetFollowers(userID uuid.UUID, page, pageSize int) ([]models.User, int64, error) {
	var followers []models.User
	var total int64

	// 获取粉丝ID
	var followerIDs []uuid.UUID
	s.db.Model(&models.Follow{}).Where("followee_id = ?", userID).Pluck("follower_id", &followerIDs)

	if len(followerIDs) == 0 {
		return []models.User{}, 0, nil
	}

	// 获取粉丝用户详情
	query := s.db.Model(&models.User{}).Where("id IN (?)", followerIDs)
	query.Count(&total)
	query.Limit(pageSize).Offset((page - 1) * pageSize).Find(&followers)

	return followers, total, nil
}

// GetFollowing 获取用户关注的人列表
func (s *UserService) GetFollowing(userID uuid.UUID, page, pageSize int) ([]models.User, int64, error) {
	var following []models.User
	var total int64

	// 获取关注的人的ID
	var followeeIDs []uuid.UUID
	s.db.Model(&models.Follow{}).Where("follower_id = ?", userID).Pluck("followee_id", &followeeIDs)

	if len(followeeIDs) == 0 {
		return []models.User{}, 0, nil
	}

	// 获取关注的人的用户详情
	query := s.db.Model(&models.User{}).Where("id IN (?)", followeeIDs)
	query.Count(&total)
	query.Limit(pageSize).Offset((page - 1) * pageSize).Find(&following)

	return following, total, nil
}

// IsFollowing 检查一个用户是否关注了另一个用户
func (s *UserService) IsFollowing(followerID, followeeID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&models.Follow{}).Where("follower_id = ? AND followee_id = ?", followerID, followeeID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}








