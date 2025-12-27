package services

import (
	"errors"
	"fmt"
	"time"

	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/models"
	"fluent-life-backend/pkg/auth"
	"fluent-life-backend/pkg/validator"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db                      *gorm.DB
	cfg                     *config.Config
	verificationCodeService *VerificationCodeService
}

func NewAuthService(db *gorm.DB, cfg *config.Config) *AuthService {
	return &AuthService{
		db:                      db,
		cfg:                     cfg,
		verificationCodeService: NewVerificationCodeService(db, cfg),
	}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *AuthService) VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func (s *AuthService) Register(username, identifier, password, code string) (*models.User, string, error) {
	// 验证输入
	if !validator.IsEmailOrPhone(identifier) {
		return nil, "", errors.New("邮箱或手机号格式不正确")
	}
	if !validator.ValidatePassword(password) {
		return nil, "", errors.New("密码至少需要6个字符")
	}

	// 验证验证码
	/*
		if err := s.verificationCodeService.ValidateCode(identifier, code, "register"); err != nil {
			return nil, "", err
		}*/

	// 检查用户名是否已存在
	var existingUser models.User
	if err := s.db.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return nil, "", errors.New("用户名已存在")
	}

	// 检查邮箱或手机号是否已注册
	var existingIdentifier models.User
	if validator.IsEmail(identifier) {
		if err := s.db.Where("email = ?", identifier).First(&existingIdentifier).Error; err == nil {
			return nil, "", errors.New("该邮箱已被注册")
		}
	} else {
		if err := s.db.Where("phone = ?", identifier).First(&existingIdentifier).Error; err == nil {
			return nil, "", errors.New("该手机号已被注册")
		}
	}

	// 加密密码
	passwordHash, err := s.HashPassword(password)
	if err != nil {
		return nil, "", errors.New("密码加密失败")
	}

	// 创建用户
	user := models.User{
		Username:     username,
		PasswordHash: passwordHash,
	}

	if validator.IsEmail(identifier) {
		user.Email = &identifier
	} else {
		user.Phone = &identifier
	}

	// 生成头像URL
	avatarURL := fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s", username)
	user.AvatarURL = &avatarURL

	if err := s.db.Create(&user).Error; err != nil {
		return nil, "", errors.New("创建用户失败")
	}

	// 生成Token
	token, err := auth.GenerateToken(user.ID, s.cfg.JWTSecret, s.cfg.JWTExpiration)
	if err != nil {
		return nil, "", errors.New("生成Token失败")
	}

	return &user, token, nil
}

func (s *AuthService) Login(identifier, password string) (*models.User, string, error) {
	// 查找用户
	var user models.User
	if validator.IsEmail(identifier) {
		if err := s.db.Where("email = ?", identifier).First(&user).Error; err != nil {
			return nil, "", errors.New("邮箱或密码错误")
		}
	} else if validator.IsPhone(identifier) {
		if err := s.db.Where("phone = ?", identifier).First(&user).Error; err != nil {
			return nil, "", errors.New("手机号或密码错误")
		}
	} else {
		return nil, "", errors.New("邮箱或手机号格式不正确")
	}

	// 验证密码
	if !s.VerifyPassword(user.PasswordHash, password) {
		return nil, "", errors.New("邮箱或密码错误")
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Save(&user)

	// 生成Token
	token, err := auth.GenerateToken(user.ID, s.cfg.JWTSecret, s.cfg.JWTExpiration)
	if err != nil {
		return nil, "", errors.New("生成Token失败")
	}

	return &user, token, nil
}
