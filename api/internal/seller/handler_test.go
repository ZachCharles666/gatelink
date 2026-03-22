package seller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZachCharles666/gatelink/api/internal/auth"
	engineclient "github.com/ZachCharles666/gatelink/api/internal/engine"
	"github.com/ZachCharles666/gatelink/api/internal/response"
	"github.com/gin-gonic/gin"
)

type fakeSettlement struct {
	svc *Service
}

func (f fakeSettlement) RequestSettlement(ctx context.Context, sellerID string) error {
	sellerState, err := f.svc.GetSeller(ctx, sellerID)
	if err != nil {
		return err
	}
	if sellerState.PendingEarnUSD < 10 {
		return errors.New("minimum settlement amount is $10, current: $0.00")
	}

	_, err = f.svc.CreateSettlement(ctx, sellerID, sellerState.PendingEarnUSD, "2026-03-01T00:00:00Z", "2026-03-15T00:00:00Z")
	return err
}

type fakeEngine struct{}

func (fakeEngine) CreateAccount(_ context.Context, req engineclient.CreateAccountRequest) (*engineclient.CreateAccountResult, error) {
	return &engineclient.CreateAccountResult{
		AccountID:  "eng-account-1",
		APIKeyHint: "sk-ant-***",
		Vendor:     req.Vendor,
		Status:     "active",
	}, nil
}

func (fakeEngine) GetAccountHealth(_ context.Context, accountID string) (*engineclient.AccountHealth, error) {
	return &engineclient.AccountHealth{
		AccountID:   accountID,
		HealthScore: 88,
		Status:      "healthy",
		Vendor:      "anthropic",
	}, nil
}

func (fakeEngine) GetConsoleUsage(_ context.Context, accountID string) (*engineclient.ConsoleUsage, error) {
	return &engineclient.ConsoleUsage{
		AccountID: accountID,
		Records: []engineclient.UsageRecord{
			{
				Date:         "2026-03-21",
				TotalCostUSD: 12.5,
				InputTokens:  1000,
				OutputTokens: 800,
				RequestCount: 3,
			},
		},
	}, nil
}

func (fakeEngine) GetAccountDiff(_ context.Context, accountID string) (*engineclient.DiffResult, error) {
	return &engineclient.DiffResult{
		AccountID: accountID,
		Diffs: []engineclient.DiffEvent{
			{
				Type:      "pass",
				Detail:    "no drift",
				CreatedAt: "2026-03-21T00:00:00Z",
			},
		},
	}, nil
}

type fakeCreateInvalidParamEngine struct{}

func (fakeCreateInvalidParamEngine) CreateAccount(_ context.Context, _ engineclient.CreateAccountRequest) (*engineclient.CreateAccountResult, error) {
	return nil, &engineclient.EngineError{Code: response.CodeInvalidParam, Msg: "unsupported vendor: unsupported"}
}

func (fakeCreateInvalidParamEngine) GetAccountHealth(_ context.Context, accountID string) (*engineclient.AccountHealth, error) {
	return nil, errors.New("not used in this test: " + accountID)
}

func (fakeCreateInvalidParamEngine) GetConsoleUsage(_ context.Context, accountID string) (*engineclient.ConsoleUsage, error) {
	return nil, errors.New("not used in this test: " + accountID)
}

func (fakeCreateInvalidParamEngine) GetAccountDiff(_ context.Context, accountID string) (*engineclient.DiffResult, error) {
	return nil, errors.New("not used in this test: " + accountID)
}

func TestSellerWeek2Flow(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	svc := NewService()
	settlementSvc := fakeSettlement{svc: svc}
	handler := NewHandler(svc, settlementSvc, fakeEngine{})
	seller, err := svc.Register(context.Background(), "13900000001", "Seller A")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}

	router := newSellerTestRouter(handler)
	token := sellerToken(t, seller.ID)

	accountResp := performJSONRequest(t, router, http.MethodPost, "/api/v1/seller/accounts", token, map[string]any{
		"vendor":                 "anthropic",
		"api_key":                "sk-ant-test",
		"authorized_credits_usd": 100,
		"expected_rate":          0.75,
		"expire_at":              "2026-09-01T00:00:00Z",
	})

	if accountResp["code"].(float64) != 0 {
		t.Fatalf("unexpected add account response: %#v", accountResp)
	}
	accountData := accountResp["data"].(map[string]any)
	accountID := accountData["account_id"].(string)
	if accountData["status"] != "active" {
		t.Fatalf("unexpected account status: %#v", accountData)
	}
	if accountData["api_key_hint"] != "sk-ant-***" {
		t.Fatalf("unexpected api key hint: %#v", accountData)
	}

	authResp := performJSONRequest(t, router, http.MethodPatch, "/api/v1/seller/accounts/"+accountID+"/authorization", token, map[string]any{
		"authorized_credits_usd": 150,
	})
	if authResp["code"].(float64) != 0 {
		t.Fatalf("unexpected update authorization response: %#v", authResp)
	}
	authData := authResp["data"].(map[string]any)
	if authData["authorized_credits_usd"].(float64) != 150 {
		t.Fatalf("authorization not updated: %#v", authData)
	}

	accountsResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/seller/accounts", token, nil)
	if accountsResp["code"].(float64) != 0 {
		t.Fatalf("unexpected accounts response: %#v", accountsResp)
	}
	accountsData := accountsResp["data"].(map[string]any)
	if len(accountsData["accounts"].([]any)) != 1 {
		t.Fatalf("unexpected accounts list: %#v", accountsData)
	}

	accountDetailResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/seller/accounts/"+accountID, token, nil)
	if accountDetailResp["code"].(float64) != 0 {
		t.Fatalf("unexpected account detail response: %#v", accountDetailResp)
	}
	accountDetail := accountDetailResp["data"].(map[string]any)
	if accountDetail["id"] != accountID {
		t.Fatalf("unexpected account detail payload: %#v", accountDetail)
	}

	usageResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/seller/accounts/"+accountID+"/usage", token, nil)
	if usageResp["code"].(float64) != 0 {
		t.Fatalf("unexpected usage response: %#v", usageResp)
	}
	usageData := usageResp["data"].(map[string]any)
	if usageData["health_score"].(float64) != 88 {
		t.Fatalf("unexpected health score: %#v", usageData)
	}
	if len(usageData["daily_records"].([]any)) != 1 {
		t.Fatalf("unexpected daily records: %#v", usageData)
	}
	if len(usageData["diff_events"].([]any)) != 1 {
		t.Fatalf("unexpected diff events: %#v", usageData)
	}
	diffDetail := usageData["diff_events"].([]any)[0].(map[string]any)["detail"]
	if diffDetail != "no drift" {
		t.Fatalf("unexpected diff detail: %#v", usageData)
	}

	revokeResp := performJSONRequest(t, router, http.MethodDelete, "/api/v1/seller/accounts/"+accountID+"/authorization", token, nil)
	if revokeResp["code"].(float64) != 0 {
		t.Fatalf("unexpected revoke response: %#v", revokeResp)
	}
	revokeData := revokeResp["data"].(map[string]any)
	if revokeData["revoked_amount_usd"].(float64) != 150 {
		t.Fatalf("unexpected revoked amount: %#v", revokeData)
	}

	earningsResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/seller/earnings", token, nil)
	if earningsResp["code"].(float64) != 0 {
		t.Fatalf("unexpected earnings response: %#v", earningsResp)
	}
	earningsData := earningsResp["data"].(map[string]any)
	if earningsData["pending_usd"].(float64) != 0 || earningsData["total_earned_usd"].(float64) != 0 {
		t.Fatalf("unexpected earnings data: %#v", earningsData)
	}

	settlementsResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/seller/settlements?page=1", token, nil)
	if settlementsResp["code"].(float64) != 0 {
		t.Fatalf("unexpected settlements response: %#v", settlementsResp)
	}
	settlementsData := settlementsResp["data"].(map[string]any)
	if settlementsData["total"].(float64) != 0 {
		t.Fatalf("unexpected settlements total: %#v", settlementsData)
	}
	if len(settlementsData["settlements"].([]any)) != 0 {
		t.Fatalf("unexpected settlements list: %#v", settlementsData)
	}
}

func TestSellerAccountForbiddenForOtherSeller(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	svc := NewService()
	settlementSvc := fakeSettlement{svc: svc}
	handler := NewHandler(svc, settlementSvc, fakeEngine{})
	sellerA, err := svc.Register(context.Background(), "13900000001", "Seller A")
	if err != nil {
		t.Fatalf("register seller A: %v", err)
	}
	sellerB, err := svc.Register(context.Background(), "13900000002", "Seller B")
	if err != nil {
		t.Fatalf("register seller B: %v", err)
	}

	accountID, err := svc.PreCreateAccount(context.Background(), sellerA.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}

	router := newSellerTestRouter(handler)
	token := sellerToken(t, sellerB.ID)
	recorder := performRequest(t, router, http.MethodGet, "/api/v1/seller/accounts/"+accountID, token, nil)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode forbidden response: %v", err)
	}
	if resp["code"].(float64) != 1003 {
		t.Fatalf("unexpected forbidden payload: %#v", resp)
	}
}

func TestSellerAddAccountPropagatesEngineBadRequest(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	svc := NewService()
	settlementSvc := fakeSettlement{svc: svc}
	handler := NewHandler(svc, settlementSvc, fakeCreateInvalidParamEngine{})
	seller, err := svc.Register(context.Background(), "13900000003", "Seller C")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}

	router := newSellerTestRouter(handler)
	token := sellerToken(t, seller.ID)
	recorder := performRequest(t, router, http.MethodPost, "/api/v1/seller/accounts", token, map[string]any{
		"vendor":                 "anthropic",
		"api_key":                "sk-ant-test",
		"authorized_credits_usd": 100,
		"expected_rate":          0.75,
		"expire_at":              "2026-09-01T00:00:00Z",
	})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"].(float64) != float64(response.CodeInvalidParam) {
		t.Fatalf("unexpected response payload: %#v", resp)
	}
	if resp["msg"] != "unsupported vendor: unsupported" {
		t.Fatalf("unexpected response message: %#v", resp)
	}
}

func TestSellerRequestSettlement(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	svc := NewService()
	settlementSvc := fakeSettlement{svc: svc}
	handler := NewHandler(svc, settlementSvc, fakeEngine{})
	sellerUser, err := svc.Register(context.Background(), "13900000004", "Seller D")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}

	accountID, err := svc.PreCreateAccount(context.Background(), sellerUser.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}
	if _, err := svc.ApplyDispatchEarning(context.Background(), accountID, 20); err != nil {
		t.Fatalf("apply dispatch earning: %v", err)
	}

	router := newSellerTestRouter(handler)
	token := sellerToken(t, sellerUser.ID)
	resp := performJSONRequest(t, router, http.MethodPost, "/api/v1/seller/settlements/request", token, nil)
	if resp["code"].(float64) != 0 {
		t.Fatalf("unexpected settlement request response: %#v", resp)
	}

	settlementsResp := performJSONRequest(t, router, http.MethodGet, "/api/v1/seller/settlements?page=1", token, nil)
	settlementsData := settlementsResp["data"].(map[string]any)
	if settlementsData["total"].(float64) != 1 {
		t.Fatalf("unexpected settlements total after request: %#v", settlementsData)
	}
}

func newSellerTestRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	sellerGroup := router.Group("/api/v1/seller")
	sellerGroup.Use(auth.RequireRole("seller"))
	sellerGroup.GET("/accounts", handler.ListAccounts)
	sellerGroup.GET("/accounts/:id", handler.GetAccount)
	sellerGroup.POST("/accounts", handler.AddAccount)
	sellerGroup.PATCH("/accounts/:id/authorization", handler.UpdateAuthorization)
	sellerGroup.DELETE("/accounts/:id/authorization", handler.RevokeAuthorization)
	sellerGroup.GET("/accounts/:id/usage", handler.GetAccountUsage)
	sellerGroup.GET("/earnings", handler.GetEarnings)
	sellerGroup.GET("/settlements", handler.ListSettlements)
	sellerGroup.POST("/settlements/request", handler.RequestSettlement)
	return router
}

func sellerToken(t *testing.T, sellerID string) string {
	t.Helper()

	token, err := auth.GenerateToken(sellerID, "seller")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func performJSONRequest(t *testing.T, router http.Handler, method, path, token string, body any) map[string]any {
	t.Helper()

	recorder := performRequest(t, router, method, path, token, body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return resp
}

func performRequest(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
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
