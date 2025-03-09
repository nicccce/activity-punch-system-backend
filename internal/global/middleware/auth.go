package middleware

import (
	"activity-punch-system-backend/internal/global/jwt"
	"activity-punch-system-backend/internal/global/response"
	"github.com/gin-gonic/gin"
)

func Auth(minRoleID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("token")
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
