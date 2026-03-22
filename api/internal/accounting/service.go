package accounting

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/ZachCharles666/gatelink/api/internal/buyer"
	"github.com/ZachCharles666/gatelink/api/internal/engine"
	"github.com/ZachCharles666/gatelink/api/internal/seller"
)

type buyerLedger interface {
	ApplyDispatchCharge(ctx context.Context, buyerID string, chargedUSD float64, usage buyer.UsageRecord) error
}

type sellerLedger interface {
	GetAccount(ctx context.Context, accountID string) (*seller.Account, error)
	ApplyDispatchEarning(ctx context.Context, accountID string, consumedCreditsUSD float64) (float64, error)
}

type Service struct {
	mu      sync.RWMutex
	buyers  buyerLedger
	sellers sellerLedger
	models  []ModelInfo
}

type ModelInfo struct {
	Vendor           string  `json:"vendor"`
	Model            string  `json:"model"`
	InputPricePer1M  float64 `json:"input_price_per_1m"`
	OutputPricePer1M float64 `json:"output_price_per_1m"`
	PlatformDiscount float64 `json:"platform_discount"`
}

func NewService(buyers buyerLedger, sellers sellerLedger) *Service {
	return &Service{
		buyers:  buyers,
		sellers: sellers,
		models: []ModelInfo{
			{Vendor: "anthropic", Model: "claude-opus-4-6", InputPricePer1M: 15.00, OutputPricePer1M: 75.00, PlatformDiscount: 0.88},
			{Vendor: "anthropic", Model: "claude-sonnet-4-6", InputPricePer1M: 3.00, OutputPricePer1M: 15.00, PlatformDiscount: 0.88},
			{Vendor: "anthropic", Model: "claude-haiku-4-5", InputPricePer1M: 0.80, OutputPricePer1M: 4.00, PlatformDiscount: 0.88},
			{Vendor: "openai", Model: "gpt-4o", InputPricePer1M: 2.50, OutputPricePer1M: 10.00, PlatformDiscount: 0.88},
			{Vendor: "openai", Model: "gpt-4o-mini", InputPricePer1M: 0.15, OutputPricePer1M: 0.60, PlatformDiscount: 0.88},
			{Vendor: "gemini", Model: "gemini-2.5-pro", InputPricePer1M: 1.25, OutputPricePer1M: 10.00, PlatformDiscount: 0.88},
		},
	}
}

func (s *Service) ListModels(_ context.Context) ([]ModelInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	models := make([]ModelInfo, len(s.models))
	copy(models, s.models)
	sort.Slice(models, func(i, j int) bool {
		if models[i].Vendor == models[j].Vendor {
			return models[i].Model < models[j].Model
		}
		return models[i].Vendor < models[j].Vendor
	})

	return models, nil
}

func (s *Service) ChargeAfterDispatch(ctx context.Context, buyerID string, result *engine.DispatchResult) error {
	if buyerID == "" {
		return errors.New("buyer_id is required")
	}
	if result == nil {
		return errors.New("dispatch result is required")
	}

	account, err := s.sellers.GetAccount(ctx, result.AccountID)
	if err != nil {
		return err
	}

	platformDiscount := s.platformDiscountForVendor(account.Vendor)
	buyerChargedUSD := result.CostUSD * platformDiscount

	usage := buyer.UsageRecord{
		Vendor:          result.Vendor,
		Model:           s.defaultModelForVendor(result.Vendor),
		InputTokens:     result.InputTokens,
		OutputTokens:    result.OutputTokens,
		CostUSD:         result.CostUSD,
		BuyerChargedUSD: buyerChargedUSD,
	}
	if err := s.buyers.ApplyDispatchCharge(ctx, buyerID, buyerChargedUSD, usage); err != nil {
		return err
	}

	_, err = s.sellers.ApplyDispatchEarning(ctx, result.AccountID, result.CostUSD)
	return err
}

func (s *Service) ChargeAfterStream(ctx context.Context, buyerID, accountID, vendor, model string, inputTokens, outputTokens int) error {
	if accountID == "" {
		return errors.New("account_id is required")
	}

	modelInfo, ok := s.lookupModel(vendor, model)
	if !ok {
		return errors.New("model pricing not found")
	}

	costUSD := float64(inputTokens)/1e6*modelInfo.InputPricePer1M + float64(outputTokens)/1e6*modelInfo.OutputPricePer1M
	return s.ChargeAfterDispatch(ctx, buyerID, &engine.DispatchResult{
		AccountID:    accountID,
		Vendor:       vendor,
		CostUSD:      costUSD,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
	})
}

func (s *Service) platformDiscountForVendor(vendor string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, model := range s.models {
		if model.Vendor == vendor && model.PlatformDiscount > 0 {
			return model.PlatformDiscount
		}
	}
	return 0.88
}

func (s *Service) defaultModelForVendor(vendor string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, model := range s.models {
		if model.Vendor == vendor {
			return model.Model
		}
	}
	return ""
}

func (s *Service) lookupModel(vendor, model string) (ModelInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.models {
		if item.Vendor == vendor && item.Model == model {
			return item, true
		}
	}
	for _, item := range s.models {
		if item.Vendor == vendor {
			return item, true
		}
	}

	return ModelInfo{}, false
}
