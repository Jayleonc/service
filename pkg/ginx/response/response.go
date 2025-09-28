package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/pkg/xerr"
)

// Response 定义了 API 响应的通用结构。
type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// PageResult 标准分页响应结构
type PageResult[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}

// Success 使用默认 200 状态码返回成功响应。
func Success(c *gin.Context, data any) {
	SuccessWithStatus(c, http.StatusOK, data)
}

// SuccessWithStatus 使用指定的 HTTP 状态码返回成功响应。
func SuccessWithStatus(c *gin.Context, status int, data any) {
	c.JSON(status, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 使用自定义文案返回成功响应。
func SuccessWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error 根据传入错误类型返回统一的失败响应，并优先使用业务错误信息。
func Error(c *gin.Context, httpStatus int, err error) {
	var business *xerr.Error
	if errors.As(err, &business) {
		c.JSON(httpStatus, Response{Code: business.Code, Message: business.Message})
		return
	}

	message := http.StatusText(httpStatus)
	if err != nil && err.Error() != "" {
		message = err.Error()
	}

	c.JSON(httpStatus, Response{Code: httpStatus, Message: message})
}
