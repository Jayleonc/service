package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response defines the standard structure shared by all API responses.
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageResult 标准分页响应结构
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// Success writes a successful response with the provided data payload.
func Success(c *gin.Context, data interface{}) {
	SuccessWithStatus(c, http.StatusOK, data)
}

// SuccessWithStatus writes a successful response with the supplied HTTP status code.
func SuccessWithStatus(c *gin.Context, status int, data interface{}) {
	c.JSON(status, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage writes a successful response with a custom message.
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error writes an error response with the supplied business code and message.
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}
