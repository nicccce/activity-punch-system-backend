package middleware

import (
	"activity-punch-system-backend/internal/global/errs"
	"activity-punch-system-backend/internal/global/jwt"
	"github.com/gin-gonic/gin"
)

func Auth(minRoleID int) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("token")
		if payload, valid := jwt.ParseToken(token); !valid {
			errs.Fail(c, errs.ErrTokenInvalid)
			c.Abort()
			return
		} else if payload.RoleID < minRoleID {
			errs.Fail(c, errs.ErrUnauthorized)
			c.Abort()
			return
		} else {
			c.Set("payload", payload)
		}
		c.Next()
	}
}
