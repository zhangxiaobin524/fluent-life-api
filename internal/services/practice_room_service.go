package services

import (
	"fluent-life-backend/internal/models"
	"fluent-life-backend/internal/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PracticeRoomService struct {
	db *gorm.DB
}

func NewPracticeRoomService(db *gorm.DB) *PracticeRoomService {
	return &PracticeRoomService{db: db}
}

// CreateRoom 创建对练房
func (s *PracticeRoomService) CreateRoom(userID uuid.UUID, title, theme, roomType string) (*models.PracticeRoom, error) {
	utils.APILog("[PracticeRoomService.CreateRoom] 开始创建房间，用户ID: %s, 标题: %s, 主题: %s, 类型: %s", userID.String(), title, theme, roomType)

	room := &models.PracticeRoom{
		UserID:         userID,
		Title:          title,
		Theme:          theme,
		Type:           roomType,
		MaxMembers:     7,
		CurrentMembers: 1,
		IsActive:       true,
	}

	utils.APILog("[PracticeRoomService.CreateRoom] 准备创建房间记录到数据库")
	if err := s.db.Create(room).Error; err != nil {
		utils.APILog("[PracticeRoomService.CreateRoom] ❌ 创建房间记录失败: %v", err)
		return nil, err
	}
	utils.APILog("[PracticeRoomService.CreateRoom] ✅ 房间记录创建成功，房间ID: %s", room.ID.String())

	// 创建房主成员记录
	member := &models.PracticeRoomMember{
		RoomID:   room.ID,
		UserID:   room.UserID,
		IsHost:   true,
		JoinedAt: time.Now(),
	}
	utils.APILog("[PracticeRoomService.CreateRoom] 准备创建房主成员记录，房间ID: %s, 用户ID: %s", room.ID.String(), room.UserID.String())
	if err := s.db.Create(member).Error; err != nil {
		utils.APILog("[PracticeRoomService.CreateRoom] ❌ 创建房主成员记录失败: %v", err)
		return nil, err
	}
	utils.APILog("[PracticeRoomService.CreateRoom] ✅ 房主成员记录创建成功")

	return room, nil
}

// GetRooms 获取房间列表
func (s *PracticeRoomService) GetRooms(page, pageSize int, theme, roomType string) ([]models.PracticeRoom, int64, error) {
	var rooms []models.PracticeRoom
	var total int64

	query := s.db.Model(&models.PracticeRoom{}).Where("is_active = ?", true)

	if theme != "" {
		query = query.Where("theme = ?", theme)
	}
	if roomType != "" {
		query = query.Where("type = ?", roomType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Preload("User").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rooms).Error; err != nil {
		return nil, 0, err
	}

	return rooms, total, nil
}

// GetRoom 获取房间详情
func (s *PracticeRoomService) GetRoom(roomID uuid.UUID) (*models.PracticeRoom, error) {
	var room models.PracticeRoom
	if err := s.db.Preload("User").Preload("Members.User").First(&room, "id = ?", roomID).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

// JoinRoom 加入房间
func (s *PracticeRoomService) JoinRoom(roomID, userID uuid.UUID) error {
	var room models.PracticeRoom
	if err := s.db.First(&room, "id = ?", roomID).Error; err != nil {
		return err
	}

	if !room.IsActive {
		return gorm.ErrRecordNotFound
	}

	if room.CurrentMembers >= room.MaxMembers {
		return gorm.ErrRecordNotFound // 房间已满
	}

	// 检查是否已经是成员
	var existingMember models.PracticeRoomMember
	if err := s.db.Where("room_id = ? AND user_id = ?", roomID, userID).First(&existingMember).Error; err == nil {
		// 如果用户是房间创建者，确保是房主
		if room.UserID == userID && !existingMember.IsHost {
			existingMember.IsHost = true
			if err := s.db.Save(&existingMember).Error; err != nil {
				return err
			}
		}
		return nil // 已经是成员
	}

	// 检查用户是否是房间创建者（房主）
	isHost := room.UserID == userID

	// 添加成员
	member := &models.PracticeRoomMember{
		RoomID:   roomID,
		UserID:   userID,
		IsHost:   isHost, // 如果是房间创建者，设置为房主
		JoinedAt: time.Now(),
	}
	if err := s.db.Create(member).Error; err != nil {
		return err
	}

	// 更新房间成员数
	room.CurrentMembers++
	return s.db.Save(&room).Error
}

// LeaveRoom 离开房间
func (s *PracticeRoomService) LeaveRoom(roomID, userID uuid.UUID) error {
	// 先检查离开的用户是否是房主
	var leavingMember models.PracticeRoomMember
	if err := s.db.Where("room_id = ? AND user_id = ?", roomID, userID).First(&leavingMember).Error; err != nil {
		return err
	}

	isHost := leavingMember.IsHost

	// 删除成员记录
	if err := s.db.Where("room_id = ? AND user_id = ?", roomID, userID).Delete(&models.PracticeRoomMember{}).Error; err != nil {
		return err
	}

	var room models.PracticeRoom
	if err := s.db.First(&room, "id = ?", roomID).Error; err != nil {
		return err
	}

	// 更新成员数
	if room.CurrentMembers > 0 {
		room.CurrentMembers--
	}

	// 查询剩余成员数（在删除后）
	var remainingMemberCount int64
	s.db.Model(&models.PracticeRoomMember{}).Where("room_id = ?", roomID).Count(&remainingMemberCount)

	// 如果离开的是房主，需要处理房主转移或关闭房间
	if isHost {
		// 查询剩余成员
		var remainingMembers []models.PracticeRoomMember
		if err := s.db.Where("room_id = ?", roomID).Find(&remainingMembers).Error; err != nil {
			return err
		}

		// 如果有剩余成员，需要转移房主或关闭房间
		// 注意：这里我们无法直接知道谁在麦上，需要在调用方传入上麦用户列表
		// 暂时先检查是否有剩余成员
		if len(remainingMembers) > 0 {
			// 有剩余成员，但需要调用方决定是否转移房主
			// 这里只更新房间信息，不关闭
			room.CurrentMembers = int(remainingMemberCount)
		} else {
			// 没有剩余成员，关闭房间
			room.IsActive = false
			room.CurrentMembers = 0
		}
	} else {
		// 更新当前成员数为实际剩余成员数
		room.CurrentMembers = int(remainingMemberCount)
		// 如果成员数为0，关闭房间
		if room.CurrentMembers == 0 {
			room.IsActive = false
		}
	}

	return s.db.Save(&room).Error
}

// TransferHost 转移房主
func (s *PracticeRoomService) TransferHost(roomID, newHostUserID uuid.UUID) error {
	// 取消所有成员的房主身份
	if err := s.db.Model(&models.PracticeRoomMember{}).
		Where("room_id = ?", roomID).
		Update("is_host", false).Error; err != nil {
		return err
	}

	// 设置新房主
	if err := s.db.Model(&models.PracticeRoomMember{}).
		Where("room_id = ? AND user_id = ?", roomID, newHostUserID).
		Update("is_host", true).Error; err != nil {
		return err
	}

	// 更新房间的 UserID 为新房主
	if err := s.db.Model(&models.PracticeRoom{}).
		Where("id = ?", roomID).
		Update("user_id", newHostUserID).Error; err != nil {
		return err
	}

	return nil
}

// GetRoomMemberCount 获取房间成员数
func (s *PracticeRoomService) GetRoomMemberCount(roomID uuid.UUID) (int, error) {
	var count int64
	if err := s.db.Model(&models.PracticeRoomMember{}).Where("room_id = ?", roomID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}
