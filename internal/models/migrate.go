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
		&CommentLike{}, // Add CommentLike here
		&Achievement{},
		&AIConversation{},
		&PracticeRoom{},
		&PracticeRoomMember{},
		&Follow{},
		&PostCollection{},
	)
}






