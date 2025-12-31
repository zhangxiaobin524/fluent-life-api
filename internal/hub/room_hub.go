package hub

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// getReadPumpLogDir 获取 ReadPump 日志目录路径（相对于项目根目录）
func getReadPumpLogDir() string {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		// 如果获取失败，使用相对路径
		return "logs"
	}

	// 检查是否在 cmd/server 目录下
	if filepath.Base(wd) == "server" && filepath.Base(filepath.Dir(wd)) == "cmd" {
		// 如果在 cmd/server 目录，向上两级到项目根目录
		absPath, _ := filepath.Abs(filepath.Join(wd, "..", "..", "logs"))
		return absPath
	}

	// 检查是否在项目根目录（通过查找 go.mod 或 logs 目录）
	if _, err := os.Stat(filepath.Join(wd, "logs")); err == nil {
		return filepath.Join(wd, "logs")
	}

	// 尝试向上查找项目根目录（查找 go.mod 文件）
	currentDir := wd
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			// 找到了项目根目录
			return filepath.Join(currentDir, "logs")
		}
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // 已经到根目录了
		}
		currentDir = parent
	}

	// 默认使用相对路径（相对于当前工作目录）
	return "logs"
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

const (
	// 消息类型
	MessageTypeJoin           = "join"
	MessageTypeLeave          = "leave"
	MessageTypeMessage        = "message"
	MessageTypeMicOn          = "mic_on"
	MessageTypeMicOff         = "mic_off"
	MessageTypeMicMute        = "mic_mute"
	MessageTypeMicRequest     = "mic_request"
	MessageTypeMicApprove     = "mic_approve"
	MessageTypeMicApproved    = "mic_approved"
	MessageTypeMicReject      = "mic_reject"
	MessageTypeMemberUpdate   = "member_update"
	MessageTypeRoomUpdate     = "room_update"
	MessageTypeRoomCreated    = "room_created"
	MessageTypeRoomDeleted    = "room_deleted"
	MessageTypeRoomListUpdate = "room_list_update"
	MessageTypeWebRTCOffer    = "webrtc_offer"
	MessageTypeWebRTCAnswer   = "webrtc_answer"
	MessageTypeWebRTCICE      = "webrtc_ice"
	// 1v1 连麦相关消息类型
	MessageType1v1MatchRequest = "1v1_match_request"
	MessageType1v1MatchCancel  = "1v1_match_cancel"
	MessageType1v1MatchAccept  = "1v1_match_accept"
	MessageType1v1MatchReject  = "1v1_match_reject"
	MessageType1v1MatchTimeout = "1v1_match_timeout"
	MessageType1v1MatchSuccess = "1v1_match_success"
)

// Message WebSocket 消息结构
type Message struct {
	Type      string      `json:"type"`
	RoomID    string      `json:"room_id,omitempty"`
	UserID    string      `json:"user_id,omitempty"`
	Username  string      `json:"username,omitempty"`
	AvatarURL string      `json:"avatar_url,omitempty"`
	Content   string      `json:"content,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

// Client 代表一个 WebSocket 连接
type Client struct {
	Hub       *RoomHub
	Conn      *websocket.Conn
	Send      chan Message // Buffered channel for outbound messages
	RoomID    string
	UserID    string
	Username  string
	AvatarURL string
	OnLeave   func() // 离开房间的回调函数
}

// RoomHub 管理房间的 WebSocket 连接
type RoomHub struct {
	// 房间ID -> 客户端映射
	Rooms map[string]map[*Client]bool

	// 房间ID -> 上麦用户ID集合 (roomID -> set of userIDs)
	OnMicUsers map[string]map[string]bool

	// 注册新客户端
	Register chan *Client

	// 注销客户端
	Unregister chan *Client

	// 广播消息到房间
	Broadcast chan Message

	// 互斥锁
	Mutex sync.RWMutex

	// 全局客户端连接 (用于1v1匹配)
	GlobalClients map[*Client]bool
	// userID -> 最新的一条全局连接（用于稳定匹配/信令转发）
	GlobalByUserID map[string]*Client
	// 正在匹配的用户 (userID -> bool)
	MatchingUsers map[string]bool
	// 匹配结果通道 (userID -> chan bool)
	MatchChannels map[string]chan bool

	// 1v1 匹配请求通道
	MatchRequests chan *Client

	// 等待匹配的客户端队列
	WaitingClients []*Client
}

var (
	readPumpLogFile *os.File
	readPumpLogger  *log.Logger
	logFileMutex    sync.Mutex
)

// initReadPumpLogger 初始化 ReadPump 日志文件
func initReadPumpLogger() error {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	if readPumpLogFile != nil {
		return nil // 已经初始化
	}

	// 创建日志目录（使用绝对路径或相对于项目根目录）
	logDir := getReadPumpLogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// 打开日志文件（追加模式）
	logPath := filepath.Join(logDir, "readpump.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	readPumpLogFile = file
	readPumpLogger = log.New(file, "", log.LstdFlags)
	return nil
}

// readPumpLog 写入 ReadPump 日志（同时输出到控制台和文件）
func readPumpLog(format string, v ...interface{}) {
	// 确保日志文件已初始化
	if readPumpLogger == nil {
		if err := initReadPumpLogger(); err != nil {
			// 如果初始化失败，只输出到控制台
			log.Printf(format, v...)
			return
		}
	}

	// 同时输出到控制台和文件
	log.Printf(format, v...)
	logFileMutex.Lock()
	readPumpLogger.Printf(format, v...)
	logFileMutex.Unlock()
}

// NewRoomHub 创建新的房间 Hub
func NewRoomHub() *RoomHub {
	return &RoomHub{
		Rooms:          make(map[string]map[*Client]bool),
		OnMicUsers:     make(map[string]map[string]bool),
		GlobalClients:  make(map[*Client]bool),
		GlobalByUserID: make(map[string]*Client),
		MatchingUsers:  make(map[string]bool),
		MatchChannels:  make(map[string]chan bool),
		Register:       make(chan *Client),
		Unregister:     make(chan *Client),
		Broadcast:      make(chan Message),
		MatchRequests:  make(chan *Client),
		WaitingClients: make([]*Client, 0),
	}
}

// Run 运行 Hub
func (h *RoomHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mutex.Lock()
			readPumpLog("[Register] 新客户端注册 - 用户ID: %s, 用户名: %s, 房间ID: %s", client.UserID, client.Username, client.RoomID)

			if h.Rooms[client.RoomID] == nil {
				h.Rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.Rooms[client.RoomID][client] = true

			// 同一个用户可能会重复建立连接（页面切换/重连）。这里用 userID 去重，保留最新连接。
			if old := h.GlobalByUserID[client.UserID]; old != nil && old != client {
				readPumpLog("[Register] 检测到用户 %s (%s) 的旧连接，准备替换 - 旧连接: %p, 新连接: %p", client.UserID, client.Username, old, client)
				delete(h.GlobalClients, old)
				readPumpLog("[Register] 已从 GlobalClients 中移除用户 %s (%s) 的旧连接", client.UserID, client.Username)
			}
			h.GlobalByUserID[client.UserID] = client

			// 保存需要广播的消息（在解锁后广播）
			var broadcastMsg *Message
			if client.RoomID == "global" {
				// 全局连接（用于1v1匹配）
				h.GlobalClients[client] = true
				readPumpLog("[Register] ✅ 全局连接已成功注册 - 用户ID: %s, 用户名: %s, 当前全局在线用户数: %d, 连接指针: %p", client.UserID, client.Username, len(h.GlobalClients), client)
			} else {
				// 如果不是全局连接，准备广播消息（但不在持锁时调用）
				broadcastMsg = &Message{
					Type:      MessageTypeMemberUpdate,
					RoomID:    client.RoomID,
					UserID:    client.UserID,
					Username:  client.Username,
					AvatarURL: client.AvatarURL,
					Data:      map[string]interface{}{"action": "join"},
				}
			}
			h.Mutex.Unlock()

			// 解锁后再广播，避免死锁
			if broadcastMsg != nil {
				h.broadcastToRoom(broadcastMsg.RoomID, *broadcastMsg, client)
			}

		case client := <-h.Unregister:
			roomID := client.RoomID
			userID := client.UserID
			username := client.Username
			h.Mutex.Lock()
			readPumpLog("[Unregister] 客户端注销 - 用户ID: %s, 用户名: %s, 房间ID: %s", userID, username, roomID)

			// Global client cleanup
			if _, wasInGlobal := h.GlobalClients[client]; wasInGlobal {
				delete(h.GlobalClients, client)
				readPumpLog("[Unregister] 已从 GlobalClients 中移除用户 %s (%s)", userID, username)
			}
			if h.GlobalByUserID[userID] == client {
				delete(h.GlobalByUserID, userID)
				readPumpLog("[Unregister] 已从 GlobalByUserID 中移除用户 %s (%s)", userID, username)
			}

			// Clean up matching state if this client was involved
			if h.MatchingUsers[userID] {
				delete(h.MatchingUsers, userID)
				if ch, ok := h.MatchChannels[userID]; ok {
					close(ch)
					delete(h.MatchChannels, userID)
				}
			}
			// Remove from WaitingClients if present
			for i := 0; i < len(h.WaitingClients); i++ {
				c := h.WaitingClients[i]
				if c == client || c.UserID == userID {
					h.WaitingClients = append(h.WaitingClients[:i], h.WaitingClients[i+1:]...)
					i--
				}
			}

			// 保存需要广播的消息（在解锁后广播，避免死锁）
			var leaveBroadcastMsg *Message
			var roomDeletedMsg *Message

			if client.RoomID != "global" {
				// Room client cleanup (existing logic)
				if room, ok := h.Rooms[roomID]; ok {
					if _, ok := room[client]; ok {
						delete(room, client)
						close(client.Send)
						remainingCount := len(room)
						shouldDeleteRoom := remainingCount == 0

						readPumpLog("[Unregister] 房间 %s 清理客户端后，剩余成员数: %d, shouldDeleteRoom: %v", roomID, remainingCount, shouldDeleteRoom)

						if shouldDeleteRoom {
							// 只有房间内完全没有在线成员时才删除房间
							delete(h.Rooms, roomID)
							// 清理上麦用户记录
							delete(h.OnMicUsers, roomID)
							readPumpLog("[Unregister] 房间 %s 已删除（没有在线成员）", roomID)
						} else {
							// 清理该用户的上麦记录（但保留房间）
							if h.OnMicUsers[roomID] != nil {
								delete(h.OnMicUsers[roomID], userID)
							}
							readPumpLog("[Unregister] 房间 %s 保留（还有 %d 个在线成员）", roomID, remainingCount)
						}

						// 准备需要广播的消息（无论是否删除房间，都要通知其他成员有人离开）
						leaveBroadcastMsg = &Message{
							Type:     MessageTypeMemberUpdate,
							RoomID:   roomID,
							UserID:   userID,
							Username: username,
							Data:     map[string]interface{}{"action": "leave", "remaining_count": remainingCount},
						}
						if shouldDeleteRoom {
							// 调用 OnLeave 回调，清理数据库记录
							if client.OnLeave != nil {
								go client.OnLeave()
							}
							// 准备房间关闭消息（在解锁后广播）
							roomDeletedMsg = &Message{
								Type:      MessageTypeRoomDeleted,
								RoomID:    roomID,
								UserID:    userID,
								Username:  username,
								Data:      map[string]interface{}{"room_id": roomID},
								Timestamp: time.Now().Unix(),
							}
						}
					} else {
						readPumpLog("[Unregister] 客户端 %s 不在房间 %s 的映射中", userID, roomID)
					}
				} else {
					readPumpLog("[Unregister] 房间 %s 不存在", roomID)
				}
			}
			// 统一解锁
			h.Mutex.Unlock()

			// 解锁后再广播，避免死锁
			if leaveBroadcastMsg != nil {
				h.broadcastToRoom(roomID, *leaveBroadcastMsg, nil)
			}
			if roomDeletedMsg != nil {
				h.broadcastToAll(*roomDeletedMsg)
			}

		case message := <-h.Broadcast:
			readPumpLog("[RoomHub.Run] 收到 Broadcast 消息，类型: %s, 房间ID: %s", message.Type, message.RoomID)
			h.broadcastToRoom(message.RoomID, message, nil)
			readPumpLog("[RoomHub.Run] ✅ Broadcast 消息已处理完成")

		case reqClient := <-h.MatchRequests:
			readPumpLog("收到匹配请求，用户ID: %s, 用户名: %s, 等待队列长度: %d, global在线数(去重): %d", reqClient.UserID, reqClient.Username, len(h.WaitingClients), len(h.GlobalByUserID))
			h.Mutex.Lock()
			if cur := h.GlobalByUserID[reqClient.UserID]; cur != nil {
				reqClient = cur
			}

			// 先清理等待队列里已离线的用户（避免一直匹配到离线用户）
			if len(h.WaitingClients) > 0 {
				filtered := make([]*Client, 0, len(h.WaitingClients))
				for _, c := range h.WaitingClients {
					if cur := h.GlobalByUserID[c.UserID]; cur != nil {
						filtered = append(filtered, cur)
					}
				}
				h.WaitingClients = filtered
			}

			// ...（后续内容省略，保持原文件其他部分不变）
		}
	}
}

// broadcastToRoom 向房间内所有客户端广播消息（排除发送者）
func (h *RoomHub) broadcastToAll(message Message) {
	// 优化：先复制所有客户端列表，避免持锁发送
	h.Mutex.RLock()
	recipients := make([]*Client, 0)
	for _, room := range h.Rooms {
		for client := range room {
			recipients = append(recipients, client)
		}
	}
	h.Mutex.RUnlock()

	for _, client := range recipients {
		select {
		case client.Send <- message:
		default:
			// 忽略：WritePump 超时/断线会自行清理
		}
	}
}

func (h *RoomHub) broadcastToRoom(roomID string, message Message, exclude *Client) {
	// DEBUG: 进入函数
	readPumpLog("[broadcastToRoom] ENTER | roomID=%s type=%s", roomID, message.Type)

	// 优化：先加读锁，复制接收者列表，然后立刻解锁，避免长时间持锁
	readPumpLog("[broadcastToRoom] 准备获取 RLock | roomID=%s", roomID)
	h.Mutex.RLock()
	readPumpLog("[broadcastToRoom] 已获取 RLock | roomID=%s", roomID)

	room, ok := h.Rooms[roomID]
	if !ok {
		h.Mutex.RUnlock()
		readPumpLog("[broadcastToRoom] 🚫 房间不存在，无法广播 | roomID=%s", roomID)
		return
	}

	readPumpLog("[broadcastToRoom] 房间存在，开始复制 recipients | roomID=%s roomSize=%d", roomID, len(room))
	recipients := make([]*Client, 0, len(room))
	for client := range room {
		if exclude != nil && client == exclude {
			continue
		}
		recipients = append(recipients, client)
	}
	h.Mutex.RUnlock()
	readPumpLog("[broadcastToRoom] 已释放 RLock，recipients=%d | roomID=%s", len(recipients), roomID)

	if len(recipients) == 0 {
		readPumpLog("[broadcastToRoom] ⚠️ 无可接收者（排除发送者后为0）| roomID=%s", roomID)
		return
	}

	readPumpLog("[broadcastToRoom] 📢 向房间 %s 的 %d 个客户端广播消息类型: %s", roomID, len(recipients), message.Type)

	// 不持锁进行发送，避免因某个客户端阻塞而卡住整个 Hub
	for _, client := range recipients {
		select {
		case client.Send <- message:
			// success
		default:
			// 发送失败，可能客户端已断开或通道已满
			readPumpLog("[broadcastToRoom] ❌ 无法发送消息给用户 %s (%s)，准备清理连接", client.UserID, client.Username)
			// 单独加写锁进行清理
			h.Mutex.Lock()
			if currentRoom, roomExists := h.Rooms[roomID]; roomExists {
				if _, clientExists := currentRoom[client]; clientExists {
					delete(currentRoom, client)
					// 安全地关闭 channel（close 已关闭的 channel 会 panic，用 recover 保底）
					func() {
						defer func() { _ = recover() }()
						close(client.Send)
					}()
				}
				if len(currentRoom) == 0 {
					delete(h.Rooms, roomID)
				}
			}
			h.Mutex.Unlock()
		}
	}
}

// GetRoomMemberCount 获取房间成员数
func (h *RoomHub) GetRoomMemberCount(roomID string) int {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()
	if room, ok := h.Rooms[roomID]; ok {
		return len(room)
	}
	return 0
}

// GetOnMicUsers 获取房间中上麦的用户ID列表
func (h *RoomHub) GetOnMicUsers(roomID string) []string {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	onMicSet, ok := h.OnMicUsers[roomID]
	if !ok {
		return []string{}
	}

	userIDs := make([]string, 0, len(onMicSet))
	for uid := range onMicSet {
		userIDs = append(userIDs, uid)
	}
	return userIDs
}

// sendToUser 向特定用户发送消息
func (h *RoomHub) sendToUser(message Message, sender *Client) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	// 从消息中获取目标用户ID
	targetUserID := ""
	if data, ok := message.Data.(map[string]interface{}); ok {
		if tid, ok := data["target_user_id"].(string); ok {
			targetUserID = tid
		}
	}

	if targetUserID == "" {
		return
	}

	// 在同一个房间内查找目标用户
	if room, ok := h.Rooms[sender.RoomID]; ok {
		for client := range room {
			if client.UserID == targetUserID && client != sender {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
				}
				return
			}
		}
	}
}

// ReadPump 从 WebSocket 连接读取消息
func (c *Client) ReadPump() {
	readPumpLog("[ReadPump] ========== ReadPump 启动 ========== 用户ID: %s, 用户名: %s, 房间ID: %s, 连接指针: %p", c.UserID, c.Username, c.RoomID, c)
	defer func() {
		readPumpLog("[ReadPump] ========== ReadPump 退出 ========== 用户ID: %s, 用户名: %s, 房间ID: %s, 连接指针: %p", c.UserID, c.Username, c.RoomID, c)
		if c.OnLeave != nil {
			c.OnLeave()
		}
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	readPumpLog("[ReadPump] ✅ 开始监听用户 %s (%s) 在房间 %s 的消息，连接指针: %p", c.UserID, c.Username, c.RoomID, c)
	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				readPumpLog("ℹ️ WebSocket 连接被客户端正常关闭 (用户ID: %s, 用户名: %s, 房间ID: %s): %v", c.UserID, c.Username, c.RoomID, err)
			} else if websocket.IsCloseError(err, websocket.CloseNoStatusReceived) {
				readPumpLog("ℹ️ WebSocket 连接关闭但未收到状态码 (用户ID: %s, 用户名: %s, 房间ID: %s): %v", c.UserID, c.Username, c.RoomID, err)
			} else {
				readPumpLog("❌ WebSocket 读取错误 (用户ID: %s, 用户名: %s, 房间ID: %s): %v", c.UserID, c.Username, c.RoomID, err)
			}
			break
		}

		readPumpLog("[ReadPump] ========== 收到用户 %s (%s) 的原始消息 ==========", c.UserID, c.Username)
		readPumpLog("[ReadPump] 消息内容: %s", string(messageBytes))

		var message Message
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			readPumpLog("[ReadPump] 解析用户 %s (%s) 的消息失败: %v", c.UserID, c.Username, err)
			continue
		}

		readPumpLog("[ReadPump] 解析用户 %s (%s) 的消息类型为 '%s'", c.UserID, c.Username, message.Type)

		message.UserID = c.UserID
		message.Username = c.Username
		message.AvatarURL = c.AvatarURL
		message.RoomID = c.RoomID
		message.Timestamp = time.Now().Unix()

		switch message.Type {
		case MessageTypeMessage:
			c.Hub.Broadcast <- message
		case MessageTypeMicApprove:
			readPumpLog("[ReadPump] ========== 收到用户 %s (%s) 的 mic_approve 消息 ==========", c.UserID, c.Username)
			readPumpLog("[ReadPump] mic_approve 消息完整内容: %+v", message)
			readPumpLog("[ReadPump] message.Data 类型: %T, 值: %+v", message.Data, message.Data)

			if data, ok := message.Data.(map[string]interface{}); ok {
				readPumpLog("[ReadPump] Data 是 map，内容: %+v", data)
				readPumpLog("[ReadPump] Data 中的所有键: %v", func() []string {
					keys := make([]string, 0, len(data))
					for k := range data {
						keys = append(keys, k)
					}
					return keys
				}())
				if targetUserID, ok := data["target_user_id"].(string); ok {
					readPumpLog("[ReadPump] ✅ 找到 target_user_id: %s", targetUserID)
					approvedMessage := Message{
						Type:      MessageTypeMicApproved,
						RoomID:    c.RoomID,
						UserID:    targetUserID,
						Username:  c.Username,
						AvatarURL: c.AvatarURL,
						Data:      map[string]interface{}{"user_id": targetUserID},
						Timestamp: time.Now().Unix(),
					}
					readPumpLog("[ReadPump] 准备发送 mic_approved 给用户 %s (房间 %s)", targetUserID, c.RoomID)
					// 注意：sendToUser 需要 target_user_id，但 approvedMessage 的 Data 是 user_id
					// 所以这里直接广播，让被批准者也能收到
					c.Hub.broadcastToRoom(c.RoomID, approvedMessage, c)
					readPumpLog("[ReadPump] ✅ mic_approved 已广播给房间 %s", c.RoomID)
				} else {
					readPumpLog("[ReadPump] ⚠️ Data 中没有 target_user_id 字段，或类型不是 string")
					if val, exists := data["target_user_id"]; exists {
						readPumpLog("[ReadPump] target_user_id 存在但类型是 %T, 值: %v", val, val)
					}
				}
			} else {
				readPumpLog("[ReadPump] ⚠️ message.Data 不是 map[string]interface{} 类型，无法解析")
			}
		case MessageTypeMicReject:
			if data, ok := message.Data.(map[string]interface{}); ok {
				if targetUserID, ok := data["target_user_id"].(string); ok {
					rejectMessage := Message{
						Type:      MessageTypeMicReject,
						RoomID:    c.RoomID,
						UserID:    targetUserID,
						Username:  c.Username,
						AvatarURL: c.AvatarURL,
						Data:      map[string]interface{}{"user_id": targetUserID},
						Timestamp: time.Now().Unix(),
					}
					c.Hub.sendToUser(rejectMessage, c)
				}
			}
		case MessageTypeMicOn:
			c.Hub.Mutex.Lock()
			if c.Hub.OnMicUsers[c.RoomID] == nil {
				c.Hub.OnMicUsers[c.RoomID] = make(map[string]bool)
			}
			c.Hub.OnMicUsers[c.RoomID][c.UserID] = true
			c.Hub.Mutex.Unlock()
			c.Hub.broadcastToRoom(c.RoomID, message, c)
		case MessageTypeMicOff:
			c.Hub.Mutex.Lock()
			if c.Hub.OnMicUsers[c.RoomID] != nil {
				delete(c.Hub.OnMicUsers[c.RoomID], c.UserID)
			}
			c.Hub.Mutex.Unlock()
			c.Hub.broadcastToRoom(c.RoomID, message, c)
		case MessageTypeMicRequest:
			readPumpLog("[ReadPump] ========== 收到用户 %s (%s) 的连麦申请 ==========", c.UserID, c.Username)
			readPumpLog("[ReadPump] 房间ID: %s, 准备广播给房间内其他成员", c.RoomID)

			if message.Data == nil {
				message.Data = make(map[string]interface{})
				readPumpLog("[ReadPump] Data 字段为 nil，已初始化为空 map")
			}
			if data, ok := message.Data.(map[string]interface{}); ok {
				if _, exists := data["user_id"]; !exists {
					data["user_id"] = c.UserID
					readPumpLog("[ReadPump] 已添加 user_id 到 Data: %s", c.UserID)
				}
				if _, exists := data["username"]; !exists {
					data["username"] = c.Username
					readPumpLog("[ReadPump] 已添加 username 到 Data: %s", c.Username)
				}
				if _, exists := data["avatar_url"]; !exists {
					data["avatar_url"] = c.AvatarURL
					readPumpLog("[ReadPump] 已添加 avatar_url 到 Data: %s", c.AvatarURL)
				}
				readPumpLog("[ReadPump] Data 字段内容: user_id=%s, username=%s, avatar_url=%s", data["user_id"], data["username"], data["avatar_url"])
			} else {
				readPumpLog("[ReadPump] ⚠️ Data 字段类型不是 map[string]interface{}，类型: %T", message.Data)
			}

			message.UserID = c.UserID
			message.Username = c.Username
			message.AvatarURL = c.AvatarURL
			readPumpLog("[ReadPump] 消息顶层字段: UserID=%s, Username=%s, AvatarURL=%s", message.UserID, message.Username, message.AvatarURL)

			readPumpLog("[ReadPump] 准备调用 broadcastToRoom，房间ID: %s", c.RoomID)
			c.Hub.broadcastToRoom(c.RoomID, message, c)
			readPumpLog("[ReadPump] ✅ 连麦申请已广播给房间 %s 的其他成员", c.RoomID)
		case MessageTypeMicMute, MessageTypeMicApproved:
			c.Hub.broadcastToRoom(c.RoomID, message, c)
		case MessageTypeWebRTCOffer, MessageTypeWebRTCAnswer, MessageTypeWebRTCICE:
			c.Hub.sendToUser(message, c)
		case MessageType1v1MatchRequest:
			readPumpLog("[ReadPump] 收到用户 %s (%s) 的 1v1 匹配请求，准备加入匹配队列", c.UserID, c.Username)
			c.Hub.Mutex.RLock()
			currentClient := c.Hub.GlobalByUserID[c.UserID]
			isLatestConnection := currentClient == c
			c.Hub.Mutex.RUnlock()

			if !isLatestConnection {
				readPumpLog("[ReadPump] ⚠️ 警告：用户 %s (%s) 的匹配请求来自旧连接，当前最新连接: %p, 请求连接: %p", c.UserID, c.Username, currentClient, c)
			} else {
				readPumpLog("[ReadPump] ✅ 用户 %s (%s) 的匹配请求来自最新连接，发送到匹配队列", c.UserID, c.Username)
			}
			c.Hub.MatchRequests <- c
		case MessageType1v1MatchCancel:
			readPumpLog("[ReadPump] 收到用户 %s (%s) 的 1v1 匹配取消请求", c.UserID, c.Username)
		default:
			readPumpLog("[ReadPump] ⚠️ 收到未直接处理的消息类型: '%s' (来自用户 %s, 房间 %s)", message.Type, c.UserID, c.RoomID)
			readPumpLog("[ReadPump] 原始消息详情: %+v", message)
			c.Hub.broadcastToRoom(c.RoomID, message, c)
		}
	}
}

// WritePump 向 WebSocket 连接写入消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				readPumpLog("[WritePump] 用户 %s (%s): Hub 关闭了发送通道", c.UserID, c.Username)
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				readPumpLog("[WritePump] 用户 %s (%s): 获取写入器失败: %v", c.UserID, c.Username, err)
				return
			}
			jsonMessage, err := json.Marshal(message)
			if err != nil {
				readPumpLog("[WritePump] 用户 %s (%s): 序列化消息失败: %v", c.UserID, c.Username, err)
				return
			}
			w.Write(jsonMessage)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				queuedMessage := <-c.Send
				jsonQueuedMessage, err := json.Marshal(queuedMessage)
				if err != nil {
					readPumpLog("[WritePump] 用户 %s (%s): 序列化队列消息失败: %v", c.UserID, c.Username, err)
					return
				}
				w.Write(jsonQueuedMessage)
			}

			if err := w.Close(); err != nil {
				readPumpLog("[WritePump] 用户 %s (%s): 关闭写入器失败: %v", c.UserID, c.Username, err)
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				readPumpLog("[WritePump] 用户 %s (%s): 发送 ping 消息失败: %v", c.UserID, c.Username, err)
				return
			}
		}
	}
}
