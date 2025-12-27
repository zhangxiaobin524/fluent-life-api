package services

import (
	"time"

	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
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







