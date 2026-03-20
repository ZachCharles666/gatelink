package openai_test

import (
	"testing"

	"github.com/yourname/tokenglide-engine/pkg/adapters"
	"github.com/yourname/tokenglide-engine/pkg/adapters/openai"
)

func TestFormatRequest_StandardFormat(t *testing.T) {
	adapter := openai.New()
	req := &vendor.ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []vendor.Message{{Role: "user", Content: "Hello"}},
	}
	body, err := adapter.FormatRequest(req)
	if err != nil {
		t.Fatalf("FormatRequest: %v", err)
	}
	if len(body) == 0 {
		t.Error("body should not be empty")
	}
}

func TestParseResponse(t *testing.T) {
	adapter := openai.New()
	rawResp := `{
		"id": "chatcmpl-test",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4o-mini",
		"choices": [{"index":0,"message":{"role":"assistant","content":"Hi!"},"finish_reason":"stop"}],
		"usage": {"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8}
	}`
	resp, err := adapter.ParseResponse([]byte(rawResp))
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if resp.Choices[0].Message.Content != "Hi!" {
		t.Errorf("content mismatch: %s", resp.Choices[0].Message.Content)
	}
}

func TestCalcCost_Accuracy(t *testing.T) {
	adapter := openai.New()
	usage := &vendor.Usage{PromptTokens: 1_000_000, CompletionTokens: 1_000_000}
	cost, err := adapter.CalcCost(usage, "gpt-4o-mini")
	if err != nil {
		t.Fatalf("CalcCost: %v", err)
	}
	// gpt-4o-mini: $0.15 + $0.60 = $0.75
	if cost.CostUSD != 0.75 {
		t.Errorf("cost mismatch: got %f, want 0.75", cost.CostUSD)
	}
}

func TestImplementsAdapter(t *testing.T) {
	var _ vendor.Adapter = openai.New()
}
