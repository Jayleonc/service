package xerr

import "fmt"

// Error 表示包含业务错误码与消息的结构化错误。
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error 实现 error 接口。
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Message
}

// New 构造一个带错误码的业务错误。
func New(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// WithMessage 克隆当前错误并覆盖消息内容。
func (e *Error) WithMessage(message string) *Error {
	if e == nil {
		return nil
	}
	clone := *e
	clone.Message = message
	return &clone
}

// Format 支持对错误进行格式化输出。
func (e *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		fmt.Fprint(s, e.Error())
	case 'q':
		fmt.Fprintf(s, "%q", e.Error())
	}
}
