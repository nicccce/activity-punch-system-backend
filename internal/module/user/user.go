package user

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"activity-punch-system/internal/protected/sduLogin"
	"activity-punch-system/tools"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"net/http"
	"strings"
)

// User 定义登录和注册请求的结构体
type User struct {
	StudentID string `json:"student_id" binding:"required"` // 学号，唯一标识用户
	Password  string `json:"password" binding:"required"`   // 密码，登录时验证，注册时加密
}

// Login 处理用户登录请求
func Login(c *gin.Context) {
	// 调用 sduLogin 进行登录验证

	switch config.Get().SduLogin.Mode {
	case "cas":
		// 如果是 CAS 模式，调用 CasLogin 并返回其结果
		casLoginResult := sduLogin.CasLogin(c)
		if !casLoginResult.Success {
			if casLoginResult.RedirectUrl != "" {
				c.Redirect(http.StatusFound, casLoginResult.RedirectUrl)
				return
			}
			response.Fail(c, response.ErrInvalidRequest.WithTips(casLoginResult.Message))
			return
		}
		if casLoginResult.StudentID == "" {
			response.Fail(c, response.ErrInvalidRequest.WithTips("用户信息获取失败"))
			return
		}
		user, token, err := handleLoginSuccess(casLoginResult.StudentID)
		if err != nil {
			response.Fail(c, err)
			return
		}
		if casLoginResult.RedirectUrl != "" {
			if strings.Contains(casLoginResult.RedirectUrl, "?") {
				c.Redirect(http.StatusFound, casLoginResult.RedirectUrl+"&token="+*token)
			} else {
				c.Redirect(http.StatusFound, casLoginResult.RedirectUrl+"?token="+*token)
			}
			return
		}
		response.Success(c, map[string]interface{}{
			"token":      *token,
			"student_id": user.StudentID,
			"role_id":    user.RoleID,
		})
	case "spider":
		response.Fail(c, response.ErrInvalidRequest.WithTips("爬虫登录模式暂不支持"))
	case "default":
		// 定义请求结构体并绑定 JSON 数据
		var req User
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error("绑定登录请求失败", "error", err, "student_id", req.StudentID)
			response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
			return
		}

		// 查询用户是否存在
		var user model.User
		err := database.DB.Where("student_id = ?", req.StudentID).First(&user).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			log.Warn("用户不存在", "student_id", req.StudentID)
			response.Fail(c, response.ErrNotFound.WithTips("用户不存在"))
			return
		case err != nil:
			log.Error("数据库查询失败", "error", err, "student_id", req.StudentID)
			response.Fail(c, response.ErrDatabase.WithOrigin(err))
			return
		}

		// 验证密码
		if !tools.PasswordCompare(req.Password, user.Password) {
			log.Warn("密码错误", "student_id", req.StudentID)
			response.Fail(c, response.ErrInvalidPassword)
			return
		}

		// 记录登录成功的日志
		log.Info("用户登录成功",
			"student_id", user.StudentID,
			"role_id", user.RoleID)

		// 生成 JWT 令牌并返回用户信息
		response.Success(c, map[string]interface{}{
			"token": jwt.CreateToken(jwt.Payload{
				StudentID: user.StudentID,
				RoleID:    user.RoleID,
			}),
			"student_id": user.StudentID,
			"role_id":    user.RoleID,
		})
	default:
		response.Fail(c, response.ErrInvalidRequest.WithTips("登录模式错误"))
	}
}

func handleLoginSuccess(studentID string) (*model.User, *string, error) {
	// 查询用户是否存在
	var user model.User
	err := database.DB.Where("student_id = ?", studentID).First(&user).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 用户不存在，创建新用户
		user = model.User{
			StudentID: studentID,
			RoleID:    1,
			NickName:  "用户" + studentID,
		}
		if err := database.DB.Create(&user).Error; err != nil {
			log.Error("创建用户失败", "error", err, "student_id", studentID)
			return &user, nil, response.ErrDatabase.WithOrigin(err)
		}
	case err != nil:
		log.Error("数据库查询失败", "error", err, "student_id", studentID)
		return &user, nil, response.ErrDatabase.WithOrigin(err)
	}

	// 记录登录成功的日志
	log.Info("用户登录成功",
		"student_id", user.StudentID,
		"role_id", user.RoleID)

	token := jwt.CreateToken(jwt.Payload{
		StudentID: user.StudentID,
		RoleID:    user.RoleID,
	})

	return &user, &token, nil
}
