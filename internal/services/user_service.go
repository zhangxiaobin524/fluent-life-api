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








