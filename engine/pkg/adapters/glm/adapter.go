// Package glm 实现智谱 AI GLM API 适配器。
// GLM 兼容 OpenAI 格式，鉴权使用 Bearer token。
package glm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourname/gatelink-engine/pkg/adapters"
)

const (
	baseURL      = "https://open.bigmodel.cn/api/paas"
	chatEndpoint = "/v4/chat/completions"
)

var supportedModels = []string{
	"glm-4",
	"glm-4-flash",
	"glm-4-air",
	"glm-3-turbo",
}

var pricingTable = map[string][2]float64{
	"glm-4":       {0.14, 0.14},
	"glm-4-flash": {0.014, 0.014},
	"glm-4-air":   {0.028, 0.028},
	"glm-3-turbo": {0.007, 0.007},
}

// Adapter 智谱 GLM 适配器（OpenAI 兼容格式）
type Adapter struct {
	httpClient *http.Client
}

func New() *Adapter {
	return &Adapter{httpClient: &http.Client{Timeout: 60 * time.Second}}
}

func (a *Adapter) Vendor() vendor.Vendor       { return vendor.VendorGLM }
func (a *Adapter) SupportedModels() []string   { return supportedModels }
func (a *Adapter) BaseURL() string             { return baseURL }
func (a *Adapter) ChatEndpoint() string        { return chatEndpoint }

func (a *Adapter) Headers(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}
}

func (a *Adapter) FormatRequest(req *vendor.ChatRequest) ([]byte, error) {
	type glmReq struct {
		Model       string           `json:"model"`
		Messages    []vendor.Message `json:"messages"`
		Stream      bool             `json:"stream,omitempty"`
		MaxTokens   int              `json:"max_tokens,omitempty"`
		Temperature float64          `json:"temperature,omitempty"`
	}
	return json.Marshal(glmReq{
		Model: req.Model, Messages: req.Messages,
		Stream: req.Stream, MaxTokens: req.MaxTokens, Temperature: req.Temperature,
	})
}

type glmResponse struct {
	ID      string `json:"id"`
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
	var r glmResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parse glm response: %w", err)
	}
	resp := &vendor.ChatResponse{
		ID: r.ID, Object: "chat.completion", Created: r.Created, Model: r.Model,
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
	var r glmResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("parse glm usage: %w", err)
	}
	return &vendor.Usage{
		PromptTokens: r.Usage.PromptTokens, CompletionTokens: r.Usage.CompletionTokens,
		TotalTokens: r.Usage.TotalTokens,
	}, nil
}

func (a *Adapter) CalcCost(usage *vendor.Usage, model string) (*vendor.Cost, error) {
	prices, ok := pricingTable[model]
	if !ok {
		return nil, fmt.Errorf("unknown model: %s", model)
	}
	// GLM 输入输出同价
	total := float64(usage.TotalTokens) / 1_000_000 * prices[0]
	return &vendor.Cost{
		InputTokens: usage.PromptTokens, OutputTokens: usage.CompletionTokens,
		CostUSD: total,
	}, nil
}

func (a *Adapter) ValidateKey(ctx context.Context, apiKey string) (*vendor.ValidationResult, error) {
	if len(apiKey) < 10 {
		return &vendor.ValidationResult{Valid: false, ErrorMsg: "invalid key format"}, nil
	}
	return &vendor.ValidationResult{Valid: true}, nil
}
