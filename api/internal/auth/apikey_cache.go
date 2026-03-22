package auth

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const buyerAPIKeyCacheTTL = 5 * time.Minute

type APIKeyCacheInvalidator interface {
	InvalidateAPIKey(ctx context.Context, apiKey string)
}

// CachedBuyerRepo wraps BuyerRepo with an optional Redis cache so the
// proxy-facing API key lookup path does not always hit the underlying store.
type CachedBuyerRepo struct {
	underlying BuyerRepo
	rdb        *redis.Client
}

func NewCachedBuyerRepo(underlying BuyerRepo, rdb *redis.Client) *CachedBuyerRepo {
	return &CachedBuyerRepo{
		underlying: underlying,
		rdb:        rdb,
	}
}

func (c *CachedBuyerRepo) FindByAPIKey(ctx context.Context, apiKey string) (*BuyerInfo, error) {
	if c.rdb == nil || apiKey == "" {
		return c.underlying.FindByAPIKey(ctx, apiKey)
	}

	cacheKey := buyerAPIKeyCacheKey(apiKey)
	cached, err := c.rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var info BuyerInfo
		if json.Unmarshal(cached, &info) == nil {
			return &info, nil
		}
	}

	info, err := c.underlying.FindByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(info); err == nil {
		_ = c.rdb.Set(ctx, cacheKey, data, buyerAPIKeyCacheTTL).Err()
	}

	return info, nil
}

func (c *CachedBuyerRepo) InvalidateAPIKey(ctx context.Context, apiKey string) {
	if c.rdb == nil || apiKey == "" {
		return
	}

	_ = c.rdb.Del(ctx, buyerAPIKeyCacheKey(apiKey)).Err()
}

func buyerAPIKeyCacheKey(apiKey string) string {
	if len(apiKey) > 8 {
		apiKey = apiKey[:8]
	}
	return "buyer:apikey:" + apiKey
}
