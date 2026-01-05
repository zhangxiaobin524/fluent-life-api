package handlers

import (
	"strconv"

	"fluent-life-backend/internal/services"
	"fluent-life-backend/internal/utils"
	"fluent-life-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FollowHandler struct {
	db           *gorm.DB
	followService *services.FollowService
}

func NewFollowHandler(db *gorm.DB) *FollowHandler {
	return &FollowHandler{
		db:           db,
		followService: services.NewFollowService(db),
	}
}

// FollowUser 关注用户
func (h *FollowHandler) FollowUser(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	followingIDStr := c.Param("id")
	followingID, err := uuid.Parse(followingIDStr)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := h.followService.FollowUser(userID, followingID); err != nil {
		if err == gorm.ErrInvalidTransaction {
			response.BadRequest(c, "不能关注自己")
			return
		}
		response.InternalError(c, "关注失败")
		return
	}

	response.Success(c, nil, "关注成功")
}

// UnfollowUser 取消关注
func (h *FollowHandler) UnfollowUser(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	followingIDStr := c.Param("id")
	followingID, err := uuid.Parse(followingIDStr)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	if err := h.followService.UnfollowUser(userID, followingID); err != nil {
		response.InternalError(c, "取消关注失败")
		return
	}

	response.Success(c, nil, "取消关注成功")
}

// GetFollowers 获取粉丝列表
func (h *FollowHandler) GetFollowers(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.followService.GetFollowers(userID, page, pageSize)
	if err != nil {
		response.InternalError(c, "获取粉丝列表失败")
		return
	}

	response.Success(c, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取成功")
}

// GetFollowing 获取关注列表
func (h *FollowHandler) GetFollowing(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, total, err := h.followService.GetFollowing(userID, page, pageSize)
	if err != nil {
		response.InternalError(c, "获取关注列表失败")
		return
	}

	response.Success(c, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取成功")
}

// CheckFollowStatus 检查关注状态
func (h *FollowHandler) CheckFollowStatus(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	targetIDStr := c.Param("id")
	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	isFollowing, err := h.followService.IsFollowing(userID, targetID)
	if err != nil {
		response.InternalError(c, "检查关注状态失败")
		return
	}

	response.Success(c, gin.H{"is_following": isFollowing}, "获取成功")
}
