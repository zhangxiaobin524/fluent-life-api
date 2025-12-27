package handlers

import (
	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/services"
	"fluent-life-backend/internal/utils"
	"fluent-life-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AIHandler struct {
	db        *gorm.DB
	aiService *services.AIService
}

func NewAIHandler(db *gorm.DB, cfg *config.Config) *AIHandler {
	return &AIHandler{
		db:        db,
		aiService: services.NewAIService(db, cfg),
	}
}

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

type AnalyzeSpeechRequest struct {
	Transcription string `json:"transcription" binding:"required"`
}

func (h *AIHandler) Chat(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	reply, err := h.aiService.Chat(userID, req.Message)
	if err != nil {
		response.InternalError(c, "AI回复失败")
		return
	}

	response.Success(c, gin.H{"reply": reply}, "获取成功")
}

func (h *AIHandler) GetConversation(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	conversation, err := h.aiService.GetConversation(userID)
	if err != nil {
		response.InternalError(c, "获取对话失败")
		return
	}

	response.Success(c, conversation, "获取成功")
}

func (h *AIHandler) AnalyzeSpeech(c *gin.Context) {
	var req AnalyzeSpeechRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	analysis, err := h.aiService.AnalyzeSpeech(req.Transcription)
	if err != nil {
		response.InternalError(c, "分析失败")
		return
	}

	response.Success(c, gin.H{"analysis": analysis}, "分析成功")
}







