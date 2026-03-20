// Package qwen 实现阿里云通义千问 API 适配器。
// Qwen 兼容 OpenAI 格式，使用不同的 base URL 和 bearer token 鉴权。
package qwen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourname/gatelink-engine/pkg/adapters"
)

const (
	baseURL      = "https://dashscope.aliyuncs.com/compatible-mode"
	chatEndpoint = "/v1/chat/completions"
)

var supportedModels = []string{
	"qwen-max",
	"qwen-plus",
	"qwen-turbo",
	"qwen-long",
}

var pricingTable = map[string][2]float64{
	"qwen-max":   {0.40, 1.20},  // 每百万 token（美元估算）
	"qwen-plus":  {0.08, 0.24},
	"qwen-turbo": {0.02, 0.06},
	"qwen-long":  {0.05, 0.14},
}

// Adapter 通义千问适配器（OpenAI 兼容格式）
type Adapter struct {
	httpClient *http.Client
}

func New() *Adapter {
	return &Adapter{httpClient: &http.Client{Timeout: 60 * time.Second}}
}

func (a *Adapter) Vendor() vendor.Vendor       { return vendor.VendorQwen }
func (a *Adapter) SupportedModels() []string   { return supportedModels }
func (a *Adapter) BaseURL() string             { return baseURL }
func (a *Adapter) ChatEndpoint() string        { return chatEndpoint }

func (a *Adapter) Headers(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}
}

// FormatRequest Qwen 兼容 OpenAI 格式，直接透传
func (a *Adapter) FormatRequest(req *vendor.ChatRequest) ([]byte, error) {
	type openAIReq struct {
		Model       string           `json:"model"`
		Messages    []vendor.Message `json:"messages"`
		Stream      bool             `json:"stream,omitempty"`
		MaxTokens   int              `json:"max_tokens,omitempty"`
		Temperature float64          `json:"temperature,omitempty"`
	}
	return json.Marshal(openAIReq{
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      req.Stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	})
}

type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int            `json:"index"`
		Message      vendor.Message `json:"message"`
		FinishReason string         `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (a *Adapter) ParseResponse(body []byte) (*vendor.ChatResponse, error) {
	var r openAIResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parse qwen response: %w", err)
	}
	resp := &vendor.ChatResponse{
		ID: r.ID, Object: r.Object, Created: r.Created, Model: r.Model,
		Usage: vendor.Usage{
			PromptTokens: r.Usage.PromptTokens, CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens: r.Usage.TotalTokens,
		},
	}
	for _, c := range r.Choices {
		resp.Choices = append(resp.Choices, vendor.Choice{
			Index: c.Index, Message: c.Message, FinishReason: c.FinishReason,
		})
	}
	return resp, nil
}

func (a *Adapter) ParseUsage(body []byte) (*vendor.Usage, error) {
	var r openAIResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parse qwen usage: %w", err)
	}
	return &vendor.Usage{
		PromptTokens:     r.Usage.PromptTokens,
		CompletionTokens: r.Usage.CompletionTokens,
		TotalTokens:      r.Usage.TotalTokens,
	}, nil
}

func (a *Adapter) CalcCost(usage *vendor.Usage, model string) (*vendor.Cost, error) {
	prices, ok := pricingTable[model]
	if !ok {
		return nil, fmt.Errorf("unknown model: %s", model)
	}
	input := float64(usage.PromptTokens) / 1_000_000 * prices[0]
	output := float64(usage.CompletionTokens) / 1_000_000 * prices[1]
	return &vendor.Cost{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		CostUSD:      input + output,
	}, nil
}

func (a *Adapter) ValidateKey(ctx context.Context, apiKey string) (*vendor.ValidationResult, error) {
	if len(apiKey) < 10 {
		return &vendor.ValidationResult{Valid: false, ErrorMsg: "invalid key format"}, nil
	}
	return &vendor.ValidationResult{Valid: true}, nil
}
