package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZachCharles666/gatelink/api/internal/accounting"
	"github.com/ZachCharles666/gatelink/api/internal/admin"
	"github.com/ZachCharles666/gatelink/api/internal/buyer"
	engineclient "github.com/ZachCharles666/gatelink/api/internal/engine"
	"github.com/ZachCharles666/gatelink/api/internal/proxy"
	"github.com/ZachCharles666/gatelink/api/internal/seller"
	"github.com/gin-gonic/gin"
)

type proxyEngineStub struct{}

func (proxyEngineStub) Audit(_ context.Context, _ engineclient.AuditRequest) (*engineclient.AuditResult, error) {
	return &engineclient.AuditResult{Safe: true}, nil
}

func (proxyEngineStub) Dispatch(_ context.Context, _ engineclient.DispatchRequest) (*engineclient.DispatchResult, error) {
	return nil, &engineclient.EngineError{Code: 4001, Msg: "no available account"}
}

func (proxyEngineStub) DispatchStream(_ context.Context, _ engineclient.DispatchRequest) (*engineclient.StreamResponse, error) {
	return &engineclient.StreamResponse{
		ContentType: "text/event-stream",
		Body:        io.NopCloser(strings.NewReader("data: [DONE]\n\n")),
	}, nil
}

func TestSetupRoutesMountsProxyEndpointsWithAPIKeyAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sellerSvc := seller.NewService()
	settlementSvc := accounting.NewSettlementService(sellerSvc)
	sellerH := seller.NewHandler(sellerSvc, settlementSvc, nil)

	buyerSvc := buyer.NewService()
	testBuyer, err := buyerSvc.Register(context.Background(), "router@example.com", "", "pass123", "router-key")
	if err != nil {
		t.Fatalf("register buyer: %v", err)
	}
	if err := buyerSvc.CreditBalance(context.Background(), testBuyer.ID, 10); err != nil {
		t.Fatalf("credit balance: %v", err)
	}
	buyerH := buyer.NewHandler(buyerSvc)
	adminH := admin.NewHandler(buyerSvc, sellerSvc)

	acctSvc := accounting.NewService(buyerSvc, sellerSvc)
	proxyEngine := &proxyEngineStub{}
	proxyH := proxy.NewHandler(proxyEngine, acctSvc)

	router := gin.New()
	SetupRoutes(router, sellerH, buyerH, adminH, proxyH, buyerSvc)

	modelsReq := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	modelsReq.Header.Set("Authorization", "Bearer router-key")
	modelsRec := httptest.NewRecorder()
	router.ServeHTTP(modelsRec, modelsReq)
	if modelsRec.Code != http.StatusOK {
		t.Fatalf("expected models 200, got %d with body %s", modelsRec.Code, modelsRec.Body.String())
	}

	var modelsPayload map[string]any
	if err := json.Unmarshal(modelsRec.Body.Bytes(), &modelsPayload); err != nil {
		t.Fatalf("decode models response: %v", err)
	}
	if modelsPayload["object"] != "list" {
		t.Fatalf("unexpected models payload: %#v", modelsPayload)
	}

	unauthReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte(`{"model":"claude-sonnet-4-6","messages":[{"role":"user","content":"hello"}]}`)))
	unauthReq.Header.Set("Content-Type", "application/json")
	unauthRec := httptest.NewRecorder()
	router.ServeHTTP(unauthRec, unauthReq)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized 401, got %d with body %s", unauthRec.Code, unauthRec.Body.String())
	}
}
