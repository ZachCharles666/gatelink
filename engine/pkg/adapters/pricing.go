package vendor

import (
	"context"
	"fmt"

	"github.com/yourname/tokenglide-engine/internal/db"
)

// PricingDB 从数据库查询模型定价
type PricingDB struct {
	pool *db.Pool
}

func NewPricingDB(pool *db.Pool) *PricingDB {
	return &PricingDB{pool: pool}
}

// ModelPricing 模型定价信息
type ModelPricing struct {
	Vendor           string
	Model            string
	InputPricePer1M  float64
	OutputPricePer1M float64
	PlatformDiscount float64
}

func (p *PricingDB) Get(ctx context.Context, vendor, model string) (*ModelPricing, error) {
	var pricing ModelPricing
	err := p.pool.QueryRow(ctx,
		`SELECT vendor, model, input_price_per_1m, output_price_per_1m, platform_discount
		 FROM vendor_pricing
		 WHERE vendor = $1 AND model = $2 AND is_active = true`,
		vendor, model,
	).Scan(
		&pricing.Vendor,
		&pricing.Model,
		&pricing.InputPricePer1M,
		&pricing.OutputPricePer1M,
		&pricing.PlatformDiscount,
	)
	if err != nil {
		return nil, fmt.Errorf("get pricing for %s/%s: %w", vendor, model, err)
	}
	return &pricing, nil
}

func (p *PricingDB) CalcCostFromDB(ctx context.Context, vendor, model string, usage *Usage) (*Cost, error) {
	pricing, err := p.Get(ctx, vendor, model)
	if err != nil {
		return nil, err
	}
	inputCost := float64(usage.PromptTokens) / 1_000_000 * pricing.InputPricePer1M
	outputCost := float64(usage.CompletionTokens) / 1_000_000 * pricing.OutputPricePer1M
	return &Cost{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		CostUSD:      inputCost + outputCost,
	}, nil
}
