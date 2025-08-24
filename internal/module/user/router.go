package user

import (
	"activity-punch-system/internal/global/middleware"

	"github.com/gin-gonic/gin"
)

// InitRouter 初始化用户模块的路由
// 将用户相关的 HTTP 端点挂载到指定的路由组
// 该方法会在模块初始化时被调用
// 参数:
//   - r: gin.RouterGroup，表示父路由组，用于挂载子路由
func (u *ModuleUser) InitRouter(r *gin.RouterGroup) {
	// 定义用户模块的路由组，所有用户相关端点以 /user 为前缀
	userGroup := r.Group("/user")

	// 注册登录端点，处理用户登录请求
	userGroup.POST("/login", Login)

	// 注册注册端点，处理用户注册请求
	userGroup.POST("/register", Register)

	userGroup.Use(middleware.Auth(0))
	{
		// 注册获取用户信息端点，处理修改密码请求
		userGroup.POST("/change-password", ChangePassword)
		userGroup.GET("/profile", getMe)
		userGroup.PUT("/update", updateUser)
	}

}
