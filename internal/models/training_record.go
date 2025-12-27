package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), j)
	}
	return json.Unmarshal(bytes, j)
}

type TrainingRecord struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_training_records_user_id" json:"user_id"`
	Type      string    `gorm:"type:varchar(20);not null;index:idx_training_records_type" json:"type"` // 'meditation' | 'airflow' | 'exposure' | 'practice'
	Duration  int       `gorm:"not null" json:"duration"`                                              // 秒
	Data      JSONB     `gorm:"type:jsonb;index:,type:gin" json:"data,omitempty"`
	Timestamp time.Time `gorm:"not null;index:idx_training_records_timestamp;index:idx_training_records_user_timestamp" json:"timestamp"`
	CreatedAt time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (t *TrainingRecord) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}






