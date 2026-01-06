package handlers

import (
	"strconv"
	"time"

	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/models"
	"fluent-life-backend/internal/services"
	"fluent-life-backend/internal/utils"
	"fluent-life-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TrainingHandler struct {
	db              *gorm.DB
	trainingService *services.TrainingService
}

func NewTrainingHandler(db *gorm.DB, cfg *config.Config) *TrainingHandler {
	return &TrainingHandler{
		db:              db,
		trainingService: services.NewTrainingService(db, cfg),
	}
}

type CreateRecordRequest struct {
	Type      string                 `json:"type" binding:"required,oneof=meditation airflow exposure practice"`
	Duration  int                    `json:"duration" binding:"required,min=1"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp *time.Time             `json:"timestamp,omitempty"`
}

func (h *TrainingHandler) CreateRecord(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	var req CreateRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	timestamp := time.Now()
	if req.Timestamp != nil {
		timestamp = *req.Timestamp
	}

	data := models.JSONB(req.Data)
	record, err := h.trainingService.CreateRecord(userID, req.Type, req.Duration, data, timestamp)
	if err != nil {
		response.InternalError(c, "创建记录失败")
		return
	}

	response.Success(c, record, "创建成功")
}

func (h *TrainingHandler) GetRecords(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	recordType := c.Query("type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	records, total, err := h.trainingService.GetRecords(userID, page, pageSize, recordType)
	if err != nil {
		response.InternalError(c, "获取记录失败")
		return
	}

	response.Success(c, gin.H{
		"records": records,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
	}, "获取成功")
}

func (h *TrainingHandler) GetStats(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	stats, err := h.trainingService.GetStats(userID)
	if err != nil {
		response.InternalError(c, "获取统计失败")
		return
	}

	response.Success(c, stats, "获取成功")
}

func (h *TrainingHandler) GetMeditationProgress(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	progress, err := h.trainingService.GetMeditationProgress(userID)
	if err != nil {
		response.InternalError(c, "获取进度失败")
		return
	}

	response.Success(c, progress, "获取成功")
}

func (h *TrainingHandler) GetWeeklyStats(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	weeklyStats, err := h.trainingService.GetWeeklyStats(userID)
	if err != nil {
		response.InternalError(c, "获取周统计失败")
		return
	}

	response.Success(c, gin.H{"weekly_stats": weeklyStats}, "获取成功")
}

func (h *TrainingHandler) GetSkillLevels(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	skillLevels, err := h.trainingService.GetSkillLevels(userID)
	if err != nil {
		response.InternalError(c, "获取技能水平失败")
		return
	}

	response.Success(c, gin.H{"skill_levels": skillLevels}, "获取成功")
}

func (h *TrainingHandler) GetRecommendations(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	recommendations, err := h.trainingService.GetRecommendations(userID)
	if err != nil {
		response.InternalError(c, "获取推荐失败")
		return
	}

	response.Success(c, gin.H{"recommendations": recommendations}, "获取成功")
}

func (h *TrainingHandler) GetProgressTrend(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	trendData, err := h.trainingService.GetProgressTrend(userID)
	if err != nil {
		response.InternalError(c, "获取进步趋势失败")
		return
	}

	response.Success(c, gin.H{"trend_data": trendData}, "获取成功")
}

func (h *TrainingHandler) GetLearningPartnerStats(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	stats, err := h.trainingService.GetLearningPartnerStats(userID)
	if err != nil {
		response.InternalError(c, "获取学习伙伴统计失败")
		return
	}

	response.Success(c, gin.H{"learning_partner_stats": stats}, "获取成功")
}

func (h *TrainingHandler) GetLearningPartners(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	partners, err := h.trainingService.GetLearningPartners(userID)
	if err != nil {
		response.InternalError(c, "获取学习伙伴失败")
		return
	}

	response.Success(c, gin.H{"partners": partners}, "获取成功")
}







