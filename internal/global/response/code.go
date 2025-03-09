package response

import "net/http"

// 200 OK
var (
	success = newError(200, "Success")
)

// 400 Bad Request
var (
	ErrInvalidRequest  = newError(http.StatusBadRequest, "无效的请求")     // 400 Bad Request
	ErrInvalidPassword = newError(http.StatusBadRequest, "账号或密码错误") // 400 Bad Request
	ErrTokenInvalid    = newError(http.StatusBadRequest, "无效的token")    // 400 Bad Request
)

// 401 Unauthorized
var (
	ErrUnauthorized = newError(http.StatusUnauthorized, "权限不足") // 401 Unauthorized
)

// 403 Forbidden
var (
	ErrForbidden = newError(http.StatusForbidden, "禁止访问") // 403 Forbidden
)

// 404 Not Found
var (
	ErrNotFound = newError(http.StatusNotFound, "目标不存在") // 404 Not Found
)

// 409 Conflict
var (
	ErrAlreadyExists = newError(http.StatusConflict, "目标已存在") // 409 Conflict
)

// 500 Internal Server Error
var (
	ErrServerInternal = newError(http.StatusInternalServerError, "服务器内部错误") // 500 Internal Server Error
	ErrDatabase       = newError(http.StatusInternalServerError, "数据库错误")     // 500 Internal Server Error
)
