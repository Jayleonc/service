package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Jayleonc/service/pkg/xerr"
)

// Response defines the standard structure shared by all API responses.
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

// Success writes a successful response with the provided data payload.
func Success(c *gin.Context, data any) {
	SuccessWithStatus(c, http.StatusOK, data)
}

// SuccessWithStatus writes a successful response with the supplied HTTP status code.
func SuccessWithStatus(c *gin.Context, status int, data any) {
	c.JSON(status, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage writes a successful response with a custom message.
func SuccessWithMessage(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error writes an error response leveraging typed business errors when available.
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
