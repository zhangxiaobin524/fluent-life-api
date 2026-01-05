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

type CollectionHandler struct {
	db               *gorm.DB
	collectionService *services.CollectionService
}

func NewCollectionHandler(db *gorm.DB) *CollectionHandler {
	return &CollectionHandler{
		db:               db,
		collectionService: services.NewCollectionService(db),
	}
}

// CollectPost 收藏帖子
func (h *CollectionHandler) CollectPost(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	postIDStr := c.Param("id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	if err := h.collectionService.CollectPost(userID, postID); err != nil {
		response.InternalError(c, "收藏失败")
		return
	}

	favoritesCount, err := h.collectionService.GetPostCollectionCount(postID)
	if err != nil {
		response.InternalError(c, "获取收藏数量失败")
		return
	}

	response.Success(c, gin.H{"favorited": true, "favorites_count": favoritesCount}, "收藏成功")
}

// UncollectPost 取消收藏
func (h *CollectionHandler) UncollectPost(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	postIDStr := c.Param("id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	if err := h.collectionService.UncollectPost(userID, postID); err != nil {
		response.InternalError(c, "取消收藏失败")
		return
	}

	favoritesCount, err := h.collectionService.GetPostCollectionCount(postID)
	if err != nil {
		response.InternalError(c, "获取收藏数量失败")
		return
	}

	response.Success(c, gin.H{"favorited": false, "favorites_count": favoritesCount}, "取消收藏成功")
}

// GetCollectedPosts 获取收藏的帖子列表
func (h *CollectionHandler) GetCollectedPosts(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
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

	posts, total, err := h.collectionService.GetCollectedPosts(userID, page, pageSize)
	if err != nil {
		response.InternalError(c, "获取收藏列表失败")
		return
	}

	response.Success(c, gin.H{
		"posts":     posts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取成功")
}

// CheckCollectionStatus 检查收藏状态
func (h *CollectionHandler) CheckCollectionStatus(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	postIDStr := c.Param("id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	isCollected, err := h.collectionService.IsCollected(userID, postID)
	if err != nil {
		response.InternalError(c, "检查收藏状态失败")
		return
	}

	response.Success(c, gin.H{"is_collected": isCollected}, "获取成功")
}

// ToggleCollectPost 切换帖子收藏状态
func (h *CollectionHandler) ToggleCollectPost(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	postIDStr := c.Param("id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	isCollected, err := h.collectionService.IsCollected(userID, postID)
	if err != nil {
		response.InternalError(c, "检查收藏状态失败")
		return
	}

	if isCollected {
		// 如果已收藏，则取消收藏
		if err := h.collectionService.UncollectPost(userID, postID); err != nil {
			response.InternalError(c, "取消收藏失败")
			return
		}
		isCollected = false
	} else {
		// 如果未收藏，则收藏
		if err := h.collectionService.CollectPost(userID, postID); err != nil {
			response.InternalError(c, "收藏失败")
			return
		}
		isCollected = true
	}

	favoritesCount, err := h.collectionService.GetPostCollectionCount(postID)
	if err != nil {
		response.InternalError(c, "获取收藏数量失败")
		return
	}

	response.Success(c, gin.H{"collected": isCollected, "favorites_count": favoritesCount}, "操作成功")
}
