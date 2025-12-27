package services

import (
	"fluent-life-backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AchievementService struct {
	db *gorm.DB
}

func NewAchievementService(db *gorm.DB) *AchievementService {
	return &AchievementService{db: db}
}

type AchievementInfo struct {
	ID              string `json:"id"`
	AchievementType string `json:"achievement_type"`
	Title           string `json:"title"`
	Icon            string `json:"icon"`
	Desc            string `json:"desc"`
	Unlocked        bool   `json:"unlocked"`
	UnlockedAt       *string `json:"unlocked_at,omitempty"`
}

var achievementDefinitions = map[string]struct {
	Title string
	Icon  string
	Desc  string
}{
	"first_meditation": {
		Title: "静谧之心",
		Icon:  "🧘",
		Desc:  "完成首次冥想",
	},
	"airflow_master": {
		Title: "气流大师",
		Icon:  "🌬️",
		Desc:  "掌握起音技巧",
	},
	"courage_light": {
		Title: "勇气之光",
		Icon:  "🔥",
		Desc:  "完成社会挑战",
	},
}

func (s *AchievementService) GetAchievements(userID uuid.UUID) ([]AchievementInfo, error) {
	var unlockedAchievements []models.Achievement
	s.db.Where("user_id = ?", userID).Find(&unlockedAchievements)

	unlockedMap := make(map[string]bool)
	for _, ach := range unlockedAchievements {
		unlockedMap[ach.AchievementType] = true
	}

	var result []AchievementInfo
	for achievementType, def := range achievementDefinitions {
		info := AchievementInfo{
			ID:              achievementType,
			AchievementType: achievementType,
			Title:           def.Title,
			Icon:            def.Icon,
			Desc:            def.Desc,
			Unlocked:        unlockedMap[achievementType],
		}

		if info.Unlocked {
			var achievement models.Achievement
			if err := s.db.Where("user_id = ? AND achievement_type = ?", userID, achievementType).First(&achievement).Error; err == nil {
				unlockedAt := achievement.UnlockedAt.Format("2006-01-02 15:04:05")
				info.UnlockedAt = &unlockedAt
			}
		}

		result = append(result, info)
	}

	return result, nil
}







