package anthropic_test

import (
	"encoding/json"
	"testing"

	"github.com/yourname/tokenglide-engine/pkg/adapters"
	"github.com/yourname/tokenglide-engine/pkg/adapters/anthropic"
)

func TestFormatRequest_ExtractsSystemMessage(t *testing.T) {
	adapter := anthropic.New()

	req := &vendor.ChatRequest{
		Model: "claude-haiku-4-5",
		Messages: []vendor.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello"},
		},
	}

	body, err := adapter.FormatRequest(req)
	if err != nil {
		t.Fatalf("FormatRequest: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["system"] != "You are a helpful assistant." {
		t.Errorf("system field not extracted correctly, got: %v", result["system"])
	}

	messages, ok := result["messages"].([]interface{})
	if !ok || len(messages) != 1 {
		t.Errorf("expected 1 message (user only), got: %v", result["messages"])
	}
}

func TestFormatRequest_DefaultMaxTokens(t *testing.T) {
	adapter := anthropic.New()

	req := &vendor.ChatRequest{
		Model:    "claude-haiku-4-5",
		Messages: []vendor.Message{{Role: "user", Content: "hi"}},
	}

	body, _ := adapter.FormatRequest(req)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["max_tokens"].(float64) == 0 {
		t.Error("max_tokens should have a default value")
	}
}

func TestParseResponse(t *testing.T) {
	adapter := anthropic.New()

	rawResp := `{
		"id": "msg_test123",
		"type": "message",
		"role": "assistant",
		"content": [{"type": "text", "text": "Hello there!"}],
		"model": "claude-haiku-4-5",
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`

	resp, err := adapter.ParseResponse([]byte(rawResp))
	if err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}

	if resp.ID != "msg_test123" {
		t.Errorf("ID mismatch: got %s", resp.ID)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello there!" {
		t.Errorf("content mismatch: got %s", resp.Choices[0].Message.Content)
	}
	if resp.Usage.PromptTokens != 10 || resp.Usage.CompletionTokens != 5 {
		t.Errorf("usage mismatch: %+v", resp.Usage)
	}
}

func TestCalcCost(t *testing.T) {
	adapter := anthropic.New()

	usage := &vendor.Usage{
		PromptTokens:     1_000_000,
		CompletionTokens: 1_000_000,
	}

	cost, err := adapter.CalcCost(usage, "claude-sonnet-4-6")
	if err != nil {
		t.Fatalf("CalcCost: %v", err)
	}

	// claude-sonnet-4-6: $3.00 input + $15.00 output = $18.00
	expected := 18.00
	if cost.CostUSD != expected {
		t.Errorf("cost mismatch: got %.6f, want %.6f", cost.CostUSD, expected)
	}
}

func TestCalcCost_UnknownModel(t *testing.T) {
	adapter := anthropic.New()
	_, err := adapter.CalcCost(&vendor.Usage{}, "unknown-model")
	if err == nil {
		t.Error("should return error for unknown model")
	}
}

func TestHeaders_ContainsApiKey(t *testing.T) {
	adapter := anthropic.New()
	headers := adapter.Headers("sk-ant-test-key")

	if headers["x-api-key"] != "sk-ant-test-key" {
		t.Error("headers should contain api key")
	}
	if headers["anthropic-version"] == "" {
		t.Error("headers should contain anthropic-version")
	}
}

func TestImplementsAdapter(t *testing.T) {
	var _ vendor.Adapter = anthropic.New()
}
