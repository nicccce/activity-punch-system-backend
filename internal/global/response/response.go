package response

import (
	"activity-punch-system/config"
	"activity-punch-system/internal/global/sentry"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ResponseBody 定义了 HTTP 响应的标准 JSON 结构
type ResponseBody struct {
	// Code 是业务状态码，0 表示成功，非 0 表示错误
	Code int32 `json:"code"`
	// Msg 是响应的消息，用于向客户端描述结果或错误
	Msg string `json:"msg"`
	// Origin 是错误来源信息，仅在 debug 模式下返回，便于调试
	Origin string `json:"origin,omitempty"`
	// Data 是响应的数据内容，对于成功响应包含实际数据，错误响应通常为 null
	Data any `json:"data,omitempty"`
	// Timestamp 响应时间戳
	Timestamp int64 `json:"timestamp,omitempty"`
}

// Success 发送成功的 HTTP 响应，状态码为 200
// 参数:
//   - c: gin 上下文，用于发送响应
//   - data: 可选的数据内容，第一个参数会被设置为 ResponseBody.Data
func Success(c *gin.Context, data ...any) {
	response := ResponseBody{
		Code:      success.Code,
		Msg:       success.Message,
		Data:      nil,
		Timestamp: time.Now().Unix(),
	}
	if len(data) > 0 {
		response.Data = data[0]
	}
	c.JSON(200, response)
}

// Fail 发送错误 HTTP 响应，并终止请求处理
// 参数:
//   - c: gin 上下文，用于发送响应
//   - err: 错误对象，优先解析为自定义 *Error 类型，否则包装为内部错误
func Fail(c *gin.Context, err error) {
	var response ResponseBody

	var e *Error
	// 尝试将 err 转换为自定义 *Error 类型
	ok := errors.As(err, &e)
	if !ok {
		// 如果不是 *Error 类型，包装为内部服务错误，并保留原始错误
		e = ErrServerInternal.WithOrigin(err)
	}

	// 设置响应状态码和消息
	response.Code = e.Code
	response.Msg = e.Message
	response.Timestamp = time.Now().Unix()

	// 在 debug 模式下，添加错误来源信息
	if config.IsDebug() {
		response.Origin = e.Origin
	}

	// 将错误对象存储到 gin 上下文中，便于后续处理（如日志记录）
	c.Set(ErrorContextKey, *e)

	// 上报 5xx 错误到 Sentry
	sentry.CaptureException(c, e)

	// 发送 JSON 错误响应
	c.JSON(int(e.Code), response)

	// 终止请求处理链
	c.Abort()
}

// Recovery 是 gin 中间件，用于捕获 panic 并转换为错误响应
// 参数:
//   - c: gin 上下文，用于处理请求
func Recovery(c *gin.Context) {
	// 捕获 panic
	info := recover()
	if info != nil {
		// 尝试将 panic 转换为 error 类型
		err, ok := info.(error)
		if ok {
			// 如果是 error 类型，包装为带堆栈的错误
			Fail(c, errors.WithStack(err))
		} else {
			// 如果不是 error 类型，转换为字符串并创建新错误
			Fail(c, errors.New(fmt.Sprintf("%+v", info)))
		}
	}
}
