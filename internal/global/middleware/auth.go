package middleware

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"strings"

	"github.com/gin-gonic/gin"
)

func Auth(minRoleID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		// if config.Get().Mode == "debug" {
		// 	c.Next()
		// 	return
		// }
		// 获取 Authorization 头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Fail(c, response.ErrTokenInvalid)
			c.Abort()
			return
		}

		// 检查 Bearer 前缀并提取 token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			response.Fail(c, response.ErrTokenInvalid)
			c.Abort()
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// 解析 token
		if payload, valid := jwt.ParseToken(token); !valid {
			response.Fail(c, response.ErrTokenInvalid)
			c.Abort()
			return
		} else if payload.RoleID < minRoleID {
			response.Fail(c, response.ErrUnauthorized)
			c.Abort()
			return
		} else {
			c.Set("payload", payload)
		}
		c.Next()
	}
}
