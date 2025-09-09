package tool

import (
	"github.com/gin-gonic/gin"
	"math"
	"strconv"
	"time"
)

type BaseRequest struct {
	AskTime  int64 `json:"ask_time"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// GetPage 获取分页参数 可变参数依次是 defaultOffset, defaultPageSize, maxPageSize
func GetPage(c *gin.Context, defaults ...uint) (offset, limit int) {
	defaultOffset, defaultPageSize, maxPageSize := 0, 30, 300
	if len(defaults) > 0 && defaults[0] <= math.MaxInt {
		defaultOffset = int(defaults[0])
	}
	if len(defaults) > 1 && defaults[1] <= math.MaxInt {
		defaultPageSize = int(defaults[1])
	}
	if len(defaults) > 2 && defaults[2] <= math.MaxInt {
		maxPageSize = int(defaults[2])
	}
	var page int
	var body BaseRequest
	err := c.ShouldBindJSON(&body)
	if err == nil && body.Page > 0 && body.PageSize > 0 {
		limit = body.PageSize
		page = body.Page
	} else {
		pageQ := c.Query("page")
		pageSizeQ := c.Query("page_size")
		limit, err = strconv.Atoi(pageSizeQ)
		if err != nil {
			limit = defaultPageSize
		}
		page, err = strconv.Atoi(pageQ)
		if err != nil {
			page = defaultOffset
		}
	}
	if limit < 1 {
		limit = defaultPageSize
	} else if limit > maxPageSize {
		limit = maxPageSize
	}
	if page < 1 {
		offset = defaultOffset
	} else {
		offset = (page - 1) * limit
		if offset < 0 {
			offset = defaultOffset
		}
	}
	return
}

func GetTime(c *gin.Context) int64 {
	var body BaseRequest
	err := c.ShouldBindJSON(&body)
	if err == nil && body.AskTime > 0 {
		return body.AskTime
	}
	if ts := c.Query("time"); ts != "" {
		ti, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			//response.AddMessage(c, "解析时间戳错误，自动取当前值")
		}
		return ti
	}
	return time.Now().Unix()
}

// Punch todo: 此仅作为占位
type Punch struct{ Score float64 }
