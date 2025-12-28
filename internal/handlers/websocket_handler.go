package handlers

import (
	"fluent-life-backend/internal/hub"
	"fluent-life-backend/internal/models"
	"fluent-life-backend/internal/services"
	"fluent-life-backend/pkg/auth"
	"fluent-life-backend/pkg/response"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该检查
	},
}

type WebSocketHandler struct {
	hub                 *hub.RoomHub
	db                  *gorm.DB
	practiceRoomService *services.PracticeRoomService
	jwtSecret           string
}

func NewWebSocketHandler(hub *hub.RoomHub, db *gorm.DB, jwtSecret string) *WebSocketHandler {
	return &WebSocketHandler{
		hub:                 hub,
		db:                  db,
		practiceRoomService: services.NewPracticeRoomService(db),
		jwtSecret:           jwtSecret,
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 先升级到 WebSocket（必须在任何响应写入之前完成）
	// 注意：必须在任何响应头写入之前调用 Upgrade
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		// 如果升级失败，检查响应是否已经写入
		if !c.Writer.Written() {
			response.InternalError(c, "WebSocket升级失败: "+err.Error())
		}
		return
	}

	// 从查询参数获取 token（WebSocket 连接无法使用 HTTP 头部）
	tokenStr := c.Query("token")
	if tokenStr == "" {
		// 尝试从 Authorization 头部获取（如果通过 HTTP 中间件）
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenStr = authHeader[7:]
		}
	}

	// 验证 token（升级后验证，失败则关闭连接）
	if tokenStr == "" {
		conn.WriteJSON(gin.H{"type": "error", "message": "未提供认证令牌"})
		conn.Close()
		return
	}

	// 验证 token
	claims, err := auth.ValidateToken(tokenStr, h.jwtSecret)
	if err != nil {
		conn.WriteJSON(gin.H{"type": "error", "message": "无效的认证令牌"})
		conn.Close()
		return
	}

	userID := claims.UserID

	// 获取用户信息
	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		conn.WriteJSON(gin.H{"type": "error", "message": "用户不存在"})
		conn.Close()
		return
	}

	// 获取房间ID（可选，如果没有则创建全局连接用于监听房间列表）
	roomIDStr := c.Query("room_id")
	var roomID uuid.UUID
	var room models.PracticeRoom

	if roomIDStr != "" && roomIDStr != "global" {
		var err error
		roomID, err = uuid.Parse(roomIDStr)
		if err != nil {
			conn.WriteJSON(gin.H{"type": "error", "message": "房间ID格式错误"})
			conn.Close()
			return
		}

		// 验证房间是否存在
		if err := h.db.Preload("Members.User").First(&room, "id = ?", roomID).Error; err != nil {
			conn.WriteJSON(gin.H{"type": "error", "message": "房间不存在"})
			conn.Close()
			return
		}

		// 检查用户是否是房间成员
		var member models.PracticeRoomMember
		if err := h.db.Where("room_id = ? AND user_id = ?", roomID, userID).First(&member).Error; err != nil {
			// 如果不是成员，自动加入
			member = models.PracticeRoomMember{
				RoomID: roomID,
				UserID: userID,
				IsHost: false,
			}
			h.db.Create(&member)

			// 更新房间成员数
			room.CurrentMembers++
			h.db.Save(&room)
		}
	} else {
		// 全局连接，用于监听房间列表更新
		roomIDStr = "global"
		roomID = uuid.Nil
	}

	// 创建客户端
	client := &hub.Client{
		Hub:      h.hub,
		Conn:     conn,
		Send:     make(chan hub.Message, 256),
		RoomID:   roomIDStr,
		UserID:   userID.String(),
		Username: user.Username,
		AvatarURL: func() string {
			if user.AvatarURL != nil {
				return *user.AvatarURL
			}
			return ""
		}(),
		OnLeave: func() {
			// WebSocket断开时，从数据库删除成员记录（仅当不是全局连接时）
			if roomIDStr != "global" && roomID != uuid.Nil {
				err := h.practiceRoomService.LeaveRoom(roomID, userID)
				if err != nil {
					log.Printf("离开房间失败: %v", err)
				}
				// 检查房间是否应该关闭（成员数为0）
				var room models.PracticeRoom
				if err := h.db.First(&room, "id = ?", roomID).Error; err == nil {
					// 检查 Hub 中是否还有在线成员
					hubMemberCount := h.hub.GetRoomMemberCount(roomID.String())
					// 检查数据库中的成员数
					var dbMemberCount int64
					h.db.Model(&models.PracticeRoomMember{}).Where("room_id = ?", roomID).Count(&dbMemberCount)
					
					// 如果 Hub 和数据库都没有成员了，关闭房间
					if hubMemberCount == 0 && dbMemberCount == 0 {
						room.IsActive = false
						room.CurrentMembers = 0
						h.db.Save(&room)
						log.Printf("房间 %s 已关闭（没有在线成员）", roomID)
					}
				}
			}
		},
	}

	// 注册客户端
	h.hub.Register <- client

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()

	// 监听房间关闭（仅当不是全局连接时）
	if roomIDStr != "global" && roomID != uuid.Nil {
		go h.monitorRoom(roomID, client)
	}
}

// monitorRoom 监控房间，当成员数为0时关闭房间
func (h *WebSocketHandler) monitorRoom(roomID uuid.UUID, client *hub.Client) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			memberCount := h.hub.GetRoomMemberCount(roomID.String())

			// 如果 Hub 中没有成员，检查数据库
			if memberCount == 0 {
				var dbCount int64
				h.db.Model(&models.PracticeRoomMember{}).Where("room_id = ?", roomID).Count(&dbCount)

				if dbCount == 0 {
					// 关闭房间
					var room models.PracticeRoom
					if err := h.db.First(&room, "id = ?", roomID).Error; err == nil {
						room.IsActive = false
						room.CurrentMembers = 0
						h.db.Save(&room)
					}
					return
				}
			}
		}
	}
}
