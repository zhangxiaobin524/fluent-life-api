package handlers

import (
	"fluent-life-backend/internal/services"
	"fluent-life-backend/internal/utils"
	"fluent-life-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AchievementHandler struct {
	db                *gorm.DB
	achievementService *services.AchievementService
}

func NewAchievementHandler(db *gorm.DB) *AchievementHandler {
	return &AchievementHandler{
		db:                 db,
		achievementService: services.NewAchievementService(db),
	}
}

func (h *AchievementHandler) GetAchievements(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	achievements, err := h.achievementService.GetAchievements(userID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	response.Success(c, achievements, "获取成功")
}







