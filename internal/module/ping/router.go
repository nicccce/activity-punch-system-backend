package ping

import (
	"activity-punch-system/internal/global/response"

	"github.com/gin-gonic/gin"
)

func (p *ModulePing) InitRouter(r *gin.RouterGroup) {
	r.GET("/ping", func(c *gin.Context) {
		result := map[string]interface{}{
			"message": "pong",
			"version": "1.0.0",
		}
		response.Success(c, result)
	})
}
