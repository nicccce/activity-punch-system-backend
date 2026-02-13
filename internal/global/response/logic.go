package response

import (
	"errors"
	"fmt"

	pkgerrors "github.com/pkg/errors"
)

// ErrorContextKey 是用于在 gin.Context 中存储错误对象的键
const ErrorContextKey = "error"

// ResponseContextKey 是用于在 gin.Context 中存储响应体的键，供 Sentry 上报使用
const ResponseContextKey = "response_body"

// Error 自定义错误类型，支持错误码、消息、原始错误链和堆栈跟踪
type Error struct {
	Code    int32  `json:"code"`
	Message string `json:"msg"`
	Origin  string `json:"origin"`
	// cause 保存原始错误，用于 Unwrap() 方法和 Sentry 堆栈提取
	cause error
	// stack 保存堆栈信息，用于 Sentry 堆栈提取
	stack pkgerrors.StackTrace
}

func newError(code int32, msg string) *Error {
	return &Error{
		Code:    code,
		Message: msg,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("code:%d, msg:%s", e.Code, e.Message)
}

// GetCode 返回错误码，实现 sentry.CodedError 接口
func (e *Error) GetCode() int32 {
	return e.Code
}

// Unwrap 返回原始错误，支持 errors.Unwrap() 和 Sentry 错误链提取
func (e *Error) Unwrap() error {
	return e.cause
}

// StackTrace 返回堆栈跟踪，支持 Sentry 堆栈提取
// 实现 pkg/errors 的 stackTracer 接口
func (e *Error) StackTrace() pkgerrors.StackTrace {
	if e.stack != nil {
		return e.stack
	}
	// 如果原始错误有堆栈，提取它
	if e.cause != nil {
		type stackTracer interface {
			StackTrace() pkgerrors.StackTrace
		}
		if st, ok := e.cause.(stackTracer); ok {
			return st.StackTrace()
		}
	}
	return nil
}

func (e *Error) Is(target error) bool {
	var t *Error
	ok := errors.As(target, &t)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// WithOrigin 向前端返回用于调试的原始错误（仅限 config.DebugMode）
// 同时保留原始错误链，以便 Sentry 能够提取堆栈信息
func (e *Error) WithOrigin(err error) *Error {
	if err == nil {
		return e
	}

	// 确保错误带有堆栈信息
	wrappedErr := ensureStack(err)

	newErr := &Error{
		Code:    e.Code,
		Message: e.Message,
		Origin:  fmt.Sprintf("%+v", wrappedErr),
		cause:   wrappedErr,
	}

	// 提取堆栈信息
	type stackTracer interface {
		StackTrace() pkgerrors.StackTrace
	}
	if st, ok := wrappedErr.(stackTracer); ok {
		newErr.stack = st.StackTrace()
	}

	return newErr
}

// WithTips 向前端返回额外的提示信息（config.ReleaseMode 也可见）
func (e *Error) WithTips(details ...string) *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message + " " + fmt.Sprintf("%v", details),
		cause:   e.cause,
		stack:   e.stack,
	}
}

// ensureStack 确保错误带有堆栈信息
// 如果错误已经有堆栈，直接返回；否则添加堆栈
func ensureStack(err error) error {
	if err == nil {
		return nil
	}
	// 检查错误是否已经有堆栈
	type stackTracer interface {
		StackTrace() pkgerrors.StackTrace
	}
	if _, ok := err.(stackTracer); ok {
		return err
	}
	// 添加堆栈信息
	return pkgerrors.WithStack(err)
}
