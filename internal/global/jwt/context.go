package jwt

import (
	"github.com/gin-gonic/gin"
)

func GetUserPayload(c *gin.Context) (userPayload *Claims, exist bool) {
	payload, _ := c.Get("payload")
	userPayload, exist = payload.(*Claims)
	return
}
