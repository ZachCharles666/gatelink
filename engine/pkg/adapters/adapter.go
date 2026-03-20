// Package vendor 定义厂商适配器接口和公共类型。
// 每个厂商实现 Adapter 接口，调度引擎和代理层通过接口调用，无需感知厂商差异。
package vendor

import (
	"context"
	"time"
)

// Vendor 厂商标识
type Vendor string

const (
	VendorAnthropic Vendor = "anthropic"
	VendorOpenAI    Vendor = "openai"
	VendorGemini    Vendor = "gemini"
	VendorQwen      Vendor = "qwen"
	VendorGLM       Vendor = "glm"
	VendorKimi      Vendor = "kimi"
)

// SupportedVendors 所有支持的厂商
var SupportedVendors = []Vendor{
	VendorAnthropic, VendorOpenAI, VendorGemini,
	VendorQwen, VendorGLM, VendorKimi,
}

// ChatRequest 标准化聊天请求（OpenAI 格式）
type ChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []Message              `json:"messages"`
	Stream      bool                   `json:"stream"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	System      string                 `json:"system,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

// Message 对话消息
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse 标准化响应（OpenAI 格式）
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 响应选项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage token 用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Cost 费用计算结果
type Cost struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
}

// ValidationResult API Key 验证结果
type ValidationResult struct {
	Valid      bool      `json:"valid"`
	BalanceUSD float64   `json:"balance_usd,omitempty"`
	ExpireAt   time.Time `json:"expire_at,omitempty"`
	ErrorMsg   string    `json:"error_msg,omitempty"`
}

// Adapter 厂商适配器接口
type Adapter interface {
	Vendor() Vendor
	SupportedModels() []string
	ValidateKey(ctx context.Context, apiKey string) (*ValidationResult, error)
	FormatRequest(req *ChatRequest) ([]byte, error)
	ParseResponse(body []byte) (*ChatResponse, error)
	ParseUsage(body []byte) (*Usage, error)
	CalcCost(usage *Usage, model string) (*Cost, error)
	BaseURL() string
	Headers(apiKey string) map[string]string
	ChatEndpoint() string
}

// Registry 适配器注册表
type Registry struct {
	adapters map[Vendor]Adapter
}

func NewRegistry() *Registry {
	return &Registry{adapters: make(map[Vendor]Adapter)}
}

func (r *Registry) Register(a Adapter) {
	r.adapters[a.Vendor()] = a
}

func (r *Registry) Get(vendor Vendor) (Adapter, bool) {
	a, ok := r.adapters[vendor]
	return a, ok
}

func (r *Registry) All() []Adapter {
	var result []Adapter
	for _, a := range r.adapters {
		result = append(result, a)
	}
	return result
}
