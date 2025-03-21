package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	StatusCode int         `json:"status_code"`
	Error      string      `json:"error"`
	Data       interface{} `json:"data,omitempty"`
}

func NewResponse(statusCode int, error string, data interface{}) *Response {
	return &Response{
		StatusCode: statusCode,
		Error:      error,
		Data:       data,
	}
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, NewResponse(http.StatusOK, "success", data))
}

func Error(c *gin.Context, statusCode int, error string) {
	c.JSON(statusCode, NewResponse(statusCode, error, nil))
}

func ErrorWithData(c *gin.Context, statusCode int, error string, data interface{}) {
	c.JSON(statusCode, NewResponse(statusCode, error, data))
}
