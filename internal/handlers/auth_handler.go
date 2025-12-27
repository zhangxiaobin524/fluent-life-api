package handlers

import (
	"fluent-life-backend/internal/config"
	"fluent-life-backend/internal/services"
	"fluent-life-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AuthHandler struct {
	authService *services.AuthService
	codeService *services.VerificationCodeService
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authService: services.NewAuthService(db, cfg),
		codeService: services.NewVerificationCodeService(db, cfg),
	}
}

type SendCodeRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Type       string `json:"type" binding:"required,oneof=register login"`
}

type RegisterRequest struct {
	Username   string `json:"username" binding:"required"`
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required,min=6"`
	Code       string `json:"code" binding:"required,len=6"`
}

type LoginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

func (h *AuthHandler) SendCode(c *gin.Context) {
	var req SendCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.codeService.SendCode(req.Identifier, req.Type); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, nil, "验证码已发送")
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, token, err := h.authService.Register(req.Username, req.Identifier, req.Password, req.Code)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"user":  user,
		"token": token,
	}, "注册成功")
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, token, err := h.authService.Login(req.Identifier, req.Password)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"user":  user,
		"token": token,
	}, "登录成功")
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// TODO: 实现Token刷新逻辑
	response.Success(c, gin.H{"token": req.Token}, "Token刷新成功")
}






