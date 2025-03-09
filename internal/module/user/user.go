package user

import (
	"activity-punch-system-backend/internal/global/database"
	"activity-punch-system-backend/internal/global/jwt"
	"activity-punch-system-backend/internal/global/response"
	"activity-punch-system-backend/internal/model"
	"activity-punch-system-backend/internal/protected/sdu"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type User struct {
	StudentID string `json:"student_id" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
	var err error
	type LoginReq struct {
		User
	}
	var req LoginReq
	if err = c.ShouldBind(&req); err != nil {
		response.Fail(c, response.ErrInvalidRequest.WithOrigin(err))
		return
	}
	var sduLoginData *sdu.LoginData
	if sduLoginData, err = sdu.Login(req.StudentID, req.Password); err != nil {
		log.Info("Wrong Password", "StudentID", req.StudentID, "Password", req.Password)
		response.Fail(c, response.ErrInvalidPassword.WithOrigin(err))
		return
	}

	var user model.User
	err = database.DB.Take(&user, "student_id=?", sduLoginData.StudentID).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		user.StudentID = sduLoginData.StudentID
		user.RealName = sduLoginData.RealName
		user.RoleID = 1
		if err := database.DB.Create(&user).Error; err != nil {
			response.Fail(c, response.ErrDatabase.WithOrigin(err))
			return
		}
		log.Info("First Login", "Student_id", sduLoginData.StudentID, "RealName", sduLoginData.RealName)
	case err != nil:
		response.Fail(c, response.ErrDatabase.WithOrigin(err))
		return
	default:
		log.Info("Login Success", "StudentID", sduLoginData.StudentID, "RealName", sduLoginData.RealName)
	}

	response.Success(c, map[string]string{
		"token": jwt.CreateToken(jwt.Payload{
			StudentID: user.StudentID,
			RoleID:    user.RoleID,
		}),
	})
}
