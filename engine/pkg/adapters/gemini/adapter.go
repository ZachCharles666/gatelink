// Package gemini 实现 Google Gemini API 适配器。
// Gemini 使用自己的请求格式（contents 数组），与 OpenAI 有差异。
package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourname/gatelink-engine/pkg/adapters"
)

const (
	baseURL      = "https://generativelanguage.googleapis.com"
	chatEndpoint = "/v1beta/models/{model}:generateContent"
	apiVersion   = "v1beta"
)

var supportedModels = []string{
	"gemini-2.0-flash",
	"gemini-1.5-pro",
	"gemini-1.5-flash",
}

var pricingTable = map[string][2]float64{
	"gemini-2.0-flash": {0.10, 0.40},
	"gemini-1.5-pro":   {1.25, 5.00},
	"gemini-1.5-flash": {0.075, 0.30},
}

// Adapter Gemini 适配器
type Adapter struct {
	httpClient *http.Client
}

func New() *Adapter {
	return &Adapter{httpClient: &http.Client{Timeout: 60 * time.Second}}
}

func (a *Adapter) Vendor() vendor.Vendor       { return vendor.VendorGemini }
func (a *Adapter) SupportedModels() []string   { return supportedModels }
func (a *Adapter) BaseURL() string             { return baseURL }
func (a *Adapter) ChatEndpoint() string        { return chatEndpoint }

func (a *Adapter) Headers(apiKey string) map[string]string {
	return map[string]string{
		"x-goog-api-key": apiKey,
		"content-type":   "application/json",
	}
}

type geminiContent struct {
	Role  string `json:"role"`
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}

type geminiRequest struct {
	Contents         []geminiContent `json:"contents"`
	SystemInstruction *geminiContent `json:"system_instruction,omitempty"`
	GenerationConfig  *struct {
		MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
		Temperature     float64 `json:"temperature,omitempty"`
	} `json:"generationConfig,omitempty"`
}

func (a *Adapter) FormatRequest(req *vendor.ChatRequest) ([]byte, error) {
	gr := geminiRequest{}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			gr.SystemInstruction = &geminiContent{
				Role:  "user",
				Parts: []struct{ Text string `json:"text"` }{{Text: msg.Content}},
			}
			continue
		}
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		gr.Contents = append(gr.Contents, geminiContent{
			Role:  role,
			Parts: []struct{ Text string `json:"text"` }{{Text: msg.Content}},
		})
	}

	if req.System != "" && gr.SystemInstruction == nil {
		gr.SystemInstruction = &geminiContent{
			Role:  "user",
			Parts: []struct{ Text string `json:"text"` }{{Text: req.System}},
		}
	}

	if req.MaxTokens > 0 || req.Temperature > 0 {
		gr.GenerationConfig = &struct {
			MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
			Temperature     float64 `json:"temperature,omitempty"`
		}{
			MaxOutputTokens: req.MaxTokens,
			Temperature:     req.Temperature,
		}
	}

	return json.Marshal(gr)
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
}

func (a *Adapter) ParseResponse(body []byte) (*vendor.ChatResponse, error) {
	var gr geminiResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return nil, fmt.Errorf("parse gemini response: %w", err)
	}
	content := ""
	finishReason := ""
	if len(gr.Candidates) > 0 {
		for _, part := range gr.Candidates[0].Content.Parts {
			content += part.Text
		}
		finishReason = gr.Candidates[0].FinishReason
	}
	return &vendor.ChatResponse{
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Choices: []vendor.Choice{{
			Index:        0,
			Message:      vendor.Message{Role: "assistant", Content: content},
			FinishReason: finishReason,
		}},
		Usage: vendor.Usage{
			PromptTokens:     gr.UsageMetadata.PromptTokenCount,
			CompletionTokens: gr.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      gr.UsageMetadata.TotalTokenCount,
		},
	}, nil
}

func (a *Adapter) ParseUsage(body []byte) (*vendor.Usage, error) {
	var gr geminiResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return nil, fmt.Errorf("parse gemini usage: %w", err)
	}
	return &vendor.Usage{
		PromptTokens:     gr.UsageMetadata.PromptTokenCount,
		CompletionTokens: gr.UsageMetadata.CandidatesTokenCount,
		TotalTokens:      gr.UsageMetadata.TotalTokenCount,
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
