// Package scheduler 管理账号池和调度逻辑。
// Redis 数据结构：
//   - ZSET  "pool:{vendor}" — 成员=accountID，score=综合评分（越高越优先）
//   - HASH  "acct:{id}"    — AccountInfo 的各字段
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	poolKeyPrefix = "pool:"
	acctKeyPrefix = "acct:"

	// 字段名（HASH key）
	fieldVendor        = "vendor"
	fieldModel         = "model"
	fieldHealth        = "health"
	fieldBalanceUSD    = "balance_usd"
	fieldRPMLimit      = "rpm_limit"
	fieldRPMUsed       = "rpm_used"
	fieldDailyLimit    = "daily_limit"
	fieldDailyUsed     = "daily_used"
	fieldPeakDaily     = "peak_daily"
	fieldStatus        = "status"
	fieldEncryptedKey  = "encrypted_key"
	fieldSellerID      = "seller_id"
	fieldLastUpdated   = "last_updated"
)

// AccountInfo 代表账号池中一个账号的运行时快照。
// 所有数值来自 Redis HASH，不直接读数据库，保证调度路径低延迟。
type AccountInfo struct {
	ID           string  `json:"id"`
	SellerID     string  `json:"seller_id"`
	Vendor       string  `json:"vendor"`
	Model        string  `json:"model"`
	Health       float64 `json:"health"`        // 0–100
	BalanceUSD   float64 `json:"balance_usd"`   // 剩余余额（美元）
	RPMLimit     int     `json:"rpm_limit"`     // 每分钟请求上限
	RPMUsed      int     `json:"rpm_used"`      // 当前分钟已用
	DailyLimit   float64 `json:"daily_limit"`   // 日消耗上限（美元），0=不限
	DailyUsed    float64 `json:"daily_used"`    // 当日已消耗（美元）
	PeakDaily    float64 `json:"peak_daily"`    // 历史峰值日消耗（美元）
	Status       string  `json:"status"`        // "active"|"suspended"|"exhausted"
	EncryptedKey string  `json:"encrypted_key"` // AES-GCM 密文，只在转发时解密
	Score        float64 `json:"score,omitempty"`
}

// IsHardExcluded 判断账号是否命中硬性排除条件，命中则不参与本次调度
func (a *AccountInfo) IsHardExcluded() (bool, string) {
	if a.Status != "active" {
		return true, "status=" + a.Status
	}
	if a.Health < 30 {
		return true, fmt.Sprintf("health=%.1f<30", a.Health)
	}
	if a.BalanceUSD < 1.0 {
		return true, fmt.Sprintf("balance=%.4f<1", a.BalanceUSD)
	}
	if a.RPMLimit > 0 && float64(a.RPMUsed) >= float64(a.RPMLimit)*0.80 {
		return true, fmt.Sprintf("rpm=%d/%d≥80%%", a.RPMUsed, a.RPMLimit)
	}
	if a.PeakDaily > 0 && a.DailyUsed >= a.PeakDaily*1.50 {
		return true, fmt.Sprintf("daily=%.4f≥peak*150%%", a.DailyUsed)
	}
	return false, ""
}

// Pool 账号池：封装 Redis ZSET + HASH 操作
type Pool struct {
	rdb *redis.Client
}

func NewPool(rdb *redis.Client) *Pool {
	return &Pool{rdb: rdb}
}

// poolKey 返回指定厂商的 ZSET key
func poolKey(vendor string) string {
	return poolKeyPrefix + vendor
}

// acctKey 返回账号 HASH key
func acctKey(id string) string {
	return acctKeyPrefix + id
}

// Upsert 将账号信息写入 Redis（HASH 存字段，ZSET 存评分）
func (p *Pool) Upsert(ctx context.Context, info *AccountInfo) error {
	key := acctKey(info.ID)
	fields := map[string]interface{}{
		fieldVendor:       info.Vendor,
		fieldModel:        info.Model,
		fieldHealth:       strconv.FormatFloat(info.Health, 'f', 4, 64),
		fieldBalanceUSD:   strconv.FormatFloat(info.BalanceUSD, 'f', 6, 64),
		fieldRPMLimit:     strconv.Itoa(info.RPMLimit),
		fieldRPMUsed:      strconv.Itoa(info.RPMUsed),
		fieldDailyLimit:   strconv.FormatFloat(info.DailyLimit, 'f', 6, 64),
		fieldDailyUsed:    strconv.FormatFloat(info.DailyUsed, 'f', 6, 64),
		fieldPeakDaily:    strconv.FormatFloat(info.PeakDaily, 'f', 6, 64),
		fieldStatus:       info.Status,
		fieldEncryptedKey: info.EncryptedKey,
		fieldSellerID:     info.SellerID,
		fieldLastUpdated:  strconv.FormatInt(time.Now().Unix(), 10),
	}

	pipe := p.rdb.Pipeline()
	pipe.HSet(ctx, key, fields)
	pipe.ZAdd(ctx, poolKey(info.Vendor), redis.Z{
		Score:  info.Score,
		Member: info.ID,
	})
	_, err := pipe.Exec(ctx)
	return err
}

// Get 读取单个账号快照
func (p *Pool) Get(ctx context.Context, id string) (*AccountInfo, error) {
	vals, err := p.rdb.HGetAll(ctx, acctKey(id)).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall %s: %w", id, err)
	}
	if len(vals) == 0 {
		return nil, fmt.Errorf("account %s not found in pool", id)
	}
	return parseAccountInfo(id, vals)
}

// ListVendor 返回指定厂商的所有账号快照（按评分降序）
func (p *Pool) ListVendor(ctx context.Context, vendor string) ([]*AccountInfo, error) {
	members, err := p.rdb.ZRevRangeWithScores(ctx, poolKey(vendor), 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("zrevrange %s: %w", vendor, err)
	}

	infos := make([]*AccountInfo, 0, len(members))
	for _, z := range members {
		id := fmt.Sprintf("%v", z.Member)
		info, err := p.Get(ctx, id)
		if err != nil {
			continue // 跳过读取失败的账号
		}
		info.Score = z.Score
		infos = append(infos, info)
	}
	return infos, nil
}

// UpdateScore 更新 ZSET 中的账号评分
func (p *Pool) UpdateScore(ctx context.Context, vendor, id string, score float64) error {
	return p.rdb.ZAdd(ctx, poolKey(vendor), redis.Z{
		Score:  score,
		Member: id,
	}).Err()
}

// IncrRPMUsed 原子递增当前分钟 RPM 计数（每分钟自动过期）
func (p *Pool) IncrRPMUsed(ctx context.Context, id string) error {
	counterKey := fmt.Sprintf("rpm:%s:%d", id, time.Now().Unix()/60)
	pipe := p.rdb.Pipeline()
	pipe.Incr(ctx, counterKey)
	pipe.Expire(ctx, counterKey, 2*time.Minute)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return err
	}
	// 回写 HASH 中的 rpm_used（近似值，允许轻微不一致）
	count, _ := p.rdb.Get(ctx, counterKey).Int()
	return p.rdb.HSet(ctx, acctKey(id), fieldRPMUsed, count).Err()
}

// IncrDailyUsed 原子递增当日消耗金额
func (p *Pool) IncrDailyUsed(ctx context.Context, id string, costUSD float64) error {
	key := acctKey(id)
	cur, _ := p.rdb.HGet(ctx, key, fieldDailyUsed).Float64()
	newVal := cur + costUSD
	return p.rdb.HSet(ctx, key, fieldDailyUsed, strconv.FormatFloat(newVal, 'f', 6, 64)).Err()
}

// Remove 从池中移除账号
func (p *Pool) Remove(ctx context.Context, vendor, id string) error {
	pipe := p.rdb.Pipeline()
	pipe.ZRem(ctx, poolKey(vendor), id)
	pipe.Del(ctx, acctKey(id))
	_, err := pipe.Exec(ctx)
	return err
}

// Stats 返回各厂商池的账号数量
func (p *Pool) Stats(ctx context.Context) (map[string]int64, error) {
	vendors := []string{"anthropic", "openai", "gemini", "qwen", "glm", "kimi"}
	result := make(map[string]int64, len(vendors))
	for _, v := range vendors {
		n, err := p.rdb.ZCard(ctx, poolKey(v)).Result()
		if err != nil {
			continue
		}
		result[v] = n
	}
	return result, nil
}

// parseAccountInfo 将 Redis HASH map 解析为 AccountInfo
func parseAccountInfo(id string, vals map[string]string) (*AccountInfo, error) {
	info := &AccountInfo{ID: id}

	info.Vendor = vals[fieldVendor]
	info.Model = vals[fieldModel]
	info.Status = vals[fieldStatus]
	info.EncryptedKey = vals[fieldEncryptedKey]
	info.SellerID = vals[fieldSellerID]

	if v, err := strconv.ParseFloat(vals[fieldHealth], 64); err == nil {
		info.Health = v
	}
	if v, err := strconv.ParseFloat(vals[fieldBalanceUSD], 64); err == nil {
		info.BalanceUSD = v
	}
	if v, err := strconv.Atoi(vals[fieldRPMLimit]); err == nil {
		info.RPMLimit = v
	}
	if v, err := strconv.Atoi(vals[fieldRPMUsed]); err == nil {
		info.RPMUsed = v
	}
	if v, err := strconv.ParseFloat(vals[fieldDailyLimit], 64); err == nil {
		info.DailyLimit = v
	}
	if v, err := strconv.ParseFloat(vals[fieldDailyUsed], 64); err == nil {
		info.DailyUsed = v
	}
	if v, err := strconv.ParseFloat(vals[fieldPeakDaily], 64); err == nil {
		info.PeakDaily = v
	}
	return info, nil
}

// MarshalJSON 辅助：将 AccountInfo 序列化为 JSON（用于日志，不含 EncryptedKey）
func (a *AccountInfo) SafeJSON() string {
	safe := struct {
		ID         string  `json:"id"`
		Vendor     string  `json:"vendor"`
		Model      string  `json:"model"`
		Health     float64 `json:"health"`
		BalanceUSD float64 `json:"balance_usd"`
		Status     string  `json:"status"`
		Score      float64 `json:"score"`
	}{
		ID: a.ID, Vendor: a.Vendor, Model: a.Model,
		Health: a.Health, BalanceUSD: a.BalanceUSD,
		Status: a.Status, Score: a.Score,
	}
	b, _ := json.Marshal(safe)
	return string(b)
}
