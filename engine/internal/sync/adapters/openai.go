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

// OpenAIConsole 从 OpenAI 控制台 API 拉取用量
// 文档：https://platform.openai.com/docs/api-reference/usage
type OpenAIConsole struct {
	http *http.Client
}

func NewOpenAI() *OpenAIConsole {
	return &OpenAIConsole{
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (o *OpenAIConsole) Vendor() string { return "openai" }

func (o *OpenAIConsole) FetchDailyUsage(ctx context.Context, apiKey string, date time.Time) (*syncer.DailyUsage, error) {
	// OpenAI usage API: GET /v1/usage?date=YYYY-MM-DD
	dateStr := date.UTC().Format("2006-01-02")
	url := fmt.Sprintf("https://api.openai.com/v1/usage?date=%s", dateStr)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := o.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch usage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai usage API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			NContextTokensTotal     int `json:"n_context_tokens_total"`
			NGeneratedTokensTotal   int `json:"n_generated_tokens_total"`
		} `json:"data"`
		// OpenAI usage API doesn't directly return cost — needs model pricing lookup
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	usage := &syncer.DailyUsage{Date: date}
	for _, item := range result.Data {
		usage.InputTokens += item.NContextTokensTotal
		usage.OutputTokens += item.NGeneratedTokensTotal
	}
	// Note: OpenAI usage API doesn't return cost directly.
	// Cost calculation would require iterating per-model pricing.
	// For MVP: leave TotalCost = 0 and rely on local calculation.
	return usage, nil
}
