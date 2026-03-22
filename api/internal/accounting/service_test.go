package accounting

import (
	"context"
	"errors"
	"testing"

	"github.com/ZachCharles666/gatelink/api/internal/buyer"
	"github.com/ZachCharles666/gatelink/api/internal/engine"
	"github.com/ZachCharles666/gatelink/api/internal/seller"
)

func TestChargeAfterDispatchUpdatesBuyerAndSellerState(t *testing.T) {
	buyerSvc := buyer.NewService()
	sellerSvc := seller.NewService()
	acctSvc := NewService(buyerSvc, sellerSvc)

	testBuyer, err := buyerSvc.Register(context.Background(), "acct@example.com", "", "pass123", "buyer-key")
	if err != nil {
		t.Fatalf("register buyer: %v", err)
	}
	if err := buyerSvc.CreditBalance(context.Background(), testBuyer.ID, 50); err != nil {
		t.Fatalf("credit balance: %v", err)
	}

	testSeller, err := sellerSvc.Register(context.Background(), "13900000009", "Seller A")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}
	accountID, err := sellerSvc.PreCreateAccount(context.Background(), testSeller.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}

	err = acctSvc.ChargeAfterDispatch(context.Background(), testBuyer.ID, &engine.DispatchResult{
		AccountID:    accountID,
		Vendor:       "anthropic",
		CostUSD:      10,
		InputTokens:  120,
		OutputTokens: 80,
	})
	if err != nil {
		t.Fatalf("charge after dispatch: %v", err)
	}

	updatedBuyer, err := buyerSvc.GetBalance(context.Background(), testBuyer.ID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if updatedBuyer.BalanceUSD != 41.2 {
		t.Fatalf("unexpected buyer balance: %+v", updatedBuyer)
	}
	if updatedBuyer.TotalConsumedUSD != 8.8 {
		t.Fatalf("unexpected consumed total: %+v", updatedBuyer)
	}

	usageRecords, total, err := buyerSvc.GetUsageRecords(context.Background(), testBuyer.ID, 1, 20)
	if err != nil {
		t.Fatalf("get usage records: %v", err)
	}
	if total != 1 || len(usageRecords) != 1 {
		t.Fatalf("unexpected usage record count: total=%d len=%d", total, len(usageRecords))
	}
	if usageRecords[0].Vendor != "anthropic" || usageRecords[0].BuyerChargedUSD != 8.8 {
		t.Fatalf("unexpected usage record: %+v", usageRecords[0])
	}

	updatedSeller, err := sellerSvc.GetSeller(context.Background(), testSeller.ID)
	if err != nil {
		t.Fatalf("get seller: %v", err)
	}
	if updatedSeller.PendingEarnUSD != 7.5 {
		t.Fatalf("unexpected seller pending earn: %+v", updatedSeller)
	}

	account, err := sellerSvc.GetAccount(context.Background(), accountID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if account.ConsumedCreditsUSD != 10 {
		t.Fatalf("unexpected consumed credits: %+v", account)
	}
}

func TestChargeAfterDispatchRejectsInsufficientBalance(t *testing.T) {
	buyerSvc := buyer.NewService()
	sellerSvc := seller.NewService()
	acctSvc := NewService(buyerSvc, sellerSvc)

	testBuyer, err := buyerSvc.Register(context.Background(), "acct2@example.com", "", "pass123", "buyer-key-2")
	if err != nil {
		t.Fatalf("register buyer: %v", err)
	}
	if err := buyerSvc.CreditBalance(context.Background(), testBuyer.ID, 5); err != nil {
		t.Fatalf("credit balance: %v", err)
	}

	testSeller, err := sellerSvc.Register(context.Background(), "13900000010", "Seller B")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}
	accountID, err := sellerSvc.PreCreateAccount(context.Background(), testSeller.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}

	err = acctSvc.ChargeAfterDispatch(context.Background(), testBuyer.ID, &engine.DispatchResult{
		AccountID: accountID,
		Vendor:    "anthropic",
		CostUSD:   10,
	})
	if !errors.Is(err, buyer.ErrInsufficientBalance) {
		t.Fatalf("expected insufficient balance error, got %v", err)
	}

	updatedBuyer, err := buyerSvc.GetBalance(context.Background(), testBuyer.ID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if updatedBuyer.BalanceUSD != 5 || updatedBuyer.TotalConsumedUSD != 0 {
		t.Fatalf("buyer should not be charged: %+v", updatedBuyer)
	}

	updatedSeller, err := sellerSvc.GetSeller(context.Background(), testSeller.ID)
	if err != nil {
		t.Fatalf("get seller: %v", err)
	}
	if updatedSeller.PendingEarnUSD != 0 {
		t.Fatalf("seller should not be credited: %+v", updatedSeller)
	}
}

func TestListModelsReturnsDefaultCatalog(t *testing.T) {
	acctSvc := NewService(buyer.NewService(), seller.NewService())

	models, err := acctSvc.ListModels(context.Background())
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected non-empty model catalog")
	}
	if models[0].Model == "" || models[0].Vendor == "" {
		t.Fatalf("unexpected first model: %+v", models[0])
	}
}

func TestChargeAfterStreamUsesModelPricing(t *testing.T) {
	buyerSvc := buyer.NewService()
	sellerSvc := seller.NewService()
	acctSvc := NewService(buyerSvc, sellerSvc)

	testBuyer, err := buyerSvc.Register(context.Background(), "streamacct@example.com", "", "pass123", "buyer-key-stream")
	if err != nil {
		t.Fatalf("register buyer: %v", err)
	}
	if err := buyerSvc.CreditBalance(context.Background(), testBuyer.ID, 20); err != nil {
		t.Fatalf("credit balance: %v", err)
	}

	testSeller, err := sellerSvc.Register(context.Background(), "13900000022", "Seller Stream")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}
	accountID, err := sellerSvc.PreCreateAccount(context.Background(), testSeller.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}

	err = acctSvc.ChargeAfterStream(context.Background(), testBuyer.ID, accountID, "anthropic", "claude-sonnet-4-6", 1000000, 0)
	if err != nil {
		t.Fatalf("charge after stream: %v", err)
	}

	updatedBuyer, err := buyerSvc.GetBalance(context.Background(), testBuyer.ID)
	if err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if updatedBuyer.BalanceUSD != 17.36 || updatedBuyer.TotalConsumedUSD != 2.64 {
		t.Fatalf("unexpected buyer state after stream charge: %+v", updatedBuyer)
	}

	updatedSeller, err := sellerSvc.GetSeller(context.Background(), testSeller.ID)
	if err != nil {
		t.Fatalf("get seller: %v", err)
	}
	if updatedSeller.PendingEarnUSD != 2.25 {
		t.Fatalf("unexpected seller state after stream charge: %+v", updatedSeller)
	}
}
