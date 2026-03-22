package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	CodeOK              = 0
	CodeInvalidParam    = 1001
	CodeUnauthorized    = 1002
	CodeForbidden       = 1003
	CodeNotFound        = 1004
	CodeInsufficientBal = 1005
	CodeInternalError   = 5000
)

type R struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, R{Code: CodeOK, Msg: "ok", Data: data})
}

func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, R{Code: code, Msg: msg})
}

func BadRequest(c *gin.Context, msg string) {
	Fail(c, http.StatusBadRequest, CodeInvalidParam, msg)
}

func Unauthorized(c *gin.Context) {
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, "unauthorized")
}

func NotFound(c *gin.Context, msg string) {
	Fail(c, http.StatusNotFound, CodeNotFound, msg)
}

func InternalError(c *gin.Context) {
	Fail(c, http.StatusInternalServerError, CodeInternalError, "internal server error")
}

func InsufficientBalance(c *gin.Context) {
	Fail(c, http.StatusPaymentRequired, CodeInsufficientBal, "insufficient balance")
}
