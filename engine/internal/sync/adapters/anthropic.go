// Package adapters 提供各厂商控制台 API 的具体实现
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	syncer "github.com/yourname/gatelink-engine/internal/sync"
)

// AnthropicConsole 从 Anthropic 控制台 API 拉取用量
// 文档：https://docs.anthropic.com/en/api/usage
type AnthropicConsole struct {
	http *http.Client
}

func NewAnthropic() *AnthropicConsole {
	return &AnthropicConsole{
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *AnthropicConsole) Vendor() string { return "anthropic" }

func (a *AnthropicConsole) FetchDailyUsage(ctx context.Context, apiKey string, date time.Time) (*syncer.DailyUsage, error) {
	// Anthropic usage API: GET /v1/usage?start_time=...&end_time=...
	start := date.UTC().Format("2006-01-02T15:04:05Z")
	end := date.UTC().Add(24 * time.Hour).Format("2006-01-02T15:04:05Z")
	url := fmt.Sprintf("https://api.anthropic.com/v1/usage?start_time=%s&end_time=%s", start, end)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic usage API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			InputTokens  int     `json:"input_tokens"`
			OutputTokens int     `json:"output_tokens"`
			CostUSD      float64 `json:"cost"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	usage := &syncer.DailyUsage{Date: date}
	for _, item := range result.Data {
		usage.InputTokens += item.InputTokens
		usage.OutputTokens += item.OutputTokens
		usage.TotalCost += item.CostUSD
	}
	return usage, nil
}
