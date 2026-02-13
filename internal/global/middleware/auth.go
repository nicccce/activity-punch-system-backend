package middleware

import (
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"fmt"
	"strings"

	sentrylib "github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
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

			// 将用户信息注入 Sentry Scope，后续所有上报自动携带 user_id
			if hub := sentrygin.GetHubFromContext(c); hub != nil {
				hub.ConfigureScope(func(scope *sentrylib.Scope) {
					scope.SetUser(sentrylib.User{
						ID:        fmt.Sprintf("%d", payload.ID),
						Username:  payload.StudentID,
						IPAddress: c.ClientIP(),
					})
					scope.SetTag("user_id", fmt.Sprintf("%d", payload.ID))
					scope.SetTag("student_id", payload.StudentID)
				})
			}
		}
		c.Next()
	}
}
