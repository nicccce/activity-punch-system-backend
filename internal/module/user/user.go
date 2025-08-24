package user

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/jwt"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"activity-punch-system/internal/protected/sduLogin"
	"activity-punch-system/tools"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
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

// validatePasswordStrength 验证密码强度
func validatePasswordStrength(password string) error {
	if password == "" {
		return errors.New("密码不能为空")
	}
	if len(password) < 8 {
		return errors.New("密码长度必须至少8字符")
	}

	// 检查是否包含至少一个字母
	hasLetter := false
	// 检查是否包含至少一个数字
	hasDigit := false
	// 检查是否包含至少一个特殊字符
	hasSpecial := false
	specialChars := "!@#$%^&*-"

	for _, char := range password {
		switch {
		case strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", char):
			hasLetter = true
		case strings.ContainsRune("0123456789", char):
			hasDigit = true
		case strings.ContainsRune(specialChars, char):
			hasSpecial = true
		}
	}

	if !hasLetter {
		return errors.New("密码必须包含至少一个字母")
	}
	if !hasDigit {
		return errors.New("密码必须包含至少一个数字")
	}
	if !hasSpecial {
		return errors.New("密码必须包含至少一个特殊字符（!@#$%^&*）")
	}

	return nil
}

type registerReq struct {
	User
	NickName string `json:"nick_name" binding:"required"`
}

// Register 处理用户注册请求
func Register(c *gin.Context) {
	if config.Get().SduLogin.Mode != "default" {
		response.Fail(c, response.ErrInvalidRequest.WithTips("当前登录模式不支持"))
		return
	}
	// 定义请求结构体并绑定 JSON 数据
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定注册请求失败", "error", err, "student_id", req.StudentID)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 验证密码强度
	if err := validatePasswordStrength(req.Password); err != nil {
		log.Warn("密码强度验证失败", "error", err, "student_id", req.StudentID)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err).WithTips(err.Error()))
		return
	}

	// 检查学号是否已存在
	var existingUser model.User
	err := database.DB.Where("student_id = ?", req.StudentID).First(&existingUser).Error
	if err == nil {
		log.Warn("用户已存在", "student_id", req.StudentID)
		response.Fail(c, response.ErrAlreadyExists)
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error("数据库查询失败", "error", err, "student_id", req.StudentID)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 加密密码
	encryptedPassword := tools.PasswordEncrypt(req.Password)

	// 创建新的用户
	user := model.User{
		StudentID: req.StudentID,
		Password:  encryptedPassword,
		NickName:  req.NickName,
		RoleID:    1, // 默认角色 ID，可根据需求调整
	}

	// 保存用户到数据库
	if err := database.DB.Create(&user).Error; err != nil {
		log.Error("创建用户失败", "error", err, "student_id", req.StudentID)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 记录注册成功的日志
	log.Info("用户注册成功",
		"student_id", user.StudentID,
		"nick_name", user.NickName,
		"role_id", user.RoleID)

	// 返回成功响应
	response.Success(c)
}

// ChangePasswordReq 定义修改密码请求的结构体
type ChangePasswordReq struct {
	OldPassword string `json:"old_password" binding:"required"` // 旧密码，用于验证
	NewPassword string `json:"new_password" binding:"required"` // 新密码，需加密后保存
}

// ChangePassword 处理用户修改密码请求
// 验证旧密码正确性后更新新密码，要求用户已通过认证
// 参数:
//   - c: gin 上下文，用于接收请求和发送响应
func ChangePassword(c *gin.Context) {
	if config.Get().SduLogin.Mode != "default" {
		response.Fail(c, response.ErrInvalidRequest.WithTips("当前登录模式不支持"))
		return
	}
	// 获取认证信息
	payload, exists := c.Get("payload")
	if !exists {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	userPayload, ok := payload.(*jwt.Claims)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}

	// 定义请求结构体并绑定 JSON 数据
	var req ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("绑定修改密码请求失败", "error", err, "student_id", userPayload.StudentID)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 验证新密码强度
	if err := validatePasswordStrength(req.NewPassword); err != nil {
		log.Warn("新密码强度验证失败", "error", err, "student_id", userPayload.StudentID)
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}

	// 查询用户
	var user model.User
	err := database.DB.Where("student_id = ?", userPayload.StudentID).First(&user).Error
	if err != nil {
		log.Error("查询用户失败", "error", err, "student_id", userPayload.StudentID)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 验证旧密码
	if !tools.PasswordCompare(req.OldPassword, user.Password) {
		log.Warn("旧密码错误", "student_id", userPayload.StudentID)
		response.Fail(c, response.ErrInvalidPassword)
		return
	}

	// 加密新密码
	newEncryptedPassword := tools.PasswordEncrypt(req.NewPassword)

	// 更新用户密码
	if err := database.DB.Model(&user).Update("password", newEncryptedPassword).Error; err != nil {
		log.Error("更新密码失败", "error", err, "student_id", userPayload.StudentID)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}

	// 记录修改密码成功的日志
	log.Info("用户修改密码成功",
		"student_id", user.StudentID,
		"nick_name", user.NickName,
		"role_id", user.RoleID)

	// 返回成功响应
	response.Success(c, nil)
}

func getMe(c *gin.Context) {
	// 获取认证信息
	payload, exists := c.Get("payload")
	if !exists {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	userPayload, ok := payload.(*jwt.Claims)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	// 查询用户
	var user model.User
	err := database.DB.Where("student_id = ?", userPayload.StudentID).First(&user).Error
	if err != nil {
		log.Error("查询用户失败", "error", err, "student_id", userPayload.StudentID)
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	}
	response.Success(c, user)
}
