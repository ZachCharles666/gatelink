package scheduler

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

const (
	// 同一买家连续使用同一账号的最大次数，超出后强制切换
	maxConsecutiveSameAccount = 3

	// 反相关追踪：每个买家最近选择的账号列表（Redis key）
	buyerHistoryKeyPrefix = "buyer_hist:"
)

// DispatchRequest 调度请求
type DispatchRequest struct {
	BuyerID string // 买家 ID（用于反相关检测）
	Vendor  string // 目标厂商
	Model   string // 目标模型（可选，用于过滤）
}

// DispatchResult 调度结果
type DispatchResult struct {
	Account *AccountInfo
}

// Engine 调度引擎主逻辑
type Engine struct {
	pool *Pool
}

func NewEngine(pool *Pool) *Engine {
	return &Engine{pool: pool}
}

// Dispatch 从账号池中为本次请求选择最优账号
//
// 选择流程：
//  1. 从 Redis ZSET 获取该厂商所有账号（按评分降序）
//  2. 应用硬性排除条件
//  3. 应用反相关约束（同一买家连续 ≥3 次同账号则跳过）
//  4. 返回最终选择的账号
func (e *Engine) Dispatch(ctx context.Context, req *DispatchRequest) (*DispatchResult, error) {
	accounts, err := e.pool.ListVendor(ctx, req.Vendor)
	if err != nil {
		return nil, fmt.Errorf("list vendor pool: %w", err)
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts in pool for vendor=%s", req.Vendor)
	}

	// 获取买家最近连续账号历史
	consecutiveID, consecutiveCount := e.getConsecutiveHistory(ctx, req.BuyerID)

	var chosen *AccountInfo
	for _, acct := range accounts {
		// 1. 硬性排除
		if excluded, reason := acct.IsHardExcluded(); excluded {
			log.Debug().
				Str("account", acct.ID).
				Str("reason", reason).
				Msg("account hard excluded")
			continue
		}

		// 2. 反相关约束：同一买家连续 ≥3 次使用同一账号，强制跳过
		if acct.ID == consecutiveID && consecutiveCount >= maxConsecutiveSameAccount {
			log.Debug().
				Str("account", acct.ID).
				Str("buyer", req.BuyerID).
				Int("consecutive", consecutiveCount).
				Msg("anti-correlation: forced skip")
			continue
		}

		// 3. 模型过滤（可选）
		if req.Model != "" && acct.Model != "" && acct.Model != req.Model {
			continue
		}

		chosen = acct
		break // ZSET 已按评分降序排列，第一个满足条件的即为最优
	}

	if chosen == nil {
		return nil, fmt.Errorf("no eligible account for vendor=%s buyer=%s", req.Vendor, req.BuyerID)
	}

	log.Info().
		Str("account", chosen.ID).
		Str("vendor", chosen.Vendor).
		Str("buyer", req.BuyerID).
		Float64("score", chosen.Score).
		Msg("account dispatched")

	return &DispatchResult{Account: chosen}, nil
}

// RecordConsumed 记录一次成功请求的消耗（RPM + 日消耗）并更新评分
func (e *Engine) RecordConsumed(ctx context.Context, accountID, vendor string, costUSD float64) {
	if err := e.pool.IncrRPMUsed(ctx, accountID); err != nil {
		log.Warn().Err(err).Str("account", accountID).Msg("incr rpm failed")
	}
	if err := e.pool.IncrDailyUsed(ctx, accountID, costUSD); err != nil {
		log.Warn().Err(err).Str("account", accountID).Msg("incr daily failed")
	}

	// 重新计算评分并更新 ZSET
	info, err := e.pool.Get(ctx, accountID)
	if err != nil {
		log.Warn().Err(err).Str("account", accountID).Msg("get account for rescore failed")
		return
	}
	newScore := Score(info)
	if err := e.pool.UpdateScore(ctx, vendor, accountID, newScore); err != nil {
		log.Warn().Err(err).Str("account", accountID).Msg("update score failed")
	}
}

// RecordBuyerHistory 更新买家连续使用账号的历史记录
func (e *Engine) RecordBuyerHistory(ctx context.Context, buyerID, accountID string) {
	key := buyerHistoryKeyPrefix + buyerID
	// 存储格式：`accountID:count` 作为简单字符串
	cur, count := e.getConsecutiveHistory(ctx, buyerID)
	if cur == accountID {
		count++
	} else {
		count = 1
	}
	e.pool.rdb.Set(ctx, key, fmt.Sprintf("%s:%d", accountID, count), 0)
}

// getConsecutiveHistory 返回买家最近连续使用的账号 ID 和次数
func (e *Engine) getConsecutiveHistory(ctx context.Context, buyerID string) (string, int) {
	if buyerID == "" {
		return "", 0
	}
	key := buyerHistoryKeyPrefix + buyerID
	val, err := e.pool.rdb.Get(ctx, key).Result()
	if err != nil {
		return "", 0
	}
	var id string
	var count int
	fmt.Sscanf(val, "%s:%d", &id, &count)
	// Sscanf 用空格分割，需要手动解析冒号格式
	for i := len(val) - 1; i >= 0; i-- {
		if val[i] == ':' {
			id = val[:i]
			fmt.Sscanf(val[i+1:], "%d", &count)
			break
		}
	}
	return id, count
}
