package poller

import (
	"context"
	"testing"
	"time"

	"github.com/ZachCharles666/gatelink/api/internal/seller"
)

func TestAccountPollerDetectsTrackedStatusChange(t *testing.T) {
	sellerSvc := seller.NewService()
	sellerUser, err := sellerSvc.Register(context.Background(), "13900000041", "Poller Seller")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}
	accountID, err := sellerSvc.PreCreateAccount(context.Background(), sellerUser.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("pre create account: %v", err)
	}

	changeCh := make(chan struct {
		id     string
		status string
	}, 1)

	p := NewAccountPoller(sellerSvc, func(accountID, status string) {
		changeCh <- struct {
			id     string
			status string
		}{id: accountID, status: status}
	})
	p.interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Start(ctx)

	time.Sleep(30 * time.Millisecond)
	if _, err := sellerSvc.ForceSuspendAccount(context.Background(), accountID); err != nil {
		t.Fatalf("force suspend account: %v", err)
	}

	select {
	case change := <-changeCh:
		if change.id != accountID || change.status != "suspended" {
			t.Fatalf("unexpected change payload: %#v", change)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected poller to detect suspended status change")
	}
}

func TestAccountPollerIgnoresInitialSnapshot(t *testing.T) {
	sellerSvc := seller.NewService()
	sellerUser, err := sellerSvc.Register(context.Background(), "13900000042", "Poller Seller Initial")
	if err != nil {
		t.Fatalf("register seller: %v", err)
	}
	if _, err := sellerSvc.PreCreateAccount(context.Background(), sellerUser.ID, "anthropic", 100, 0.75, 100, "2026-09-01T00:00:00Z"); err != nil {
		t.Fatalf("pre create account: %v", err)
	}

	changeCh := make(chan string, 1)
	p := NewAccountPoller(sellerSvc, func(_, status string) {
		changeCh <- status
	})
	p.interval = 10 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go p.Start(ctx)

	select {
	case status := <-changeCh:
		t.Fatalf("unexpected initial status callback: %s", status)
	case <-time.After(50 * time.Millisecond):
	}
}
