package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ZachCharles666/gatelink/api/internal/auth"
	"github.com/ZachCharles666/gatelink/api/internal/buyer"
	"github.com/ZachCharles666/gatelink/api/internal/seller"
	"github.com/gin-gonic/gin"
)

func TestAdminTopupAndSettlementFlow(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	buyerSvc := buyer.NewService()
	sellerSvc := seller.NewService()
	handler := NewHandler(buyerSvc, sellerSvc)
	router := newAdminTestRouter(handler)
	adminToken := adminToken(t)

	buyerUser, err := buyerSvc.Register(context.Background(), "admin-flow@example.com", "", "pass123", "buyer-key")
	if err != nil {
		t.Fatalf("register buyer: %v", err)
	}
	firstTopup, err := buyerSvc.CreateTopup(context.Background(), buyerUser.ID, 100, "tx-admin-confirm", "TRC20")
	if err != nil {
		t.Fatalf("create topup: %v", err)
	}

	pendingResp := performAdminJSONRequest(t, router, http.MethodGet, "/api/v1/admin/topup/pending", adminToken, nil)
	pendingData := pendingResp["data"].(map[string]any)
	if len(pendingData["records"].([]any)) != 1 {
		t.Fatalf("unexpected pending topups: %#v", pendingData)
	}

	confirmResp := performAdminJSONRequest(t, router, http.MethodPost, "/api/v1/admin/topup/"+firstTopup.ID+"/confirm", adminToken, nil)
	confirmData := confirmResp["data"].(map[string]any)
	if confirmData["topup_id"] != firstTopup.ID || confirmData["balance_usd"].(float64) != 100 {
		t.Fatalf("unexpected confirm payload: %#v", confirmData)
	}

	buyerState, err := buyerSvc.GetBalance(context.Background(), buyerUser.ID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if buyerState.BalanceUSD != 100 {
		t.Fatalf("unexpected buyer balance after confirm: %+v", buyerState)
	}

	secondTopup, err := buyerSvc.CreateTopup(context.Background(), buyerUser.ID, 50, "tx-admin-reject", "ERC20")
	if err != nil {
		t.Fatalf("create second topup: %v", err)
	}

	rejectResp := performAdminJSONRequest(t, router, http.MethodPost, "/api/v1/admin/topup/"+secondTopup.ID+"/reject", adminToken, map[string]any{
		"reason": "hash mismatch",
	})
	rejectData := rejectResp["data"].(map[string]any)
	if rejectData["message"] != "充值已拒绝" {
		t.Fatalf("unexpected reject payload: %#v", rejectData)
	}

	topupRecords, total, err := buyerSvc.ListTopupRecords(context.Background(), buyerUser.ID, 1, 20)
	if err != nil {
		t.Fatalf("list topup records: %v", err)
	}
	if total != 2 || topupRecords[0].Status != "rejected" || topupRecords[1].Status != "confirmed" {
		t.Fatalf("unexpected topup states: total=%d records=%#v", total, topupRecords)
	}

	sellerUser, err := sellerSvc.Register(context.Background(), "13900000099", "Admin Seller")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}
	accountID, err := sellerSvc.PreCreateAccount(context.Background(), sellerUser.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}

	suspendResp := performAdminJSONRequest(t, router, http.MethodPost, "/api/v1/admin/accounts/"+accountID+"/force-suspend", adminToken, nil)
	suspendData := suspendResp["data"].(map[string]any)
	if suspendData["status"] != "suspended" {
		t.Fatalf("unexpected suspend payload: %#v", suspendData)
	}
	account, err := sellerSvc.GetAccount(context.Background(), accountID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if account.Status != "suspended" {
		t.Fatalf("account should be suspended after admin route assertion: %+v", account)
	}

	if _, err := sellerSvc.ApplyDispatchEarning(context.Background(), accountID, 10); err != nil {
		t.Fatalf("apply dispatch earning: %v", err)
	}
	settlement, err := sellerSvc.CreateSettlement(context.Background(), sellerUser.ID, 7.5, "2026-03-01T00:00:00Z", "2026-03-07T00:00:00Z")
	if err != nil {
		t.Fatalf("create settlement: %v", err)
	}

	pendingSettlementResp := performAdminJSONRequest(t, router, http.MethodGet, "/api/v1/admin/settlements/pending", adminToken, nil)
	pendingSettlementData := pendingSettlementResp["data"].(map[string]any)
	if len(pendingSettlementData["settlements"].([]any)) != 1 {
		t.Fatalf("unexpected pending settlements: %#v", pendingSettlementData)
	}

	payResp := performAdminJSONRequest(t, router, http.MethodPost, "/api/v1/admin/settlements/"+settlement.ID+"/pay", adminToken, map[string]any{
		"tx_hash": "0xsettlement",
	})
	payData := payResp["data"].(map[string]any)
	if payData["status"] != "paid" {
		t.Fatalf("unexpected pay payload: %#v", payData)
	}

	sellerState, err := sellerSvc.GetSeller(context.Background(), sellerUser.ID)
	if err != nil {
		t.Fatalf("get seller state: %v", err)
	}
	if sellerState.PendingEarnUSD != 0 || sellerState.TotalEarnedUSD != 7.5 {
		t.Fatalf("unexpected seller earnings state: %+v", sellerState)
	}
}

func TestAdminRoutesRequireAdminRole(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	gin.SetMode(gin.TestMode)

	handler := NewHandler(buyer.NewService(), seller.NewService())
	router := newAdminTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/topup/pending", nil)
	req.Header.Set("Authorization", "Bearer "+buyerToken(t, "buyer-1"))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d with body %s", recorder.Code, recorder.Body.String())
	}
}

func newAdminTestRouter(handler *Handler) *gin.Engine {
	router := gin.New()
	adminGroup := router.Group("/api/v1/admin")
	adminGroup.Use(auth.RequireRole("admin"))
	adminGroup.GET("/topup/pending", handler.ListPendingTopup)
	adminGroup.POST("/topup/:id/confirm", handler.ConfirmTopup)
	adminGroup.POST("/topup/:id/reject", handler.RejectTopup)
	adminGroup.GET("/settlements/pending", handler.ListPendingSettlements)
	adminGroup.POST("/settlements/:id/pay", handler.PaySettlement)
	adminGroup.POST("/accounts/:id/force-suspend", handler.ForceSuspend)
	return router
}

func adminToken(t *testing.T) string {
	t.Helper()
	token, err := auth.GenerateToken("admin-1", "admin")
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}
	return token
}

func buyerToken(t *testing.T, buyerID string) string {
	t.Helper()
	token, err := auth.GenerateToken(buyerID, "buyer")
	if err != nil {
		t.Fatalf("generate buyer token: %v", err)
	}
	return token
}

func performAdminJSONRequest(t *testing.T, router http.Handler, method, path, token string, body any) map[string]any {
	t.Helper()

	recorder := performAdminRequest(t, router, method, path, token, body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d with body %s", recorder.Code, recorder.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func performAdminRequest(t *testing.T, router http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
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
