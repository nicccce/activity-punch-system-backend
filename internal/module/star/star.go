package star

import (
	"activity-punch-system/internal/global/context"
	"activity-punch-system/internal/global/database"
	"activity-punch-system/internal/global/response"
	"activity-punch-system/internal/model"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

func add(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	punchIDstr := c.Query("punch_id")
	punchID, err := validPunchId(punchIDstr)
	if err != nil {
		response.Fail(c, err)
		return
	}
	var exist bool
	err = database.DB.
		Raw("SELECT EXISTS(SELECT 1 FROM punch WHERE id = ? )",
			punchIDstr).
		Scan(&exist).Error
	if err != nil {
		response.Fail(c, response.ErrDatabase)
	} else if !exist {
		response.Fail(c, response.ErrNotFound)
		return
	}
	star := &model.Star{
		UserID:  user.StudentID,
		PunchID: punchID,
	}
	if err = database.DB.Create(star).Error; err != nil {
		//perhaps
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "Duplicate") || strings.Contains(err.Error(), "DUPLICATE") {
			response.Fail(c, response.ErrAlreadyExists)
		} else {
			response.Fail(c, response.ErrDatabase)
		}
	} else {
		response.Success(c)
	}
}
func list(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	pageQ := c.Query("page")
	pageSizeQ := c.Query("page_size")
	pageSize, err := strconv.Atoi(pageSizeQ)
	if err != nil || pageSize <= 0 {
		pageSize = 30
	}
	page, err := strconv.Atoi(pageQ)
	if err != nil || page <= 0 {
		page = 1
	}
	var stars []model.Star
	if err := database.DB.Model(&model.Star{}).
		Preload("Punch").
		Where("user_id = ?", user.StudentID).
		Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&stars).Error; err != nil {
		response.Fail(c, response.ErrDatabase)
	} else {
		response.Success(c, struct {
			UserId   string       `json:"user_id"`
			PageSize int          `json:"page_size"`
			Page     int          `json:"page"`
			Stars    []model.Star `json:"stars"`
		}{
			UserId:   user.StudentID,
			PageSize: pageSize,
			Page:     page,
			Stars:    stars,
		})
	}
}
func cancel(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	punchIDstr := c.Query("punch_id")
	_, err := validPunchId(punchIDstr)
	if err != nil {
		response.Fail(c, err)
		return
	}
	dere := database.DB.Where("user_id = ? AND punch_id = ?", user.StudentID, punchIDstr).Delete(&model.Star{})
	if dere.Error != nil {
		response.Fail(c, response.ErrDatabase)
	} else if dere.RowsAffected == 0 {
		response.Fail(c, response.ErrNotFound)
	} else {
		response.Success(c)
	}
}
func ask(c *gin.Context) {
	user, ok := context.GetUserPayload(c)
	if !ok {
		response.Fail(c, response.ErrUnauthorized)
		return
	}
	punchIDstr := c.Query("punch_id")
	if punchIDstr == "" {
		response.Fail(c, response.ErrInvalidRequest)
		return
	}
	if _, err := strconv.ParseUint(punchIDstr, 10, 0); err != nil {
		response.Fail(c, response.ErrInvalidRequest)
		return
	}
	var exist bool
	err := database.DB.
		Raw("SELECT EXISTS(SELECT 1 FROM punch WHERE id = ? )",
			punchIDstr).Scan(&exist).Error
	if err != nil {
		response.Fail(c, response.ErrDatabase)
	} else if !exist {
		response.Fail(c, response.ErrNotFound)
	}
	err = database.DB.
		Raw("SELECT EXISTS(SELECT 1 FROM star WHERE user_id = ? AND punch_id = ? )",
			user.StudentID, punchIDstr).
		Scan(&exist).Error
	if err != nil {
		response.Fail(c, response.ErrDatabase)
	} else {
		response.Success(c, exist)
	}
}
func validPunchId(punchIDstr string) (uint, error) {
	if punchIDstr == "" {
		return 0, response.ErrInvalidRequest
	}
	punchID, err := strconv.ParseUint(punchIDstr, 10, 0)
	if err != nil {
		return 0, response.ErrInvalidRequest
	}
	return uint(punchID), nil
}

// count 此处不验证punch记录是否存在
func count(c *gin.Context) {
	punchIDstr := c.Query("punch_id")
	if punchIDstr == "" {
		response.Fail(c, response.ErrInvalidRequest)
		return
	}
	if _, err := strconv.ParseUint(punchIDstr, 10, 0); err != nil {
		response.Fail(c, response.ErrInvalidRequest)
		return
	}
	var sum int64
	err := database.DB.
		Model(&model.Star{}).
		Where("punch_id = ?", punchIDstr).
		Count(&sum).Error
	if err != nil {
		response.Fail(c, response.ErrDatabase)
	} else {
		response.Success(c, sum)
	}
}
