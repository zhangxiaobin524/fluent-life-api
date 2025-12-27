package services

import (
	"fmt"
	"math/rand"
	"time"

	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/models"

	"gorm.io/gorm"
)

type VerificationCodeService struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewVerificationCodeService(db *gorm.DB, cfg *config.Config) *VerificationCodeService {
	return &VerificationCodeService{db: db, cfg: cfg}
}

func (s *VerificationCodeService) GenerateCode() string {
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func (s *VerificationCodeService) SendCode(identifier, codeType string) error {
	// 检查1分钟内是否已发送
	var recentCode models.VerificationCode
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	err := s.db.Where("identifier = ? AND type = ? AND created_at > ?", identifier, codeType, oneMinuteAgo).First(&recentCode).Error
	if err == nil {
		return fmt.Errorf("请稍后再试，1分钟内只能发送一次")
	}

	// 生成验证码
	code := s.GenerateCode()
	expiresAt := time.Now().Add(s.cfg.CodeExpiration)

	// 保存验证码
	verificationCode := models.VerificationCode{
		Identifier: identifier,
		Code:       code,
		Type:       codeType,
		ExpiresAt:  expiresAt,
		Used:       false,
	}

	if err := s.db.Create(&verificationCode).Error; err != nil {
		return fmt.Errorf("保存验证码失败: %w", err)
	}

	// 发送验证码（开发环境直接打印，生产环境接入短信/邮件服务）
	if s.cfg.Environment == "development" {
		fmt.Printf("[开发环境] 验证码已发送到 %s: %s (有效期5分钟)\n", identifier, code)
	} else {
		// TODO: 接入真实的短信/邮件服务
		// if IsEmail(identifier) {
		//     sendEmail(identifier, code)
		// } else {
		//     sendSMS(identifier, code)
		// }
	}

	return nil
}

func (s *VerificationCodeService) ValidateCode(identifier, code, codeType string) error {
	var verificationCode models.VerificationCode
	err := s.db.Where("identifier = ? AND code = ? AND type = ? AND used = false", identifier, code, codeType).
		First(&verificationCode).Error
	if err != nil {
		return fmt.Errorf("验证码无效")
	}

	if time.Now().After(verificationCode.ExpiresAt) {
		return fmt.Errorf("验证码已过期")
	}

	// 标记为已使用
	verificationCode.Used = true
	s.db.Save(&verificationCode)

	return nil
}






