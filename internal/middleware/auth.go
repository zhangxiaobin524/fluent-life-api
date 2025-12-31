package middleware

import (
	"fluent-life-backend/internal/utils"
	"strings"

	"fluent-life-backend/pkg/auth"
	"fluent-life-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		utils.APILog("[Auth Middleware] 收到请求: %s %s", c.Request.Method, c.Request.URL.Path)
		
		// 允许 OPTIONS 预检请求通过（CORS 预检）
		if c.Request.Method == "OPTIONS" {
			utils.APILog("[Auth Middleware] OPTIONS 预检请求，直接通过")
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.APILog("[Auth Middleware] ❌ 未提供认证令牌")
			response.Unauthorized(c, "未提供认证令牌")
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.APILog("[Auth Middleware] ❌ 认证令牌格式错误，Header: %s", authHeader)
			response.Unauthorized(c, "认证令牌格式错误")
			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(parts[1], secret)
		if err != nil {
			utils.APILog("[Auth Middleware] ❌ 无效的认证令牌: %v", err)
			response.Unauthorized(c, "无效的认证令牌")
			c.Abort()
			return
		}

		utils.APILog("[Auth Middleware] ✅ 认证成功，用户ID: %s", claims.UserID)
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
