// Package anthropic 实现 Anthropic Claude API 适配器
package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourname/gatelink-engine/pkg/adapters"
)

const (
	baseURL      = "https://api.anthropic.com"
	chatEndpoint = "/v1/messages"
	apiVersion   = "2023-06-01"
)

var supportedModels = []string{
	"claude-opus-4-6",
	"claude-sonnet-4-6",
	"claude-haiku-4-5",
}

var pricingTable = map[string][2]float64{
	"claude-opus-4-6":   {15.00, 75.00},
	"claude-sonnet-4-6": {3.00, 15.00},
	"claude-haiku-4-5":  {0.80, 4.00},
}

// Adapter Anthropic 适配器
type Adapter struct {
	httpClient *http.Client
}

func New() *Adapter {
	return &Adapter{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (a *Adapter) Vendor() vendor.Vendor  { return vendor.VendorAnthropic }
func (a *Adapter) SupportedModels() []string { return supportedModels }
func (a *Adapter) BaseURL() string           { return baseURL }
func (a *Adapter) ChatEndpoint() string      { return chatEndpoint }

func (a *Adapter) Headers(apiKey string) map[string]string {
	return map[string]string{
		"x-api-key":         apiKey,
		"anthropic-version": apiVersion,
		"content-type":      "application/json",
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// FormatRequest 将标准请求转换为 Anthropic 格式
// 主要差异：system 消息需要单独提取到 system 字段
func (a *Adapter) FormatRequest(req *vendor.ChatRequest) ([]byte, error) {
	anthropicReq := anthropicRequest{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}

	if anthropicReq.MaxTokens == 0 {
		anthropicReq.MaxTokens = 4096
	}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			anthropicReq.System = msg.Content
		} else {
			anthropicReq.Messages = append(anthropicReq.Messages, anthropicMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	if req.System != "" {
		anthropicReq.System = req.System
	}

	return json.Marshal(anthropicReq)
}

type anthropicResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Role string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *Adapter) ParseResponse(body []byte) (*vendor.ChatResponse, error) {
	var ar anthropicResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, fmt.Errorf("parse anthropic response: %w", err)
	}

	content := ""
	for _, c := range ar.Content {
		if c.Type == "text" {
			content += c.Text
		}
	}

	return &vendor.ChatResponse{
		ID:      ar.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   ar.Model,
		Choices: []vendor.Choice{
			{
				Index:        0,
				Message:      vendor.Message{Role: "assistant", Content: content},
				FinishReason: ar.StopReason,
			},
		},
		Usage: vendor.Usage{
			PromptTokens:     ar.Usage.InputTokens,
			CompletionTokens: ar.Usage.OutputTokens,
			TotalTokens:      ar.Usage.InputTokens + ar.Usage.OutputTokens,
		},
	}, nil
}

func (a *Adapter) ParseUsage(body []byte) (*vendor.Usage, error) {
	var ar anthropicResponse
	if err := json.Unmarshal(body, &ar); err != nil {
		return nil, fmt.Errorf("parse usage: %w", err)
	}
	return &vendor.Usage{
		PromptTokens:     ar.Usage.InputTokens,
		CompletionTokens: ar.Usage.OutputTokens,
		TotalTokens:      ar.Usage.InputTokens + ar.Usage.OutputTokens,
	}, nil
}

func (a *Adapter) CalcCost(usage *vendor.Usage, model string) (*vendor.Cost, error) {
	prices, ok := pricingTable[model]
	if !ok {
		return nil, fmt.Errorf("unknown model: %s", model)
	}
	inputCost := float64(usage.PromptTokens) / 1_000_000 * prices[0]
	outputCost := float64(usage.CompletionTokens) / 1_000_000 * prices[1]
	return &vendor.Cost{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		CostUSD:      inputCost + outputCost,
	}, nil
}

func (a *Adapter) ValidateKey(ctx context.Context, apiKey string) (*vendor.ValidationResult, error) {
	if len(apiKey) < 20 || apiKey[:7] != "sk-ant-" {
		return &vendor.ValidationResult{
			Valid:    false,
			ErrorMsg: "invalid key format",
		}, nil
	}
	return &vendor.ValidationResult{Valid: true}, nil
}
