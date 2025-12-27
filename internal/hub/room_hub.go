package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
	MessageTypeMemberUpdate   = "member_update"
	MessageTypeRoomUpdate     = "room_update"
	MessageTypeRoomCreated    = "room_created"
	MessageTypeRoomDeleted    = "room_deleted"
	MessageTypeRoomListUpdate = "room_list_update"
	MessageTypeWebRTCOffer    = "webrtc_offer"
	MessageTypeWebRTCAnswer   = "webrtc_answer"
	MessageTypeWebRTCICE      = "webrtc_ice"
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
	Send      chan Message
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

	// 注册新客户端
	Register chan *Client

	// 注销客户端
	Unregister chan *Client

	// 广播消息到房间
	Broadcast chan Message

	// 互斥锁
	Mutex sync.RWMutex
}

// NewRoomHub 创建新的房间 Hub
func NewRoomHub() *RoomHub {
	return &RoomHub{
		Rooms:      make(map[string]map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan Message),
	}
}

// Run 运行 Hub
func (h *RoomHub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Mutex.Lock()
			if h.Rooms[client.RoomID] == nil {
				h.Rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.Rooms[client.RoomID][client] = true
			h.Mutex.Unlock()

			// 通知其他成员有新成员加入
			h.broadcastToRoom(client.RoomID, Message{
				Type:      MessageTypeMemberUpdate,
				RoomID:    client.RoomID,
				UserID:    client.UserID,
				Username:  client.Username,
				AvatarURL: client.AvatarURL,
				Data:      map[string]interface{}{"action": "join"},
			}, client)

		case client := <-h.Unregister:
			h.Mutex.Lock()
			if room, ok := h.Rooms[client.RoomID]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.Send)
					if len(room) == 0 {
						delete(h.Rooms, client.RoomID)
					}
				}
			}
			h.Mutex.Unlock()

			// 通知其他成员有成员离开
			h.broadcastToRoom(client.RoomID, Message{
				Type:     MessageTypeMemberUpdate,
				RoomID:   client.RoomID,
				UserID:   client.UserID,
				Username: client.Username,
				Data:     map[string]interface{}{"action": "leave"},
			}, nil)

		case message := <-h.Broadcast:
			h.broadcastToRoom(message.RoomID, message, nil)
		}
	}
}

// broadcastToRoom 向房间内所有客户端广播消息（排除发送者）
func (h *RoomHub) broadcastToRoom(roomID string, message Message, exclude *Client) {
	h.Mutex.RLock()
	room, ok := h.Rooms[roomID]
	if !ok {
		h.Mutex.RUnlock()
		return
	}
	h.Mutex.RUnlock()

	for client := range room {
		if exclude != nil && client == exclude {
			continue
		}
		select {
		case client.Send <- message:
		default:
			close(client.Send)
			h.Mutex.Lock()
			delete(room, client)
			if len(room) == 0 {
				delete(h.Rooms, roomID)
			}
			h.Mutex.Unlock()
		}
	}
}

// broadcastToAll 向所有连接的客户端广播消息
func (h *RoomHub) broadcastToAll(message Message) {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()

	for _, room := range h.Rooms {
		for client := range room {
			select {
			case client.Send <- message:
			default:
				close(client.Send)
			}
		}
	}
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

// GetRoomMemberCount 获取房间成员数
func (h *RoomHub) GetRoomMemberCount(roomID string) int {
	h.Mutex.RLock()
	defer h.Mutex.RUnlock()
	if room, ok := h.Rooms[roomID]; ok {
		return len(room)
	}
	return 0
}

// ReadPump 从 WebSocket 连接读取消息
func (c *Client) ReadPump() {
	defer func() {
		// WebSocket断开时，调用离开房间回调
		if c.OnLeave != nil {
			c.OnLeave()
		}

		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var message Message
		if err := json.Unmarshal(messageBytes, &message); err != nil {
			log.Printf("解析消息失败: %v", err)
			continue
		}

		// 设置发送者信息
		message.UserID = c.UserID
		message.Username = c.Username
		message.AvatarURL = c.AvatarURL
		message.RoomID = c.RoomID
		message.Timestamp = time.Now().Unix()

		// 处理不同类型的消息
		switch message.Type {
		case MessageTypeMessage:
			// 文字消息：广播给所有成员（包括发送者）
			c.Hub.Broadcast <- message
		case MessageTypeMicApprove:
			// mic_approve 消息：转换为 mic_approved 并发送给目标用户
			if data, ok := message.Data.(map[string]interface{}); ok {
				if targetUserID, ok := data["target_user_id"].(string); ok {
					// 创建 mic_approved 消息发送给被批准的用户
					approvedMessage := Message{
						Type:      MessageTypeMicApproved,
						RoomID:    c.RoomID,
						UserID:    targetUserID,
						Username:  c.Username,
						AvatarURL: c.AvatarURL,
						Data:      map[string]interface{}{"user_id": targetUserID},
						Timestamp: time.Now().Unix(),
					}
					c.Hub.sendToUser(approvedMessage, c)
					// 同时广播给房间内其他成员（让他们知道有人被批准了）
					c.Hub.broadcastToRoom(c.RoomID, approvedMessage, c)
				}
			}
		case MessageTypeMicOn, MessageTypeMicOff, MessageTypeMicMute, MessageTypeMicRequest, MessageTypeMicApproved:
			// 麦克风相关消息：广播给其他成员
			c.Hub.broadcastToRoom(c.RoomID, message, c)
		case MessageTypeWebRTCOffer, MessageTypeWebRTCAnswer, MessageTypeWebRTCICE:
			// WebRTC消息：只发送给目标用户
			c.Hub.sendToUser(message, c)
		default:
			// 其他消息：广播给所有成员
			c.Hub.Broadcast <- message
		}
	}
}

// WritePump 向 WebSocket 连接写入消息
func (c *Client) WritePump() {
	defer c.Conn.Close()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				log.Printf("写入消息失败: %v", err)
				return
			}
		}
	}
}
