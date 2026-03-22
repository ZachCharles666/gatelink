package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		baseURL:    getEngineURL(),
		httpClient: &http.Client{Timeout: 35 * time.Second},
	}
}

func getEngineURL() string {
	url := os.Getenv("ENGINE_INTERNAL_URL")
	if url == "" {
		return "http://engine:8081"
	}
	return url
}

type engineResp struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*engineResp, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("engine request failed: %w", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read engine response: %w", err)
	}

	var result engineResp
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("decode engine response (status=%d): %w", resp.StatusCode, err)
	}

	return &result, nil
}

func decodeData(resp *engineResp, dst any) error {
	if len(resp.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Data, dst); err != nil {
		return fmt.Errorf("decode engine data: %w", err)
	}
	return nil
}

type EngineError struct {
	Code int
	Msg  string
}

func (e *EngineError) Error() string {
	return fmt.Sprintf("engine error %d: %s", e.Code, e.Msg)
}

func IsNoAccount(err error) bool {
	e, ok := err.(*EngineError)
	return ok && e.Code == 4001
}

func IsAuditFail(err error) bool {
	e, ok := err.(*EngineError)
	return ok && e.Code == 4003
}

func IsVendorRateLimited(err error) bool {
	e, ok := err.(*EngineError)
	return ok && e.Code == 4004
}

func IsVendorError(err error) bool {
	e, ok := err.(*EngineError)
	return ok && e.Code == 5001
}

func IsNotFound(err error) bool {
	e, ok := err.(*EngineError)
	return ok && e.Code == 1004
}

type VerifyResult struct {
	AccountID string `json:"account_id"`
	Vendor    string `json:"vendor"`
	Valid     bool   `json:"valid"`
	ErrorMsg  string `json:"error_msg"`
}

type CreateAccountRequest struct {
	SellerID             string  `json:"seller_id"`
	Vendor               string  `json:"vendor"`
	APIKey               string  `json:"api_key"`
	TotalCreditsUSD      float64 `json:"total_credits_usd"`
	AuthorizedCreditsUSD float64 `json:"authorized_credits_usd"`
	ExpectedRate         float64 `json:"expected_rate,omitempty"`
	ExpireAt             string  `json:"expire_at"`
}

type CreateAccountResult struct {
	AccountID  string `json:"account_id"`
	APIKeyHint string `json:"api_key_hint"`
	Vendor     string `json:"vendor"`
	Status     string `json:"status"`
}

func (c *Client) CreateAccount(ctx context.Context, req CreateAccountRequest) (*CreateAccountResult, error) {
	resp, err := c.do(ctx, http.MethodPost, "/internal/v1/accounts", req)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var result CreateAccountResult
	if err := decodeData(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) VerifyAccount(ctx context.Context, accountID, apiKey string) (*VerifyResult, error) {
	resp, err := c.do(ctx, http.MethodPost, "/internal/v1/accounts/"+accountID+"/verify", map[string]string{
		"api_key": apiKey,
	})
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var result VerifyResult
	if err := decodeData(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type PoolStatus struct {
	PoolCounts map[string]int `json:"pool_counts"`
}

func (c *Client) GetPoolStatus(ctx context.Context) (*PoolStatus, error) {
	resp, err := c.do(ctx, http.MethodGet, "/internal/v1/pool/status", nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var status PoolStatus
	if err := decodeData(resp, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

type HealthEvent struct {
	Type       string `json:"type"`
	Delta      int    `json:"delta"`
	ScoreAfter int    `json:"score_after"`
	CreatedAt  string `json:"created_at"`
}

type AccountHealth struct {
	AccountID    string        `json:"account_id"`
	HealthScore  int           `json:"health_score"`
	Status       string        `json:"status"`
	Vendor       string        `json:"vendor"`
	RecentEvents []HealthEvent `json:"recent_events"`
}

func (c *Client) GetAccountHealth(ctx context.Context, accountID string) (*AccountHealth, error) {
	resp, err := c.do(ctx, http.MethodGet, "/internal/v1/accounts/"+accountID+"/health", nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var health AccountHealth
	if err := decodeData(resp, &health); err != nil {
		return nil, err
	}
	return &health, nil
}

type UsageRecord struct {
	Date         string  `json:"date"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	RequestCount int     `json:"request_count"`
}

type ConsoleUsage struct {
	AccountID string        `json:"account_id"`
	Records   []UsageRecord `json:"records"`
}

func (c *Client) GetConsoleUsage(ctx context.Context, accountID string) (*ConsoleUsage, error) {
	resp, err := c.do(ctx, http.MethodGet, "/internal/v1/accounts/"+accountID+"/console-usage", nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var usage ConsoleUsage
	if err := decodeData(resp, &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

type DiffEvent struct {
	Type      string `json:"type"`
	Detail    any    `json:"detail"`
	CreatedAt string `json:"created_at"`
}

type DiffResult struct {
	AccountID string      `json:"account_id"`
	Diffs     []DiffEvent `json:"diffs"`
}

func (c *Client) GetAccountDiff(ctx context.Context, accountID string) (*DiffResult, error) {
	resp, err := c.do(ctx, http.MethodGet, "/internal/v1/accounts/"+accountID+"/diff", nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var diff DiffResult
	if err := decodeData(resp, &diff); err != nil {
		return nil, err
	}
	return &diff, nil
}

type AccountEvent struct {
	Type       string `json:"type"`
	Delta      int    `json:"delta"`
	ScoreAfter int    `json:"score_after"`
	Detail     any    `json:"detail"`
	CreatedAt  string `json:"created_at"`
}

type AccountEvents struct {
	AccountID string         `json:"account_id"`
	Events    []AccountEvent `json:"events"`
	Count     int            `json:"count"`
}

func (c *Client) GetAccountEvents(ctx context.Context, accountID string, limit int) (*AccountEvents, error) {
	path := "/internal/v1/accounts/" + accountID + "/events"
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}

	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var events AccountEvents
	if err := decodeData(resp, &events); err != nil {
		return nil, err
	}
	return &events, nil
}

type AuditRequest struct {
	Messages []string `json:"messages"`
	BuyerID  string   `json:"buyer_id,omitempty"`
}

type AuditResult struct {
	Safe   bool   `json:"safe"`
	Level  int    `json:"level"`
	Reason string `json:"reason"`
}

func (c *Client) Audit(ctx context.Context, req AuditRequest) (*AuditResult, error) {
	resp, err := c.do(ctx, http.MethodPost, "/internal/v1/audit", req)
	if err != nil {
		return nil, err
	}
	if resp.Code == 4003 {
		return &AuditResult{Safe: false, Reason: resp.Msg}, nil
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var result AuditResult
	if err := decodeData(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type DispatchRequest struct {
	BuyerID         string    `json:"buyer_id"`
	Vendor          string    `json:"vendor"`
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	Stream          bool      `json:"stream"`
	MaxTokens       int       `json:"max_tokens,omitempty"`
	Temperature     float64   `json:"temperature,omitempty"`
	BuyerChargeRate float64   `json:"buyer_charge_rate,omitempty"`
}

type DispatchResult struct {
	Response     json.RawMessage `json:"response"`
	AccountID    string          `json:"account_id"`
	Vendor       string          `json:"vendor"`
	CostUSD      float64         `json:"cost_usd"`
	InputTokens  int             `json:"input_tokens"`
	OutputTokens int             `json:"output_tokens"`
}

type StreamResponse struct {
	ContentType string
	Body        io.ReadCloser
}

func (c *Client) Dispatch(ctx context.Context, req DispatchRequest) (*DispatchResult, error) {
	resp, err := c.do(ctx, http.MethodPost, "/internal/v1/dispatch", req)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, &EngineError{Code: resp.Code, Msg: resp.Msg}
	}

	var result DispatchResult
	if err := decodeData(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DispatchStream(ctx context.Context, req DispatchRequest) (*StreamResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/internal/v1/dispatch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream, application/json")

	streamClient := *c.httpClient
	streamClient.Timeout = 0

	resp, err := streamClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("engine stream request failed: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "text/event-stream") {
		return &StreamResponse{
			ContentType: contentType,
			Body:        resp.Body,
		}, nil
	}

	defer resp.Body.Close()
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read engine stream response: %w", err)
	}

	var result engineResp
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return nil, fmt.Errorf("decode engine stream response (status=%d): %w", resp.StatusCode, err)
	}
	if result.Code != 0 {
		return nil, &EngineError{Code: result.Code, Msg: result.Msg}
	}

	return nil, fmt.Errorf("unexpected non-stream engine response")
}
