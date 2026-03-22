package accounting

import (
	"context"
	"fmt"
	"time"

	"github.com/ZachCharles666/gatelink/api/internal/seller"
)

type settlementLedger interface {
	GetSeller(ctx context.Context, sellerID string) (*seller.Seller, error)
	CreateSettlement(ctx context.Context, sellerID string, amountUSD float64, periodStart, periodEnd string) (*seller.Settlement, error)
	ListSellersWithPending(ctx context.Context) ([]seller.Seller, error)
}

type SettlementService struct {
	sellers settlementLedger
}

func NewSettlementService(sellers settlementLedger) *SettlementService {
	return &SettlementService{sellers: sellers}
}

func (s *SettlementService) RunSettlementCycle(ctx context.Context) error {
	sellers, err := s.sellers.ListSellersWithPending(ctx)
	if err != nil {
		return err
	}

	periodEnd := time.Now().UTC()
	periodStart := periodEnd.AddDate(0, 0, -14).Format(time.RFC3339)
	periodEndStr := periodEnd.Format(time.RFC3339)

	for _, item := range sellers {
		if item.PendingEarnUSD <= 0 {
			continue
		}
		if _, err := s.sellers.CreateSettlement(ctx, item.ID, item.PendingEarnUSD, periodStart, periodEndStr); err != nil {
			return err
		}
	}

	return nil
}

func (s *SettlementService) RequestSettlement(ctx context.Context, sellerID string) error {
	sellerState, err := s.sellers.GetSeller(ctx, sellerID)
	if err != nil {
		return err
	}
	if sellerState.PendingEarnUSD < 10 {
		return fmt.Errorf("minimum settlement amount is $10, current: $%.2f", sellerState.PendingEarnUSD)
	}

	periodEnd := time.Now().UTC()
	periodStart := periodEnd.AddDate(0, 0, -14).Format(time.RFC3339)
	periodEndStr := periodEnd.Format(time.RFC3339)

	_, err = s.sellers.CreateSettlement(ctx, sellerID, sellerState.PendingEarnUSD, periodStart, periodEndStr)
	return err
}
