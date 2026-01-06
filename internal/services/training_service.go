package services

import (
	"fmt"
	"math/rand"
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
		Select("COUNT(*)").
		Scan(&totalSessions).Error
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

func (s *TrainingService) GetMeditationProgress(userID uuid.UUID) (map[string]interface{}, error) {
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

	// 构建返回数据结构
	unlockedStages := []int{}
	progressDays := make(map[int]int)

	for _, p := range progress {
		if p.Unlocked {
			unlockedStages = append(unlockedStages, p.Stage)
		}
		progressDays[p.Stage] = p.CompletedDays
	}

	// 确保至少有阶段1解锁
	stage1Found := false
	for _, stage := range unlockedStages {
		if stage == 1 {
			stage1Found = true
			break
		}
	}
	if !stage1Found {
		unlockedStages = append([]int{1}, unlockedStages...)
		if progressDays[1] == 0 {
			progressDays[1] = 0
		}
	}

	return map[string]interface{}{
		"unlocked_stages": unlockedStages,
		"progress_days":   progressDays,
	}, nil
}

// GetWeeklyStats 获取用户过去7天的训练统计
func (s *TrainingService) GetWeeklyStats(userID uuid.UUID) ([]map[string]interface{}, error) {
	var weeklyStats []map[string]interface{}
	
	// 获取过去7天的日期
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		var count int64
		s.db.Model(&models.TrainingRecord{}).
			Where("user_id = ? AND DATE(timestamp) = DATE(?)", userID, date).
			Count(&count)
		
		weeklyStats = append(weeklyStats, map[string]interface{}{
			"date":  date.Format("2006-01-02"),
			"count": count,
		})
	}
	
	return weeklyStats, nil
}

// GetSkillLevels 获取用户的技能水平
func (s *TrainingService) GetSkillLevels(userID uuid.UUID) (map[string]int, error) {
	skillLevels := make(map[string]int)
	
	// 定义技能类型
	skillTypes := []string{"meditation", "airflow", "exposure", "practice"}
	
	for _, skillType := range skillTypes {
		var totalDuration int
		s.db.Model(&models.TrainingRecord{}).
			Where("user_id = ? AND type = ?", userID, skillType).
			Select("COALESCE(SUM(duration), 0)").
			Scan(&totalDuration)
		
		// 简单的技能水平计算：每600秒（10分钟）训练增加1级
		skillLevels[skillType] = totalDuration / 600
	}
	
	return skillLevels, nil
}

// Recommendation 结构体定义
type Recommendation struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"` // meditation | airflow | exposure | practice
}

// GetRecommendations 获取个性化推荐
func (s *TrainingService) GetRecommendations(userID uuid.UUID) ([]Recommendation, error) {
	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	allPossibleRecommendations := []Recommendation{
		{
			ID:          "rec1",
			Title:       "深度冥想：放松身心",
			Description: "尝试一次15分钟的深度冥想，帮助你缓解压力，提升专注力。",
			Type:        "meditation",
		},
		{
			ID:          "rec2",
			Title:       "气流练习：掌握呼吸",
			Description: "进行10分钟的气流控制练习，改善你的发声技巧和气息稳定性。",
			Type:        "airflow",
		},
		{
			ID:          "rec3",
			Title:       "情景对话：勇敢开口",
			Description: "参与一次模拟日常对话的练习，提升你在真实场景中的表达自信。",
			Type:        "exposure",
		},
		{
			ID:          "rec4",
			Title:       "自由朗读：提升语感",
			Description: "选择一篇你感兴趣的文章进行自由朗读，培养语感和表达流畅度。",
			Type:        "practice",
		},
		{
			ID:          "rec5",
			Title:       "早晨冥想：开启活力",
			Description: "每天早晨进行5分钟冥想，帮助你清醒头脑，迎接新的一天。",
			Type:        "meditation",
		},
		{
			ID:          "rec6",
			Title:       "腹式呼吸：缓解焦虑",
			Description: "学习并练习腹式呼吸，有效缓解紧张和焦虑情绪。",
			Type:        "airflow",
		},
		{
			ID:          "rec7",
			Title:       "角色扮演：提升口语",
			Description: "选择一个角色进行扮演，模拟真实场景对话，提升口语表达能力。",
			Type:        "exposure",
		},
		{
			ID:          "rec8",
			Title:       "听力训练：磨练耳朵",
			Description: "每天听一段英文播客或新闻，提高听力理解和语速适应能力。",
			Type:        "practice",
		},
		{
			ID:          "rec9",
			Title:       "睡前冥想：改善睡眠",
			Description: "睡前进行10分钟冥想，帮助你放松身心，获得更好的睡眠质量。",
			Type:        "meditation",
		},
		{
			ID:          "rec10",
			Title:       "发音纠正：标准发音",
			Description: "针对特定音标进行发音练习，确保你的发音清晰准确。",
			Type:        "airflow",
		},
	}

	// 获取用户技能水平
	skillLevels, err := s.GetSkillLevels(userID)
	if err != nil {
		// 如果获取技能水平失败，返回随机推荐
		numRecommendations := rand.Intn(3) + 3 // 3, 4, or 5
		rand.Shuffle(len(allPossibleRecommendations), func(i, j int) {
			allPossibleRecommendations[i], allPossibleRecommendations[j] = allPossibleRecommendations[j], allPossibleRecommendations[i]
		})
		return allPossibleRecommendations[:numRecommendations], nil
	}

	var weakestSkill string
	minLevel := -1

	// 找到最弱的技能
	for skillType, level := range skillLevels {
		if minLevel == -1 || level < minLevel {
			minLevel = level
			weakestSkill = skillType
		}
	}

	var personalizedRecommendations []Recommendation
	if weakestSkill != "" {
		// 优先推荐最弱技能相关的练习
		for _, rec := range allPossibleRecommendations {
			if rec.Type == weakestSkill {
				personalizedRecommendations = append(personalizedRecommendations, rec)
			}
		}
	}

	// 补充其他推荐，直到达到3-5个
	numRecommendations := rand.Intn(3) + 3 // 3, 4, or 5
	
	// 打乱所有可能的推荐，以便随机选择补充
	rand.Shuffle(len(allPossibleRecommendations), func(i, j int) {
		allPossibleRecommendations[i], allPossibleRecommendations[j] = allPossibleRecommendations[j], allPossibleRecommendations[i]
	})

	for _, rec := range allPossibleRecommendations {
		if len(personalizedRecommendations) >= numRecommendations {
			break
		}
		// 避免重复添加
		found := false
		for _, pRec := range personalizedRecommendations {
			if pRec.ID == rec.ID {
				found = true
				break
			}
		}
		if !found {
			personalizedRecommendations = append(personalizedRecommendations, rec)
		}
	}

	// 如果最终推荐数量超过numRecommendations，则截断
	if len(personalizedRecommendations) > numRecommendations {
		personalizedRecommendations = personalizedRecommendations[:numRecommendations]
	}

	// 如果推荐数量不足，再次打乱并补充
	if len(personalizedRecommendations) < numRecommendations {
		remaining := numRecommendations - len(personalizedRecommendations)
		var tempAll []Recommendation
		for _, rec := range allPossibleRecommendations {
			found := false
			for _, pRec := range personalizedRecommendations {
				if pRec.ID == rec.ID {
					found = true
					break
				}
			}
			if !found {
				tempAll = append(tempAll, rec)
			}
		}
		rand.Shuffle(len(tempAll), func(i, j int) {
			tempAll[i], tempAll[j] = tempAll[j], tempAll[i]
		})
		if len(tempAll) > remaining {
			personalizedRecommendations = append(personalizedRecommendations, tempAll[:remaining]...)
		} else {
			personalizedRecommendations = append(personalizedRecommendations, tempAll...)
		}
	}

	return personalizedRecommendations, nil
}

// ProgressTrendData 结构体定义
type ProgressTrendData struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

// GetProgressTrend 获取用户的进步趋势
func (s *TrainingService) GetProgressTrend(userID uuid.UUID) ([]ProgressTrendData, error) {
	var trendData []ProgressTrendData
	
	// 获取过去30天的趋势数据
	for i := 29; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		var totalDuration int
		s.db.Model(&models.TrainingRecord{}).
			Where("user_id = ? AND DATE(timestamp) = DATE(?)", userID, date).
			Select("COALESCE(SUM(duration), 0)").
			Scan(&totalDuration)
		
		trendData = append(trendData, ProgressTrendData{
			Date:  date.Format("2006-01-02"),
			Value: totalDuration / 60, // 转换为分钟
		})
	}
	
	return trendData, nil
}

// LearningPartnerStats 结构体定义
type LearningPartnerStats struct {
	OnlineCount int `json:"online_count"`
	TodayActive int `json:"today_active"`
}

// GetLearningPartnerStats 获取学习伙伴统计数据
func (s *TrainingService) GetLearningPartnerStats(userID uuid.UUID) (LearningPartnerStats, error) {
	var onlineCount int64
	// 在线用户：过去30分钟内有登录记录的用户
	if err := s.db.Model(&models.User{}).
		Where("last_login_at IS NOT NULL AND last_login_at > ?", time.Now().Add(-30*time.Minute)).
		Count(&onlineCount).Error; err != nil {
		return LearningPartnerStats{}, err
	}

	var todayActive int64
	// 今日活跃用户：今天有训练记录的独立用户
	today := time.Now().Format("2006-01-02")
	if err := s.db.Model(&models.TrainingRecord{}).
		Where("DATE(timestamp) = ?", today).
		Distinct("user_id").
		Count(&todayActive).Error; err != nil {
		return LearningPartnerStats{}, err
	}

	return LearningPartnerStats{
		OnlineCount: int(onlineCount),
		TodayActive: int(todayActive),
	}, nil
}

// LearningPartner 结构体定义
type LearningPartner struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Status   string `json:"status"`   // "online" | "practicing" | "offline"
	Activity string `json:"activity"` // e.g., "正在练习气流"
	Progress int    `json:"progress"` // 0-100
}

// GetLearningPartners 获取学习伙伴列表
func (s *TrainingService) GetLearningPartners(userID uuid.UUID) ([]LearningPartner, error) {
	var users []models.User
	// Fetch a random set of users, excluding the current user
	if err := s.db.Where("id != ?", userID).Order("RANDOM()").Limit(5).Find(&users).Error; err != nil {
		return nil, err
	}

	var partners []LearningPartner
	for _, user := range users {
		fmt.Printf("Processing learning partner: UserID=%s, Username=%s\n", user.ID, user.Username)
		status := "offline"
		activity := "暂无动态"
		progress := 0

		var latestRecord models.TrainingRecord
		// Find the most recent training record for the user
		s.db.Where("user_id = ?", user.ID).Order("timestamp DESC").First(&latestRecord)

		if latestRecord.ID != uuid.Nil { // If a training record exists
			fmt.Printf("  Found latest record for %s: Type=%s, Timestamp=%s\n", user.Username, latestRecord.Type, latestRecord.Timestamp)
			timeSinceLastActivity := time.Since(latestRecord.Timestamp)

			if timeSinceLastActivity < 10*time.Minute {
				status = "practicing"
				switch latestRecord.Type {
				case "meditation":
					activity = "冥想练习中"
				case "airflow":
					activity = "正在练习气流"
				case "exposure":
					activity = "脱敏训练"
				case "practice":
					activity = "实战练习"
				default:
					activity = "正在练习"
				}
				rand.Seed(time.Now().UnixNano() + int64(user.ID.ID()))
				progress = rand.Intn(100) // Random progress for practicing users
			} else if timeSinceLastActivity < 1*time.Hour {
				status = "online"
				activity = "在线"
			} else {
				status = "offline"
				activity = "上次练习: " + formatDuration(timeSinceLastActivity) + "前"
			}
		} else {
			fmt.Printf("  No training record found for %s. Checking last login.\n", user.Username)
			// If no training record, check last login for online status
			if user.LastLoginAt != nil && time.Since(*user.LastLoginAt) < 30*time.Minute {
				status = "online"
				activity = "在线"
			}
		}
		fmt.Printf("  Final status for %s: Status=%s, Activity=%s\n", user.Username, status, activity)

		// Ensure AvatarURL is not nil before dereferencing, and provide a fallback
		avatar := ""
		if user.Username != "" {
			avatar = string([]rune(user.Username)[0])
		}

		partner := LearningPartner{
			ID:       int(user.CreatedAt.Unix()), // Using CreatedAt as a pseudo-ID for now
			Name:     user.Username,
			Avatar:   avatar,
			Status:   status,
			Activity: activity,
			Progress: progress,
		}
		partners = append(partners, partner)
	}

	return partners, nil
}

// Helper function to format duration
func formatDuration(d time.Duration) string {
	if d.Hours() >= 24 {
		return fmt.Sprintf("%d天", int(d.Hours()/24))
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%d小时", int(d.Hours()))
	}
	if d.Minutes() >= 1 {
		return fmt.Sprintf("%d分钟", int(d.Minutes()))
	}
	return "刚刚"
}

