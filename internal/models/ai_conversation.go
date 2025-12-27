package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"` // 'user' | 'bot'
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type Messages []Message

func (m Messages) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m *Messages) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), m)
	}
	return json.Unmarshal(bytes, m)
}

type AIConversation struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_ai_conversations_user_id" json:"user_id"`
	Messages  Messages  `gorm:"type:jsonb;index:,type:gin" json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (a *AIConversation) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}






