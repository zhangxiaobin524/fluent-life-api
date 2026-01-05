package services

import (
	"time"

	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TrainingService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewTrainingService(db *gorm.DB, cfg *config.Config) *TrainingService {
	return &TrainingService{db: db, cfg: cfg}
}

func (s *TrainingService) CreateRecord(userID uuid.UUID, recordType string, duration int, data models.JSONB, timestamp time.Time) (*models.TrainingRecord, error) {
	record := models.TrainingRecord{
		UserID:    userID,
		Type:      recordType,
		Duration:  duration,
		Data:      data,
		Timestamp: timestamp,
	}

	if err := s.db.Create(&record).Error; err != nil {
		return nil, err
	}

	// 如果是冥想记录，更新冥想进度
	if recordType == "meditation" {
		s.updateMeditationProgress(userID, data, duration)
	}

	// 检查并解锁成就
	s.checkAndUnlockAchievements(userID, recordType)

	return &record, nil
}

func (s *TrainingService) updateMeditationProgress(userID uuid.UUID, data models.JSONB, duration int) {
	stage, ok := data["stage"].(float64)
	if !ok {
		return
	}
	stageInt := int(stage)

	// 获取目标时长（秒）
	targetDurations := map[int]int{1: 300, 2: 720, 3: 30}
	targetDuration := targetDurations[stageInt]
	if duration < targetDuration {
		return // 未达到100%目标时长，不计入有效天数
	}

	// 检查今天是否已有该阶段的记录
	var todayRecord models.TrainingRecord
	today := time.Now().Format("2006-01-02")
	s.db.Where("user_id = ? AND type = 'meditation' AND DATE(timestamp) = ?", userID, today).
		Where("data->>'stage' = ?", stageInt).
		First(&todayRecord)

	if todayRecord.ID != uuid.Nil {
		return // 今天已有记录，不重复计算
	}

	// 更新或创建进度记录
	var progress models.MeditationProgress
	err := s.db.Where("user_id = ? AND stage = ?", userID, stageInt).First(&progress).Error
	if err == gorm.ErrRecordNotFound {
		progress = models.MeditationProgress{
			UserID:        userID,
			Stage:         stageInt,
			CompletedDays: 1,
			Unlocked:      stageInt == 1, // 阶段1默认解锁
		}
		s.db.Create(&progress)
	} else {
		progress.CompletedDays++
		s.db.Save(&progress)
	}

	// 检查解锁条件
	if stageInt == 1 && progress.CompletedDays >= 14 {
		// 解锁阶段2
		var stage2Progress models.MeditationProgress
		if err := s.db.Where("user_id = ? AND stage = ?", userID, 2).First(&stage2Progress).Error; err == gorm.ErrRecordNotFound {
			stage2Progress = models.MeditationProgress{
				UserID:        userID,
				Stage:         2,
				CompletedDays: 0,
				Unlocked:      true,
			}
			s.db.Create(&stage2Progress)
		} else {
			stage2Progress.Unlocked = true
			s.db.Save(&stage2Progress)
		}
	}

	if stageInt == 2 && progress.CompletedDays >= 14 {
		// 检查阶段1是否也完成14天
		var stage1Progress models.MeditationProgress
		if err := s.db.Where("user_id = ? AND stage = ?", userID, 1).First(&stage1Progress).Error; err == nil && stage1Progress.CompletedDays >= 14 {
			// 解锁阶段3
			var stage3Progress models.MeditationProgress
			if err := s.db.Where("user_id = ? AND stage = ?", userID, 3).First(&stage3Progress).Error; err == gorm.ErrRecordNotFound {
				stage3Progress = models.MeditationProgress{
					UserID:        userID,
					Stage:         3,
					CompletedDays: 0,
					Unlocked:      true,
				}
				s.db.Create(&stage3Progress)
			} else {
				stage3Progress.Unlocked = true
				s.db.Save(&stage3Progress)
			}
		}
	}
}

func (s *TrainingService) checkAndUnlockAchievements(userID uuid.UUID, recordType string) {
	achievementMap := map[string]string{
		"meditation": "first_meditation",
		"airflow":    "airflow_master",
		"exposure":   "courage_light",
	}

	achievementType, ok := achievementMap[recordType]
	if !ok {
		return
	}

	// 检查是否已解锁
	var existing models.Achievement
	if err := s.db.Where("user_id = ? AND achievement_type = ?", userID, achievementType).First(&existing).Error; err == nil {
		return // 已解锁
	}

	// 解锁成就
	achievement := models.Achievement{
		UserID:          userID,
		AchievementType: achievementType,
		UnlockedAt:      time.Now(),
	}
	s.db.Create(&achievement)
}

func (s *TrainingService) GetRecords(userID uuid.UUID, page, pageSize int, recordType string) ([]models.TrainingRecord, int64, error) {
	var records []models.TrainingRecord
	var total int64

	query := s.db.Where("user_id = ?", userID)
	if recordType != "" {
		query = query.Where("type = ?", recordType)
	}

	query.Model(&models.TrainingRecord{}).Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Order("timestamp DESC").Offset(offset).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

func (s *TrainingService) GetStats(userID uuid.UUID) (map[string]interface{}, error) {
	var totalDuration int
	s.db.Model(&models.TrainingRecord{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(duration), 0)").
		Scan(&totalDuration)

	var totalDays int
	s.db.Model(&models.TrainingRecord{}).
		Where("user_id = ?", userID).
		Select("COUNT(DISTINCT DATE(timestamp))").
		Scan(&totalDays)

	return map[string]interface{}{
		"total_minutes": totalDuration / 60,
		"total_days":    totalDays,
	}, nil
}

// GetTotalTrainingDays 获取用户总锻炼天数
func (s *TrainingService) GetTotalTrainingDays(userID uuid.UUID) (int, error) {
	var totalDays int
	err := s.db.Model(&models.TrainingRecord{}).
		Where("user_id = ?", userID).
		Select("COUNT(DISTINCT DATE(timestamp))").
		Scan(&totalDays).Error
	return totalDays, err
}

// GetTotalTrainingCounts 获取用户总锻炼次数
func (s *TrainingService) GetTotalTrainingCounts(userID uuid.UUID) (int, error) {
	var totalCounts int
	err := s.db.Model(&models.TrainingRecord{}).
		Where("user_id = ?", userID).
		Select("COUNT(*)").
		Scan(&totalCounts).Error
	return totalCounts, err
}

// GetTotalTrainingSessions 获取用户总锻炼会话次数
func (s *TrainingService) GetTotalTrainingSessions(userID uuid.UUID) (int64, error) {
	var totalSessions int64
	err := s.db.Model(&models.TrainingRecord{}).
		Where("user_id = ?", userID).
		Count(&totalSessions).Error
	return totalSessions, err
}

// GetTotalTrainingMinutes 获取用户总锻炼分钟数
func (s *TrainingService) GetTotalTrainingMinutes(userID uuid.UUID) (int, error) {
	var totalDurationSeconds int
	err := s.db.Model(&models.TrainingRecord{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(duration), 0)").
		Scan(&totalDurationSeconds).Error
	return totalDurationSeconds / 60, err
}

func (s *TrainingService) GetMeditationProgress(userID uuid.UUID) ([]models.MeditationProgress, error) {
	var progress []models.MeditationProgress
	if err := s.db.Where("user_id = ?", userID).Order("stage ASC").Find(&progress).Error; err != nil {
		return nil, err
	}

	// 如果没有任何进度记录，创建阶段1的默认记录
	if len(progress) == 0 {
		progress = []models.MeditationProgress{
			{
				UserID:        userID,
				Stage:         1,
				CompletedDays: 0,
				Unlocked:      true,
			},
		}
	}

	return progress, nil
}

