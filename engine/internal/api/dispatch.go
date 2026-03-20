package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourname/gatelink-engine/internal/proxy"
	"github.com/yourname/gatelink-engine/internal/scheduler"
	"github.com/yourname/gatelink-engine/pkg/adapters"
)

// DispatchHandler 处理 POST /internal/v1/dispatch
type DispatchHandler struct {
	engine    *scheduler.Engine
	forwarder *proxy.Forwarder
}

func NewDispatchHandler(engine *scheduler.Engine, forwarder *proxy.Forwarder) *DispatchHandler {
	return &DispatchHandler{engine: engine, forwarder: forwarder}
}

// DispatchRequest 请求体
type DispatchRequest struct {
	BuyerID         string           `json:"buyer_id" binding:"required"`
	Vendor          string           `json:"vendor" binding:"required"`
	Model           string           `json:"model" binding:"required"`
	Messages        []vendor.Message `json:"messages" binding:"required"`
	Stream          bool             `json:"stream"`
	MaxTokens       int              `json:"max_tokens"`
	Temperature     float64          `json:"temperature"`
	BuyerChargeRate float64          `json:"buyer_charge_rate"`
}

// Handle 处理调度请求（非流式 + 流式）
func (h *DispatchHandler) Handle(c *gin.Context) {
	var req DispatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()

	// 1. 调度账号
	dispResult, err := h.engine.Dispatch(ctx, &scheduler.DispatchRequest{
		BuyerID: req.BuyerID,
		Vendor:  req.Vendor,
		Model:   req.Model,
	})
	if err != nil {
		NoAvailAccount(c)
		return
	}

	chatReq := &vendor.ChatRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      req.Stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}

	// 2. 流式分支
	if req.Stream {
		_, err := h.forwarder.ForwardStream(c, dispResult.Account, chatReq, req.BuyerID, req.BuyerChargeRate)
		if err != nil {
			Fail(c, http.StatusBadGateway, CodeVendorError, "stream failed: "+err.Error())
		}
		// 流式响应已直接写入 ResponseWriter，无需额外 OK()
		return
	}

	// 3. 非流式分支
	result, err := h.forwarder.Forward(ctx, dispResult.Account, chatReq, req.BuyerID, req.BuyerChargeRate)
	if err != nil {
		if result != nil && result.StatusCode == http.StatusTooManyRequests {
			Fail(c, http.StatusServiceUnavailable, CodeRateLimited, "vendor rate limited")
			return
		}
		Fail(c, http.StatusBadGateway, CodeVendorError, "vendor request failed: "+err.Error())
		return
	}

	registry := c.MustGet("registry").(*vendor.Registry)
	adapter, ok := registry.Get(vendor.Vendor(dispResult.Account.Vendor))
	if !ok {
		InternalError(c)
		return
	}

	chatResp, err := adapter.ParseResponse(result.Body)
	if err != nil {
		InternalError(c)
		return
	}

	OK(c, gin.H{
		"response":      chatResp,
		"account_id":    dispResult.Account.ID,
		"vendor":        dispResult.Account.Vendor,
		"cost_usd":      result.CostUSD,
		"input_tokens":  result.InputTokens,
		"output_tokens": result.OutputTokens,
	})
}
