package api

import (
	"github.com/gin-gonic/gin"
	"github.com/yourname/gatelink-engine/internal/audit"
)

// AuditHandler 处理 POST /internal/v1/audit
// Dev-B 在调用 dispatch 前先调用此接口进行内容审核
type AuditHandler struct {
	classifier *audit.Classifier
}

func NewAuditHandler(classifier *audit.Classifier) *AuditHandler {
	return &AuditHandler{classifier: classifier}
}

// AuditRequest 审核请求
type AuditRequest struct {
	Messages []string `json:"messages" binding:"required"` // 消息内容列表
	BuyerID  string   `json:"buyer_id"`
}

// AuditResponse 审核响应
type AuditResponse struct {
	Safe    bool     `json:"safe"`
	Level   int      `json:"level"`   // 0=safe, 1=low, 2=medium, 3=high, 4=critical
	Reason  string   `json:"reason,omitempty"`
	Details []string `json:"details,omitempty"`
}

// Handle 处理内容审核请求
func (h *AuditHandler) Handle(c *gin.Context) {
	var req AuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	result := h.classifier.Classify(req.Messages)

	resp := AuditResponse{
		Safe:    result.IsSafe(),
		Level:   int(result.Level),
		Reason:  result.Reason,
		Details: result.Details,
	}

	if result.ShouldBlock() {
		AuditFailed(c, result.Reason)
		return
	}

	OK(c, resp)
}
