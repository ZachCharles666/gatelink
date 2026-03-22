package buyer

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZachCharles666/gatelink/api/internal/auth"
	"github.com/gin-gonic/gin"
)

type fakeAPIKeyInvalidator struct {
	calls []string
}

func (f *fakeAPIKeyInvalidator) InvalidateAPIKey(_ context.Context, apiKey string) {
	f.calls = append(f.calls, apiKey)
}

func TestBuyerWeek3Day1Flow(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	svc := NewService()
	buyer, err := svc.Register(context.Background(), "buyer@example.com", "", "pass123", "api-key-1")
	if err != nil {
		t.Fatalf("register buyer: %v", err)
	}

	handler := NewHandler(svc)
	router := newBuyerTestRouter(handler)
	token := buyerToken(t, buyer.ID)

	balanceResp := performBuyerJSONRequest(t, router, http.MethodGet, "/api/v1/buyer/balance", token, nil)
	if balanceResp["code"].(float64) != 0 {
		t.Fatalf("unexpected balance response: %#v", balanceResp)
	}
	balanceData := balanceResp["data"].(map[string]any)
	if balanceData["balance_usd"].(float64) != 0 || balanceData["total_consumed_usd"].(float64) != 0 {
		t.Fatalf("unexpected balance payload: %#v", balanceData)
	}

	topupResp := performBuyerJSONRequest(t, router, http.MethodPost, "/api/v1/buyer/topup", token, map[string]any{
		"amount_usd": 99.5,
		"tx_hash":    "tx-1",
		"network":    "TRC20",
	})
	if topupResp["code"].(float64) != 0 {
		t.Fatalf("unexpected topup response: %#v", topupResp)
	}
	topupData := topupResp["data"].(map[string]any)
	if topupData["status"] != "pending" {
		t.Fatalf("unexpected topup payload: %#v", topupData)
	}

	topupRecordsResp := performBuyerJSONRequest(t, router, http.MethodGet, "/api/v1/buyer/topup/records?page=1", token, nil)
	if topupRecordsResp["code"].(float64) != 0 {
		t.Fatalf("unexpected topup records response: %#v", topupRecordsResp)
	}
	topupRecordsData := topupRecordsResp["data"].(map[string]any)
	if topupRecordsData["total"].(float64) != 1 {
		t.Fatalf("unexpected topup records total: %#v", topupRecordsData)
	}
	if len(topupRecordsData["records"].([]any)) != 1 {
		t.Fatalf("unexpected topup records payload: %#v", topupRecordsData)
	}

	usageResp := performBuyerJSONRequest(t, router, http.MethodGet, "/api/v1/buyer/usage?page=1", token, nil)
	if usageResp["code"].(float64) != 0 {
		t.Fatalf("unexpected usage response: %#v", usageResp)
	}
	usageData := usageResp["data"].(map[string]any)
	if usageData["total"].(float64) != 0 {
		t.Fatalf("unexpected usage total: %#v", usageData)
	}
	if len(usageData["records"].([]any)) != 0 {
		t.Fatalf("unexpected usage records: %#v", usageData)
	}

	resetResp := performBuyerJSONRequest(t, router, http.MethodPost, "/api/v1/buyer/apikeys/reset", token, nil)
	if resetResp["code"].(float64) != 0 {
		t.Fatalf("unexpected reset response: %#v", resetResp)
	}
	resetData := resetResp["data"].(map[string]any)
	newKey := resetData["api_key"].(string)
	if newKey == "" || newKey == "api-key-1" {
		t.Fatalf("unexpected reset payload: %#v", resetData)
	}

	if _, err := svc.FindByAPIKey(context.Background(), "api-key-1"); err == nil {
		t.Fatalf("old api key should be invalidated")
	}
	if info, err := svc.FindByAPIKey(context.Background(), newKey); err != nil || info.ID != buyer.ID {
		t.Fatalf("new api key not active: info=%#v err=%v", info, err)
	}
}

func TestBuyerDuplicateTopupRejected(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	svc := NewService()
	buyer, err := svc.Register(context.Background(), "buyer2@example.com", "", "pass123", "api-key-2")
	if err != nil {
		t.Fatalf("register buyer: %v", err)
	}

	handler := NewHandler(svc)
	router := newBuyerTestRouter(handler)
	token := buyerToken(t, buyer.ID)

	first := performBuyerRequest(t, router, http.MethodPost, "/api/v1/buyer/topup", token, map[string]any{
		"amount_usd": 20,
		"tx_hash":    "dup-hash",
		"network":    "ERC20",
	})
	if first.Code != http.StatusOK {
		t.Fatalf("expected first topup to succeed, got %d with body %s", first.Code, first.Body.String())
	}

	second := performBuyerRequest(t, router, http.MethodPost, "/api/v1/buyer/topup", token, map[string]any{
		"amount_usd": 20,
		"tx_hash":    "dup-hash",
		"network":    "ERC20",
	})
	if second.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate topup to fail with 400, got %d with body %s", second.Code, second.Body.String())
	}
}

func TestBuyerResetAPIKeyInvalidatesCache(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	svc := NewService()
	buyer, err := svc.Register(context.Background(), "buyer3@example.com", "", "pass123", "api-key-3")
	if err != nil {
		t.Fatalf("register buyer: %v", err)
	}

	invalidator := &fakeAPIKeyInvalidator{}
	handler := NewHandler(svc, invalidator)
	router := newBuyerTestRouter(handler)
	token := buyerToken(t, buyer.ID)

	resetResp := performBuyerJSONRequest(t, router, http.MethodPost, "/api/v1/buyer/apikeys/reset", token, nil)
	if resetResp["code"].(float64) != 0 {
		t.Fatalf("unexpected reset response: %#v", resetResp)
	}

	newKey := resetResp["data"].(map[string]any)["api_key"].(string)
	if len(invalidator.calls) != 2 {
		t.Fatalf("expected 2 invalidation calls, got %#v", invalidator.calls)
	}
	if invalidator.calls[0] != "api-key-3" {
		t.Fatalf("expected old key invalidation, got %#v", invalidator.calls)
	}
	if invalidator.calls[1] != newKey {
		t.Fatalf("expected new key invalidation, got %#v", invalidator.calls)
	}
}

func newBuyerTestRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	buyerGroup := router.Group("/api/v1/buyer")
	buyerGroup.Use(auth.RequireRole("buyer"))
	buyerGroup.GET("/balance", handler.GetBalance)
	buyerGroup.GET("/usage", handler.GetUsage)
	buyerGroup.POST("/topup", handler.Topup)
	buyerGroup.GET("/topup/records", handler.ListTopupRecords)
	buyerGroup.POST("/apikeys/reset", handler.ResetAPIKey)
	return router
}

func buyerToken(t *testing.T, buyerID string) string {
	t.Helper()

	token, err := auth.GenerateToken(buyerID, "buyer")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func performBuyerJSONRequest(t *testing.T, router http.Handler, method, path, token string, body any) map[string]any {
	t.Helper()

	recorder := performBuyerRequest(t, router, method, path, token, body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return resp
}

func performBuyerRequest(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}
