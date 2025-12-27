package handlers

import (
	"fluent-life-backend/internal/hub"
	"fluent-life-backend/internal/models"
	"fluent-life-backend/internal/services"
	"fluent-life-backend/internal/utils"
	"fluent-life-backend/pkg/response"
	"time"

	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PracticeRoomHandler struct {
	db                  *gorm.DB
	practiceRoomService *services.PracticeRoomService
	roomHub             *hub.RoomHub
}

func NewPracticeRoomHandler(db *gorm.DB, roomHub *hub.RoomHub) *PracticeRoomHandler {
	return &PracticeRoomHandler{
		db:                  db,
		practiceRoomService: services.NewPracticeRoomService(db),
		roomHub:             roomHub,
	}
}

// CreateRoom 创建对练房
func (h *PracticeRoomHandler) CreateRoom(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	var req struct {
		Title string `json:"title" binding:"required"`
		Theme string `json:"theme" binding:"required"`
		Type  string `json:"type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	room, err := h.practiceRoomService.CreateRoom(userID, req.Title, req.Theme, req.Type)
	if err != nil {
		response.InternalError(c, "创建房间失败")
		return
	}

	// 加载房间的完整信息（包括User信息）用于广播
	var fullRoom models.PracticeRoom
	if err := h.db.Preload("User").First(&fullRoom, "id = ?", room.ID).Error; err == nil {
		// 广播房间创建事件给所有在线用户（通过全局广播）
		if h.roomHub != nil {
			h.roomHub.Broadcast <- hub.Message{
				Type:      hub.MessageTypeRoomCreated,
				RoomID:    "global", // 全局广播
				Data:      fullRoom,
				Timestamp: time.Now().Unix(),
			}
		}
		response.Success(c, fullRoom, "创建成功")
	} else {
		response.Success(c, room, "创建成功")
	}
}

// GetRooms 获取房间列表
func (h *PracticeRoomHandler) GetRooms(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	theme := c.Query("theme")
	roomType := c.Query("type")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	rooms, total, err := h.practiceRoomService.GetRooms(page, pageSize, theme, roomType)
	if err != nil {
		response.InternalError(c, "获取房间列表失败")
		return
	}

	response.Success(c, gin.H{
		"rooms":     rooms,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, "获取成功")
}

// GetRoom 获取房间详情
func (h *PracticeRoomHandler) GetRoom(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "房间ID格式错误")
		return
	}

	room, err := h.practiceRoomService.GetRoom(roomID)
	if err != nil {
		response.NotFound(c, "房间不存在")
		return
	}

	response.Success(c, room, "获取成功")
}

// JoinRoom 加入房间
func (h *PracticeRoomHandler) JoinRoom(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	roomIDStr := c.Param("id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "房间ID格式错误")
		return
	}

	if err := h.practiceRoomService.JoinRoom(roomID, userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "房间不存在或已满")
			return
		}
		response.InternalError(c, "加入房间失败")
		return
	}

	response.Success(c, nil, "加入成功")
}

// LeaveRoom 离开房间
func (h *PracticeRoomHandler) LeaveRoom(c *gin.Context) {
	userID, ok := utils.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "未找到用户信息")
		return
	}

	roomIDStr := c.Param("id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		response.BadRequest(c, "房间ID格式错误")
		return
	}

	if err := h.practiceRoomService.LeaveRoom(roomID, userID); err != nil {
		response.InternalError(c, "离开房间失败")
		return
	}

	response.Success(c, nil, "离开成功")
}
