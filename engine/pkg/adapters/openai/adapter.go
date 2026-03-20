// Package openai 实现 OpenAI API 适配器
package openai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yourname/tokenglide-engine/pkg/adapters"
)

const (
	baseURL      = "https://api.openai.com"
	chatEndpoint = "/v1/chat/completions"
)

var supportedModels = []string{
	"gpt-4o",
	"gpt-4o-mini",
	"o1",
	"o1-mini",
}

var pricingTable = map[string][2]float64{
	"gpt-4o":      {2.50, 10.00},
	"gpt-4o-mini": {0.15, 0.60},
	"o1":          {15.00, 60.00},
	"o1-mini":     {3.00, 12.00},
}

// Adapter OpenAI 适配器
type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Vendor() vendor.Vendor     { return vendor.VendorOpenAI }
func (a *Adapter) SupportedModels() []string { return supportedModels }
func (a *Adapter) BaseURL() string           { return baseURL }
func (a *Adapter) ChatEndpoint() string      { return chatEndpoint }

func (a *Adapter) Headers(apiKey string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + apiKey,
		"Content-Type":  "application/json",
	}
}

func (a *Adapter) FormatRequest(req *vendor.ChatRequest) ([]byte, error) {
	type openAIRequest struct {
		Model       string          `json:"model"`
		Messages    []vendor.Message `json:"messages"`
		Stream      bool            `json:"stream,omitempty"`
		MaxTokens   int             `json:"max_tokens,omitempty"`
		Temperature float64         `json:"temperature,omitempty"`
	}
	r := openAIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Stream:      req.Stream,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
	}
	return json.Marshal(r)
}

type openAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (a *Adapter) ParseResponse(body []byte) (*vendor.ChatResponse, error) {
	var or openAIResponse
	if err := json.Unmarshal(body, &or); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}

	choices := make([]vendor.Choice, len(or.Choices))
	for i, c := range or.Choices {
		choices[i] = vendor.Choice{
			Index:        c.Index,
			Message:      vendor.Message{Role: c.Message.Role, Content: c.Message.Content},
			FinishReason: c.FinishReason,
		}
	}

	return &vendor.ChatResponse{
		ID:      or.ID,
		Object:  or.Object,
		Created: or.Created,
		Model:   or.Model,
		Choices: choices,
		Usage: vendor.Usage{
			PromptTokens:     or.Usage.PromptTokens,
			CompletionTokens: or.Usage.CompletionTokens,
			TotalTokens:      or.Usage.TotalTokens,
		},
	}, nil
}

func (a *Adapter) ParseUsage(body []byte) (*vendor.Usage, error) {
	var or openAIResponse
	if err := json.Unmarshal(body, &or); err != nil {
		return nil, fmt.Errorf("parse usage: %w", err)
	}
	return &vendor.Usage{
		PromptTokens:     or.Usage.PromptTokens,
		CompletionTokens: or.Usage.CompletionTokens,
		TotalTokens:      or.Usage.TotalTokens,
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
	if len(apiKey) < 20 || apiKey[:3] != "sk-" {
		return &vendor.ValidationResult{
			Valid:    false,
			ErrorMsg: "invalid key format",
		}, nil
	}
	return &vendor.ValidationResult{Valid: true}, nil
}
