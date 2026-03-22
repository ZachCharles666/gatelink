package proxy

import (
	"context"
	"net/http"
	"strings"

	"github.com/ZachCharles666/gatelink/api/internal/accounting"
	engineclient "github.com/ZachCharles666/gatelink/api/internal/engine"
	"github.com/ZachCharles666/gatelink/api/internal/response"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type engineAPI interface {
	Audit(ctx context.Context, req engineclient.AuditRequest) (*engineclient.AuditResult, error)
	Dispatch(ctx context.Context, req engineclient.DispatchRequest) (*engineclient.DispatchResult, error)
	DispatchStream(ctx context.Context, req engineclient.DispatchRequest) (*engineclient.StreamResponse, error)
}

type accountingAPI interface {
	ListModels(ctx context.Context) ([]accounting.ModelInfo, error)
	ChargeAfterDispatch(ctx context.Context, buyerID string, result *engineclient.DispatchResult) error
	ChargeAfterStream(ctx context.Context, buyerID, accountID, vendor, model string, inputTokens, outputTokens int) error
}

type Handler struct {
	engine     engineAPI
	accounting accountingAPI
}

type chatRequest struct {
	Model       string                 `json:"model" binding:"required"`
	Messages    []engineclient.Message `json:"messages" binding:"required,min=1"`
	Stream      bool                   `json:"stream"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature"`
}

func NewHandler(eng engineAPI, acct accountingAPI) *Handler {
	return &Handler{engine: eng, accounting: acct}
}

func (h *Handler) ChatCompletions(c *gin.Context) {
	buyerID := c.GetString("buyer_id")
	balanceUSD := c.GetFloat64("buyer_balance")

	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if balanceUSD <= 0 {
		response.InsufficientBalance(c)
		return
	}

	messages := make([]string, 0, len(req.Messages))
	for _, message := range req.Messages {
		if message.Content != "" {
			messages = append(messages, message.Content)
		}
	}

	auditResult, err := h.engine.Audit(c.Request.Context(), engineclient.AuditRequest{
		Messages: messages,
		BuyerID:  buyerID,
	})
	if err != nil {
		log.Warn().Err(err).Str("buyer_id", buyerID).Msg("engine audit failed open")
	} else if auditResult != nil && !auditResult.Safe {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "request blocked by content policy",
				"type":    "content_policy_violation",
			},
		})
		return
	}

	if req.Stream {
		h.handleStream(c, buyerID, inferVendor(req.Model), req)
		return
	}

	dispatchResult, err := h.engine.Dispatch(c.Request.Context(), engineclient.DispatchRequest{
		BuyerID:     buyerID,
		Vendor:      inferVendor(req.Model),
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      false,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	})
	if err != nil {
		switch {
		case engineclient.IsNoAccount(err):
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{"message": "no available account", "type": "service_unavailable"},
			})
		case engineclient.IsAuditFail(err):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"message": "content policy violation", "type": "content_policy_violation"},
			})
		case engineclient.IsVendorRateLimited(err):
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"message": "upstream rate limited", "type": "rate_limit_exceeded"},
			})
		default:
			c.JSON(http.StatusBadGateway, gin.H{
				"error": gin.H{"message": "upstream error", "type": "upstream_error"},
			})
		}
		return
	}

	if err := h.accounting.ChargeAfterDispatch(c.Request.Context(), buyerID, dispatchResult); err != nil {
		log.Warn().Err(err).Str("buyer_id", buyerID).Str("account_id", dispatchResult.AccountID).Msg("charge after dispatch failed")
	}

	c.Data(http.StatusOK, "application/json", dispatchResult.Response)
}

func (h *Handler) ListModels(c *gin.Context) {
	models, err := h.accounting.ListModels(c.Request.Context())
	if err != nil {
		response.InternalError(c)
		return
	}

	data := make([]gin.H, 0, len(models))
	for _, model := range models {
		data = append(data, gin.H{
			"id":       model.Model,
			"object":   "model",
			"owned_by": model.Vendor,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func inferVendor(model string) string {
	switch {
	case strings.HasPrefix(model, "claude-"):
		return "anthropic"
	case strings.HasPrefix(model, "gpt-"), strings.HasPrefix(model, "o1"):
		return "openai"
	case strings.HasPrefix(model, "gemini-"):
		return "gemini"
	default:
		return "anthropic"
	}
}
