package services

import (
	"fluent-life-backend/internal/models"

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
	room := &models.PracticeRoom{
		UserID:         userID,
		Title:          title,
		Theme:          theme,
		Type:           roomType,
		MaxMembers:     2,
		CurrentMembers: 1,
		IsActive:       true,
	}

	if err := s.db.Create(room).Error; err != nil {
		return nil, err
	}

	// 创建房主成员记录
	member := &models.PracticeRoomMember{
		RoomID: room.ID,
		UserID: room.UserID,
		IsHost: true,
	}
	if err := s.db.Create(member).Error; err != nil {
		return nil, err
	}

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
		return nil // 已经是成员
	}

	// 添加成员
	member := &models.PracticeRoomMember{
		RoomID: roomID,
		UserID: userID,
		IsHost: false,
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

	// 如果成员数为0，关闭房间
	if room.CurrentMembers == 0 {
		room.IsActive = false
	}

	return s.db.Save(&room).Error
}
