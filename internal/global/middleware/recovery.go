package middleware

import (
	"activity-punch-system-backend/internal/global/errs"
	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer errs.Recovery(c)
		c.Next()
	}
}
