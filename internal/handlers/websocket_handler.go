package handlers

import (
	"fluent-life-backend/internal/hub"
	"fluent-life-backend/internal/models"
	"fluent-life-backend/internal/services"
	"fluent-life-backend/pkg/auth"
	"fluent-life-backend/pkg/response"
	"log"
	"net/http"
	"os"
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
		Send:     make(chan hub.Message, 512),
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
				// 先检查离开的用户是否是房主
				var leavingMember models.PracticeRoomMember
				isHost := false
				if err := h.db.Where("room_id = ? AND user_id = ?", roomID, userID).First(&leavingMember).Error; err == nil {
					isHost = leavingMember.IsHost
				}

				// 获取上麦用户列表（在删除成员之前）
				onMicUserIDs := h.hub.GetOnMicUsers(roomID.String())

				// 删除成员记录
				err := h.practiceRoomService.LeaveRoom(roomID, userID)
				if err != nil {
					log.Printf("离开房间失败: %v", err)
				}

				// 如果离开的是房主，需要处理房主转移或关闭房间
				if isHost {
					// 检查是否有其他成员在麦上
					var onMicMemberID uuid.UUID
					for _, onMicUserIDStr := range onMicUserIDs {
						onMicUserID, err := uuid.Parse(onMicUserIDStr)
						if err == nil && onMicUserID != userID {
							// 检查这个用户是否还在房间中
							var member models.PracticeRoomMember
							if err := h.db.Where("room_id = ? AND user_id = ?", roomID, onMicUserID).First(&member).Error; err == nil {
								onMicMemberID = onMicUserID
								break
							}
						}
					}

					if onMicMemberID != uuid.Nil {
						// 有成员在麦上，转移房主给第一个在麦上的成员
						if err := h.practiceRoomService.TransferHost(roomID, onMicMemberID); err != nil {
							log.Printf("[OnLeave] 转移房主失败: %v", err)
						} else {
							log.Printf("[OnLeave] 房主已转移给用户 %s", onMicMemberID)
							// 广播房主变更消息
							h.hub.Broadcast <- hub.Message{
								Type:      hub.MessageTypeRoomUpdate,
								RoomID:    roomID.String(),
								Data:      map[string]interface{}{"new_host_id": onMicMemberID.String()},
								Timestamp: time.Now().Unix(),
							}
						}
					} else {
						// 没有人在麦上，检查是否还有其他成员
						var dbMemberCount int64
						h.db.Model(&models.PracticeRoomMember{}).Where("room_id = ?", roomID).Count(&dbMemberCount)
						hubMemberCount := h.hub.GetRoomMemberCount(roomID.String())

						log.Printf("[OnLeave] 房主离开，房间 %s Hub成员数: %d, DB成员数: %d", roomID, hubMemberCount, dbMemberCount)

						// 如果 Hub 和数据库都没有成员了，关闭房间
						if hubMemberCount == 0 && dbMemberCount == 0 {
							var room models.PracticeRoom
							if err := h.db.First(&room, "id = ?", roomID).Error; err == nil {
								room.IsActive = false
								room.CurrentMembers = 0
								if err := h.db.Save(&room).Error; err != nil {
									log.Printf("[OnLeave] ❌ 房主离开后更新房间 %s 状态失败: %v", roomID, err)
								} else {
									log.Printf("[OnLeave] ✅ 房间 %s 已关闭（房主离开且没有人在麦上）", roomID)
								}

								// 广播房间关闭消息给所有全局连接
								h.hub.Broadcast <- hub.Message{
									Type:      hub.MessageTypeRoomDeleted,
									RoomID:    roomID.String(),
									Data:      map[string]interface{}{"room_id": roomID.String()},
									Timestamp: time.Now().Unix(),
								}
							}
						}
					}
				} else {
					// 普通成员离开，检查房间是否应该关闭
					var dbMemberCount int64
					h.db.Model(&models.PracticeRoomMember{}).Where("room_id = ?", roomID).Count(&dbMemberCount)
					hubMemberCount := h.hub.GetRoomMemberCount(roomID.String())

					log.Printf("[OnLeave] 普通成员离开，房间 %s Hub成员数: %d, DB成员数: %d", roomID, hubMemberCount, dbMemberCount)

					// 如果 Hub 和数据库都没有成员了（或只剩1人且那个人不在线），关闭房间
					if hubMemberCount == 0 && dbMemberCount == 0 {
						var room models.PracticeRoom
						if err := h.db.First(&room, "id = ?", roomID).Error; err == nil {
							room.IsActive = false
							room.CurrentMembers = 0
							if err := h.db.Save(&room).Error; err != nil {
								log.Printf("[OnLeave] ❌ 普通成员离开后更新房间 %s 状态失败: %v", roomID, err)
							} else {
								log.Printf("[OnLeave] ✅ 房间 %s 已关闭（没有在线成员）", roomID)
							}

							// 广播房间关闭消息
							h.hub.Broadcast <- hub.Message{
								Type:      hub.MessageTypeRoomDeleted,
								RoomID:    roomID.String(),
								Data:      map[string]interface{}{"room_id": roomID.String()},
								Timestamp: time.Now().Unix(),
							}
						}
					} else if hubMemberCount == 0 && dbMemberCount == 1 {
						// 如果数据库还有1个成员，但 Hub 中没有在线成员，说明最后一个人也离线了，关闭房间
						var room models.PracticeRoom
						if err := h.db.First(&room, "id = ?", roomID).Error; err == nil {
							room.IsActive = false
							room.CurrentMembers = 0
							if err := h.db.Save(&room).Error; err != nil {
								log.Printf("[OnLeave] ❌ 最后一名成员离线后更新房间 %s 状态失败: %v", roomID, err)
							} else {
								log.Printf("[OnLeave] ✅ 房间 %s 已关闭（最后一名成员离线）", roomID)
							}

							// 广播房间关闭消息
							h.hub.Broadcast <- hub.Message{
								Type:      hub.MessageTypeRoomDeleted,
								RoomID:    roomID.String(),
								Data:      map[string]interface{}{"room_id": roomID.String()},
								Timestamp: time.Now().Unix(),
							}
						}
					}
				}
			}
		},
	}

	// 记录WebSocket连接建立
	log.Printf("WebSocket连接建立 - UserID: %s, Username: %s, RoomID: %s", userID.String(), user.Username, roomIDStr)
	os.Stdout.Sync()

	// 注册客户端
	log.Printf("准备注册客户端到 Hub - UserID: %s, RoomID: %s", userID.String(), roomIDStr)
	h.hub.Register <- client
	log.Printf("客户端已发送到 Register 通道 - UserID: %s, RoomID: %s", userID.String(), roomIDStr)

	// 启动读写协程
	log.Printf("准备启动 WritePump 和 ReadPump - UserID: %s, RoomID: %s", userID.String(), roomIDStr)
	go client.WritePump()
	log.Printf("WritePump 已启动 - UserID: %s, RoomID: %s", userID.String(), roomIDStr)
	go client.ReadPump()
	log.Printf("ReadPump 已启动 - UserID: %s, RoomID: %s", userID.String(), roomIDStr)

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
			log.Printf("[monitorRoom] 监控房间 %s: Hub成员数: %d", roomID, memberCount)

			// 如果 Hub 中没有成员，检查数据库
			if memberCount == 0 {
				var dbCount int64
				h.db.Model(&models.PracticeRoomMember{}).Where("room_id = ?", roomID).Count(&dbCount)
				log.Printf("[monitorRoom] 房间 %s Hub成员数为0，DB成员数: %d", roomID, dbCount)

				if dbCount == 0 {
					// 关闭房间
					var room models.PracticeRoom
					if err := h.db.First(&room, "id = ?", roomID).Error; err == nil {
						room.IsActive = false
						room.CurrentMembers = 0
						if err := h.db.Save(&room).Error; err != nil {
							log.Printf("[monitorRoom] ❌ 监控器更新房间 %s 状态失败: %v", roomID, err)
						} else {
							log.Printf("[monitorRoom] ✅ 房间 %s 已关闭（监控器检测到无成员）", roomID)
						}
					}
					return
				}
			}
		}
	}
}
