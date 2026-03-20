package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/yourname/gatelink-engine/internal/scheduler"
	"github.com/yourname/gatelink-engine/pkg/adapters"
)

// StreamResult 流式转发结果（完成后可用）
type StreamResult struct {
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	VendorReqID  string
	Partial      bool // true = 流中途断开，usage 为估算值
}

// ForwardStream 执行流式转发，实时将 SSE chunk 透传给客户端。
//
// 流程：
//  1. 解密 API Key（只存在于 goroutine 栈）
//  2. 向厂商发起 SSE 请求
//  3. 逐行读取 data: ... 行，实时写入 gin.ResponseWriter
//  4. 流结束时从最后一个 chunk 提取 usage，写入 usage_records
//  5. 若客户端断开（ctx.Done），记录 partial usage
func (f *Forwarder) ForwardStream(
	c *gin.Context,
	acct *scheduler.AccountInfo,
	req *vendor.ChatRequest,
	buyerID string,
	buyerChargeRate float64,
) (*StreamResult, error) {
	adapter, ok := f.registry.Get(vendor.Vendor(acct.Vendor))
	if !ok {
		return nil, fmt.Errorf("no adapter for vendor=%s", acct.Vendor)
	}

	// 在栈上解密
	apiKey, err := f.keystore.Decrypt(acct.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt key: %w", err)
	}

	// 强制 stream=true
	req.Stream = true
	body, err := adapter.FormatRequest(req)
	if err != nil {
		return nil, fmt.Errorf("format request: %w", err)
	}

	url := adapter.BaseURL() + adapter.ChatEndpoint()
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	for k, v := range adapter.Headers(apiKey) {
		httpReq.Header.Set(k, v)
	}

	// 清理 apiKey（尽早）
	apiKey = strings.Repeat("x", len(apiKey))
	_ = apiKey

	start := time.Now()
	resp, err := f.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("vendor returned %d: %s", resp.StatusCode, string(respBody))
	}

	vendorReqID := resp.Header.Get("X-Request-Id")

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	result := &StreamResult{VendorReqID: vendorReqID}
	var lastDataChunk []byte

	scanner := bufio.NewScanner(resp.Body)
	clientCtx := c.Request.Context()
	partial := false

	for scanner.Scan() {
		// 检查客户端是否断开
		select {
		case <-clientCtx.Done():
			partial = true
			goto done
		default:
		}

		line := scanner.Text()
		if line == "" {
			// 空行：SSE 事件分隔符，透传
			fmt.Fprintf(c.Writer, "\n")
			c.Writer.Flush()
			continue
		}

		// 透传原始行到客户端
		fmt.Fprintf(c.Writer, "%s\n", line)
		c.Writer.Flush()

		// 保存最后一个 data: 行（用于提取 usage）
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload != "[DONE]" {
				lastDataChunk = []byte(payload)
			}
		}
	}

done:
	latency := time.Since(start)
	result.Partial = partial

	// 从最后一个 chunk 提取 usage
	if len(lastDataChunk) > 0 {
		if usage, err := extractStreamUsage(lastDataChunk); err == nil {
			result.InputTokens = usage.PromptTokens
			result.OutputTokens = usage.CompletionTokens

			cost, _ := adapter.CalcCost(usage, req.Model)
			if cost != nil {
				result.CostUSD = cost.CostUSD
			}
		}
	}

	log.Info().
		Str("account", acct.ID).
		Str("vendor", acct.Vendor).
		Dur("latency", latency).
		Bool("partial", partial).
		Int("input_tokens", result.InputTokens).
		Int("output_tokens", result.OutputTokens).
		Msg("stream completed")

	// 写入 usage_records（即使是 partial 也记录）
	if result.InputTokens > 0 || result.OutputTokens > 0 {
		usage := &vendor.Usage{
			PromptTokens:     result.InputTokens,
			CompletionTokens: result.OutputTokens,
			TotalTokens:      result.InputTokens + result.OutputTokens,
		}
		cost := &vendor.Cost{
			InputTokens:  result.InputTokens,
			OutputTokens: result.OutputTokens,
			CostUSD:      result.CostUSD,
		}
		if err := f.writeUsageRecord(context.Background(), acct, buyerID, req.Model, usage, cost, vendorReqID, buyerChargeRate); err != nil {
			log.Error().Err(err).Str("account", acct.ID).Msg("write stream usage record failed")
		}
		f.engine.RecordConsumed(context.Background(), acct.ID, acct.Vendor, result.CostUSD)
		f.engine.RecordBuyerHistory(context.Background(), buyerID, acct.ID)
	}

	return result, nil
}

// extractStreamUsage 从 SSE chunk JSON 中提取 usage 信息
// 兼容 OpenAI 格式（stream_options.include_usage=true 的最后一个 chunk）
// 和 Anthropic message_delta 事件
func extractStreamUsage(data []byte) (*vendor.Usage, error) {
	// OpenAI 格式：{"choices":[],"usage":{"prompt_tokens":N,"completion_tokens":N}}
	var openaiChunk struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &openaiChunk); err == nil && openaiChunk.Usage != nil {
		return &vendor.Usage{
			PromptTokens:     openaiChunk.Usage.PromptTokens,
			CompletionTokens: openaiChunk.Usage.CompletionTokens,
			TotalTokens:      openaiChunk.Usage.PromptTokens + openaiChunk.Usage.CompletionTokens,
		}, nil
	}

	// Anthropic 格式：{"type":"message_delta","usage":{"output_tokens":N}}
	// Anthropic 的 input_tokens 在 message_start 事件中
	var anthropicChunk struct {
		Type  string `json:"type"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(data, &anthropicChunk); err == nil && anthropicChunk.Usage != nil {
		return &vendor.Usage{
			PromptTokens:     anthropicChunk.Usage.InputTokens,
			CompletionTokens: anthropicChunk.Usage.OutputTokens,
			TotalTokens:      anthropicChunk.Usage.InputTokens + anthropicChunk.Usage.OutputTokens,
		}, nil
	}

	return nil, fmt.Errorf("no usage in chunk")
}
