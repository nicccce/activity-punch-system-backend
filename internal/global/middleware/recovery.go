package middleware

import (
	"activity-punch-system/internal/global/response"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer response.Recovery(c)
		c.Next()
	}
}
