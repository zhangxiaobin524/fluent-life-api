package handlers

import (
	"os"
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
	db               *gorm.DB
	communityService *services.CommunityService
	followService    *services.FollowService
	collectionService *services.CollectionService
}

func NewCommunityHandler(db *gorm.DB) *CommunityHandler {
	return &CommunityHandler{
		db:               db,
		communityService: services.NewCommunityService(db),
		followService:    services.NewFollowService(db),
		collectionService: services.NewCollectionService(db),
	}
}

func (h *CommunityHandler) CreatePost(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	// Ensure uploads directory exists
	if _, err := os.Stat("./uploads"); os.IsNotExist(err) {
		err = os.Mkdir("./uploads", 0755)
		if err != nil {
			response.InternalError(c, "创建上传目录失败")
			return
		}
	}

	content := c.PostForm("content")
	if content == "" {
		response.BadRequest(c, "帖子内容不能为空")
		return
	}

	tag := c.PostForm("tag")
	if tag == "" {
		tag = "心得分享"
	}

	// 获取图片文件
	file, err := c.FormFile("image")
	var imageURL string
	if err == nil { // 如果有图片上传
		// 这里需要实现图片上传逻辑，例如保存到本地或云存储
		// 为了简化，我们暂时只打印文件名，并假设有一个服务来处理实际的存储并返回URL
		// 实际项目中，这里会调用一个文件上传服务
		// 例如：imageURL, err = h.fileUploadService.UploadFile(file)
		// if err != nil {
		//     response.InternalError(c, "图片上传失败")
		//     return
		// }
		// 暂时模拟一个图片URL
		imageURL = "/uploads/" + file.Filename
		// 实际保存文件
		filename := uuid.New().String() + ".jpg" // Assuming all images are jpg for simplicity, or parse file.Header.Get("Content-Type")
		imageURL = "/uploads/" + filename
		if err := c.SaveUploadedFile(file, "./uploads/"+filename); err != nil {
			response.InternalError(c, "保存图片失败")
			return
		}
	} else if err != nil && err.Error() != "http: no such file" {
		// 只有当错误不是“没有文件”时才认为是真正的错误
		response.BadRequest(c, "获取图片文件失败: "+err.Error())
		return
	}

	post, err := h.communityService.CreatePost(userID, content, tag, imageURL)
	if err != nil {
		response.InternalError(c, "发布失败")
		return
	}

	response.Success(c, post, "发布成功")
}

func (h *CommunityHandler) GetPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	sortBy := c.DefaultQuery("sort_by", "created_at") // Default sort by created_at
	tag := c.DefaultQuery("tag", "")                   // Default no tag filter

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

	posts, total, err := h.communityService.GetPosts(page, pageSize, sortBy, tag, userID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	response.Success(c, gin.H{
		"posts":     posts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取成功")
}

func (h *CommunityHandler) GetUserPosts(c *gin.Context) {
	userIDStr := c.Param("id")
	targetUserID, err := uuid.Parse(userIDStr)
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

	var currentUserID *uuid.UUID
	if uid, ok := utils.GetUserID(c); ok {
		currentUserID = &uid
	}

	posts, total, err := h.communityService.GetUserPosts(targetUserID, page, pageSize, currentUserID)
	if err != nil {
		response.InternalError(c, "获取失败")
		return
	}

	type PostWithLiked struct {
		models.Post
		Liked bool `json:"liked"`
	}
	
	postsWithLiked := make([]PostWithLiked, len(posts))
	for i, post := range posts {
		postsWithLiked[i] = PostWithLiked{Post: post, Liked: false}
		if currentUserID != nil {
			for _, like := range post.Likes {
				if like.UserID == *currentUserID {
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

type CreateCommentRequest struct {
	Content string `json:"content" binding:"required"`
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

func (h *CommunityHandler) DeletePost(c *gin.Context) {
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

	err = h.communityService.DeletePost(userID, postID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "帖子不存在或无权删除")
			return
		}
		response.InternalError(c, "删除失败")
		return
	}

	response.Success(c, nil, "删除成功")
}

func (h *CommunityHandler) DeleteComment(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	err = h.communityService.DeleteComment(userID, commentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "评论不存在或无权删除")
			return
		}
		response.InternalError(c, "删除失败")
		return
	}

	response.Success(c, nil, "删除成功")
}

func (h *CommunityHandler) ToggleCommentLike(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "无效的评论ID")
		return
	}

	liked, err := h.communityService.ToggleCommentLike(userID, commentID)
	if err != nil {
		response.InternalError(c, "操作失败")
		return
	}

	response.Success(c, gin.H{"liked": liked}, "操作成功")
}

// CheckUserFollowStatus 检查对用户的关注状态
func (h *CommunityHandler) CheckUserFollowStatus(c *gin.Context) {
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

// CheckPostCollectionStatus 检查帖子收藏状态
func (h *CommunityHandler) CheckPostCollectionStatus(c *gin.Context) {
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

// GetUserFollowCount 获取用户关注数和粉丝数
func (h *CommunityHandler) GetUserFollowCount(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.BadRequest(c, "无效的用户ID")
		return
	}

	followersCount, followingCount, err := h.followService.GetFollowCount(userID)
	if err != nil {
		response.InternalError(c, "获取关注数据失败")
		return
	}

	response.Success(c, gin.H{
		"followers_count": followersCount,
		"following_count": followingCount,
	}, "获取成功")
}

// GetPostCollectionCount 获取帖子收藏数
func (h *CommunityHandler) GetPostCollectionCount(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		response.BadRequest(c, "无效的帖子ID")
		return
	}

	collectionCount, err := h.collectionService.GetPostCollectionCount(postID)
	if err != nil {
		response.InternalError(c, "获取收藏数失败")
		return
	}

	response.Success(c, gin.H{"collection_count": collectionCount}, "获取成功")
}


