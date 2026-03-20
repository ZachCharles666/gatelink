package syncer

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/yourname/gatelink-engine/internal/db"
	"github.com/yourname/gatelink-engine/internal/health"
)

const (
	// 对账差异阈值：超过 5% 触发健康扣分
	diffThresholdPct = 0.05
	// 同时并发同步的账号数上限
	maxConcurrency = 5
)

// Syncer 定时对账同步器
type Syncer struct {
	adapters map[string]ConsoleAdapter
	db       *db.Pool
	rdb      *redis.Client
	scorer   *health.Scorer
}

// New 创建同步器
func New(db *db.Pool, rdb *redis.Client, scorer *health.Scorer, adapters ...ConsoleAdapter) *Syncer {
	s := &Syncer{
		adapters: make(map[string]ConsoleAdapter),
		db:       db,
		rdb:      rdb,
		scorer:   scorer,
	}
	for _, a := range adapters {
		s.adapters[a.Vendor()] = a
	}
	return s
}

// RunDaily 执行一次全量对账（应由定时任务在每日凌晨 2 点调用）
func (s *Syncer) RunDaily(ctx context.Context) {
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	log.Info().Str("date", yesterday.Format("2006-01-02")).Msg("daily sync started")

	// 查询所有活跃账号
	rows, err := s.db.Query(ctx, `
		SELECT id, vendor, api_key_encrypted
		FROM accounts
		WHERE status = 'active'`)
	if err != nil {
		log.Error().Err(err).Msg("sync: query accounts failed")
		return
	}
	defer rows.Close()

	type accountRow struct {
		ID           string
		Vendor       string
		EncryptedKey string
	}
	var accounts []accountRow
	for rows.Next() {
		var r accountRow
		if err := rows.Scan(&r.ID, &r.Vendor, &r.EncryptedKey); err != nil {
			continue
		}
		accounts = append(accounts, r)
	}

	// 并发同步，限制并发数
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	for _, acct := range accounts {
		wg.Add(1)
		sem <- struct{}{}
		go func(a accountRow) {
			defer wg.Done()
			defer func() { <-sem }()
			s.syncAccount(ctx, a.ID, a.Vendor, a.EncryptedKey, yesterday)
		}(acct)
	}
	wg.Wait()

	log.Info().Int("accounts", len(accounts)).Msg("daily sync completed")
}

// syncAccount 同步单个账号的对账数据
func (s *Syncer) syncAccount(ctx context.Context, accountID, vendor, encryptedKey string, date time.Time) {
	adapter, ok := s.adapters[vendor]
	if !ok {
		log.Debug().Str("vendor", vendor).Msg("no console adapter, skipping")
		return
	}

	// 注意：这里需要解密 key，但 Syncer 当前不持有 keystore
	// MVP 阶段：apiKey 通过环境变量注入（console key 可能与 chat key 不同）
	// TODO: 接入 keystore.Decrypt(encryptedKey) 后删除此占位
	apiKey := encryptedKey // placeholder — 实际需解密

	// 从厂商控制台拉取用量
	consoleUsage, err := adapter.FetchDailyUsage(ctx, apiKey, date)
	if err != nil {
		log.Warn().Err(err).Str("account", accountID).Str("vendor", vendor).Msg("fetch console usage failed")
		return
	}

	// 从本地 usage_records 查询同日消耗
	var localCost float64
	err = s.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM usage_records
		WHERE account_id = $1
		  AND created_at >= $2
		  AND created_at < $3`,
		accountID, date, date.Add(24*time.Hour),
	).Scan(&localCost)
	if err != nil {
		log.Warn().Err(err).Str("account", accountID).Msg("query local usage failed")
		return
	}

	// 计算差异
	diff := math.Abs(consoleUsage.TotalCost - localCost)
	var diffPct float64
	if consoleUsage.TotalCost > 0 {
		diffPct = diff / consoleUsage.TotalCost
	}

	// 写入对账记录（如有 console_usage_records 表则写入，MVP 暂时只记日志）
	log.Info().
		Str("account", accountID).
		Str("vendor", vendor).
		Str("date", date.Format("2006-01-02")).
		Float64("console_cost", consoleUsage.TotalCost).
		Float64("local_cost", localCost).
		Float64("diff_pct", diffPct).
		Msg("reconciliation result")

	// 差异 > 5%：触发健康度处理
	if diffPct > diffThresholdPct {
		detail := map[string]interface{}{
			"console_cost": fmt.Sprintf("%.6f", consoleUsage.TotalCost),
			"local_cost":   fmt.Sprintf("%.6f", localCost),
			"diff_pct":     fmt.Sprintf("%.4f", diffPct),
		}
		if err := s.scorer.Record(ctx, accountID, health.EventReconcileFail, detail); err != nil {
			log.Warn().Err(err).Str("account", accountID).Msg("record reconcile fail event failed")
		}
	} else {
		if err := s.scorer.Record(ctx, accountID, health.EventReconcilePass, nil); err != nil {
			log.Warn().Err(err).Str("account", accountID).Msg("record reconcile pass event failed")
		}
	}
}

// StartSchedule 启动定时任务（每日 UTC 02:00 执行）
// 在 goroutine 中运行，直到 ctx 取消
func (s *Syncer) StartSchedule(ctx context.Context) {
	go func() {
		for {
			now := time.Now().UTC()
			// 计算到下一个 02:00 的等待时间
			next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, time.UTC)
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			wait := next.Sub(now)

			log.Info().Str("next_run", next.Format(time.RFC3339)).Msg("sync scheduled")

			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
				s.RunDaily(ctx)
			}
		}
	}()
}
