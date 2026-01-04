package handlers

import (
	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/models"
	"fluent-life-backend/internal/services"
	"fluent-life-backend/internal/utils"
	"fluent-life-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserHandler struct {
	db          *gorm.DB
	userService *services.UserService
}

// NewUserHandler 创建一个新的 UserHandler 实例
func NewUserHandler(db *gorm.DB, cfg *config.Config) *UserHandler {
	return &UserHandler{
		db:          db,
		userService: services.NewUserService(db, cfg),
	}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	response.Success(c, user, "获取成功")
}

func (h *UserHandler) GetUserProfileByID(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	userProfile, err := h.userService.GetUserProfileWithStats(userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "用户不存在")
			return
		}
		response.InternalError(c, "获取用户资料失败")
		return
	}

	response.Success(c, userProfile, "获取成功")
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	var req struct {
		Username *string `json:"username"`
		AvatarURL *string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	if req.Username != nil {
		user.Username = *req.Username
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}

	if err := h.db.Save(&user).Error; err != nil {
		response.InternalError(c, "更新失败")
		return
	}

	response.Success(c, user, "更新成功")
}

func (h *UserHandler) GetStats(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	stats, err := h.userService.CalculateStats(userID)
	if err != nil {
		response.InternalError(c, "计算统计数据失败")
		return
	}

	response.Success(c, stats, "获取成功")
}







