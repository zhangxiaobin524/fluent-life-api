package models

import "gorm.io/gorm"

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&VerificationCode{},
		&TrainingRecord{},
		&MeditationProgress{},
		&Post{},
		&PostLike{},
		&Comment{},
		&Achievement{},
		&AIConversation{},
		&PracticeRoom{},
		&PracticeRoomMember{},
	)
}






