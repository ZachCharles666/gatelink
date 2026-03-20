package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 业务错误码
const (
	CodeOK              = 0
	CodeInvalidParam    = 1001
	CodeUnauthorized    = 1002
	CodeForbidden       = 1003
	CodeNotFound        = 1004
	CodeInsufficientBal = 1005
	CodeNoAvailAccount  = 4001
	CodeBuyerNoBalance  = 4002
	CodeAuditFailed     = 4003
	CodeRateLimited     = 4004
	CodeVendorError     = 5001
	CodeInternalError   = 5000
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: CodeOK, Msg: "ok", Data: data})
}

func Fail(c *gin.Context, httpStatus, code int, msg string) {
	c.JSON(httpStatus, Response{Code: code, Msg: msg})
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

func NoAvailAccount(c *gin.Context) {
	Fail(c, http.StatusServiceUnavailable, CodeNoAvailAccount, "no available account in pool")
}

func AuditFailed(c *gin.Context, reason string) {
	Fail(c, http.StatusBadRequest, CodeAuditFailed, "content audit failed: "+reason)
}
