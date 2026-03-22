package accounting

import (
	"context"
	"testing"

	"github.com/ZachCharles666/gatelink/api/internal/seller"
)

func TestRequestSettlementCreatesPendingRecordWhenThresholdMet(t *testing.T) {
	sellerSvc := seller.NewService()
	settlementSvc := NewSettlementService(sellerSvc)

	sellerUser, err := sellerSvc.Register(context.Background(), "13900000031", "Settlement Seller")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}
	accountID, err := sellerSvc.PreCreateAccount(context.Background(), sellerUser.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}
	if _, err := sellerSvc.ApplyDispatchEarning(context.Background(), accountID, 20); err != nil {
		t.Fatalf("apply dispatch earning: %v", err)
	}

	if err := settlementSvc.RequestSettlement(context.Background(), sellerUser.ID); err != nil {
		t.Fatalf("request settlement: %v", err)
	}

	sellerState, err := sellerSvc.GetSeller(context.Background(), sellerUser.ID)
	if err != nil {
		t.Fatalf("get seller: %v", err)
	}
	if sellerState.PendingEarnUSD != 0 {
		t.Fatalf("pending earn should be reserved into settlement: %+v", sellerState)
	}

	settlements, total, err := sellerSvc.ListSettlements(context.Background(), sellerUser.ID, 1, 20)
	if err != nil {
		t.Fatalf("list settlements: %v", err)
	}
	if total != 1 || len(settlements) != 1 {
		t.Fatalf("unexpected settlements: total=%d settlements=%#v", total, settlements)
	}
	if settlements[0].AmountUSD != 15 || settlements[0].Status != "pending" {
		t.Fatalf("unexpected settlement payload: %#v", settlements[0])
	}
}

func TestRequestSettlementRejectsSmallPendingAmount(t *testing.T) {
	sellerSvc := seller.NewService()
	settlementSvc := NewSettlementService(sellerSvc)

	sellerUser, err := sellerSvc.Register(context.Background(), "13900000032", "Settlement Seller Small")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}
	accountID, err := sellerSvc.PreCreateAccount(context.Background(), sellerUser.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}
	if _, err := sellerSvc.ApplyDispatchEarning(context.Background(), accountID, 10); err != nil {
		t.Fatalf("apply dispatch earning: %v", err)
	}

	if err := settlementSvc.RequestSettlement(context.Background(), sellerUser.ID); err == nil {
		t.Fatal("expected settlement request to fail for amount below $10")
	}
}

func TestRunSettlementCycleCreatesSettlementsForAllPendingSellers(t *testing.T) {
	sellerSvc := seller.NewService()
	settlementSvc := NewSettlementService(sellerSvc)

	sellerA, err := sellerSvc.Register(context.Background(), "13900000033", "Seller A")
	if err != nil {
		t.Fatalf("register seller A: %v", err)
	}
	accountA, err := sellerSvc.PreCreateAccount(context.Background(), sellerA.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account A: %v", err)
	}
	if _, err := sellerSvc.ApplyDispatchEarning(context.Background(), accountA, 20); err != nil {
		t.Fatalf("apply dispatch earning A: %v", err)
	}

	sellerB, err := sellerSvc.Register(context.Background(), "13900000034", "Seller B")
	if err != nil {
		t.Fatalf("register seller B: %v", err)
	}
	accountB, err := sellerSvc.PreCreateAccount(context.Background(), sellerB.ID, "openai", 100, 0.8, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account B: %v", err)
	}
	if _, err := sellerSvc.ApplyDispatchEarning(context.Background(), accountB, 25); err != nil {
		t.Fatalf("apply dispatch earning B: %v", err)
	}

	if err := settlementSvc.RunSettlementCycle(context.Background()); err != nil {
		t.Fatalf("run settlement cycle: %v", err)
	}

	settlementsA, totalA, err := sellerSvc.ListSettlements(context.Background(), sellerA.ID, 1, 20)
	if err != nil {
		t.Fatalf("list settlements A: %v", err)
	}
	settlementsB, totalB, err := sellerSvc.ListSettlements(context.Background(), sellerB.ID, 1, 20)
	if err != nil {
		t.Fatalf("list settlements B: %v", err)
	}
	if totalA != 1 || totalB != 1 || len(settlementsA) != 1 || len(settlementsB) != 1 {
		t.Fatalf("unexpected settlement totals: A=%d B=%d", totalA, totalB)
	}
}
