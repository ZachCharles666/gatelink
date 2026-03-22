package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	engineclient "github.com/ZachCharles666/gatelink/api/internal/engine"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type streamUsage struct {
	PromptTokens     int
	CompletionTokens int
}

func (h *Handler) handleStream(c *gin.Context, buyerID, vendor string, req chatRequest) {
	streamResp, err := h.engine.DispatchStream(c.Request.Context(), engineclient.DispatchRequest{
		BuyerID:     buyerID,
		Vendor:      vendor,
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      true,
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
	defer streamResp.Body.Close()

	contentType := streamResp.ContentType
	if contentType == "" {
		contentType = "text/event-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	scanner := bufio.NewScanner(streamResp.Body)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	var (
		lastUsage streamUsage
		accountID string
	)

	for scanner.Scan() {
		line := scanner.Text()
		c.Writer.WriteString(line + "\n")
		c.Writer.Flush()

		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				break
			}
			if usage, id, ok := extractStreamMetadata([]byte(payload)); ok {
				lastUsage = usage
				if id != "" {
					accountID = id
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Warn().Err(err).Str("buyer_id", buyerID).Str("model", req.Model).Msg("stream scanner stopped with error")
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.accounting.ChargeAfterStream(ctx, buyerID, accountID, vendor, req.Model, lastUsage.PromptTokens, lastUsage.CompletionTokens); err != nil {
			log.Warn().
				Err(err).
				Str("buyer_id", buyerID).
				Str("account_id", accountID).
				Str("model", req.Model).
				Msg("charge after stream failed")
		}
	}()
}

func extractStreamMetadata(data []byte) (streamUsage, string, bool) {
	var openAIChunk struct {
		AccountID string `json:"account_id"`
		Usage     *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &openAIChunk); err == nil && openAIChunk.Usage != nil {
		return streamUsage{
			PromptTokens:     openAIChunk.Usage.PromptTokens,
			CompletionTokens: openAIChunk.Usage.CompletionTokens,
		}, openAIChunk.AccountID, true
	}

	var anthropicChunk struct {
		AccountID string `json:"account_id"`
		Usage     *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &anthropicChunk); err == nil && anthropicChunk.Usage != nil {
		return streamUsage{
			PromptTokens:     anthropicChunk.Usage.InputTokens,
			CompletionTokens: anthropicChunk.Usage.OutputTokens,
		}, anthropicChunk.AccountID, true
	}

	var accountChunk struct {
		AccountID string `json:"account_id"`
	}
	if err := json.Unmarshal(data, &accountChunk); err == nil && accountChunk.AccountID != "" {
		return streamUsage{}, accountChunk.AccountID, true
	}

	return streamUsage{}, "", false
}
