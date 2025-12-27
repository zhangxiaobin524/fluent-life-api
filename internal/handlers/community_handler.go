package handlers

import (
	"strconv"

	"fluent-life-backend/internal/models"
	"fluent-life-backend/internal/services"
	"fluent-life-backend/internal/utils"
	"fluent-life-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommunityHandler struct {
	db              *gorm.DB
	communityService *services.CommunityService
}

func NewCommunityHandler(db *gorm.DB) *CommunityHandler {
	return &CommunityHandler{
		db:               db,
		communityService: services.NewCommunityService(db),
	}
}

type CreatePostRequest struct {
	Content string `json:"content" binding:"required"`
	Tag     string `json:"tag"`
}

type CreateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}

func (h *CommunityHandler) CreatePost(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.Tag == "" {
		req.Tag = "心得分享"
	}

	post, err := h.communityService.CreatePost(userID, req.Content, req.Tag)
	if err != nil {
		response.InternalError(c, "发布失败")
		return
	}

	response.Success(c, post, "发布成功")
}

func (h *CommunityHandler) GetPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var userID *uuid.UUID
	if uid, ok := utils.GetUserID(c); ok {
		userID = &uid
	}

	posts, total, err := h.communityService.GetPosts(page, pageSize, userID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	// 构建包含点赞状态的帖子列表
	type PostWithLiked struct {
		models.Post
		Liked bool `json:"liked"`
	}
	
	postsWithLiked := make([]PostWithLiked, len(posts))
	for i, post := range posts {
		postsWithLiked[i] = PostWithLiked{Post: post, Liked: false}
		if userID != nil {
			for _, like := range post.Likes {
				if like.UserID == *userID {
					postsWithLiked[i].Liked = true
					break
				}
			}
		}
	}

	response.Success(c, gin.H{
		"posts":     postsWithLiked,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取成功")
}

func (h *CommunityHandler) GetPost(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	post, err := h.communityService.GetPost(postID)
	if err != nil {
		response.NotFound(c, "帖子不存在")
		return
	}

	response.Success(c, post, "获取成功")
}

func (h *CommunityHandler) ToggleLike(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	liked, err := h.communityService.ToggleLike(userID, postID)
	if err != nil {
		response.InternalError(c, "操作失败")
		return
	}

	response.Success(c, gin.H{"liked": liked}, "操作成功")
}

func (h *CommunityHandler) GetComments(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	comments, err := h.communityService.GetComments(postID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	response.Success(c, comments, "获取成功")
}

func (h *CommunityHandler) CreateComment(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	var req CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	comment, err := h.communityService.CreateComment(userID, postID, req.Content)
	if err != nil {
		response.InternalError(c, "评论失败")
		return
	}

	response.Success(c, comment, "评论成功")
}

