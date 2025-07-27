package context

import (
	"activity-punch-system/internal/global/jwt"
	"github.com/gin-gonic/gin"
)

func GetUserPayload(c *gin.Context) (userPayload *jwt.Claims, exist bool) {
	payload, _ := c.Get("payload")
	userPayload, exist = payload.(*jwt.Claims)
	return
}
