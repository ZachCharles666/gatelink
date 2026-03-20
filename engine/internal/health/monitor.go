package health

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/yourname/gatelink-engine/internal/db"
)

const (
	// 下线阈值：health_score 低于此值自动挂起
	suspendThreshold = 20
	// 连续失败阈值：N 次连续失败触发额外扣分
	consecutiveFailThreshold = 3
	// 稳定性检测间隔
	stable24hInterval = 24 * time.Hour
)

// Monitor 负责阈值检测、自动下线和定期稳定性加分
type Monitor struct {
	scorer *Scorer
	db     *db.Pool
	rdb    *redis.Client
}

func NewMonitor(scorer *Scorer, db *db.Pool, rdb *redis.Client) *Monitor {
	return &Monitor{scorer: scorer, db: db, rdb: rdb}
}

// OnRequestResult 在每次请求结束后调用，更新健康度并检查阈值
func (m *Monitor) OnRequestResult(ctx context.Context, accountID string, statusCode int) {
	event := HTTPStatusToEvent(statusCode)

	// 检查连续失败
	if isFailure(statusCode) {
		m.handleConsecutiveFailure(ctx, accountID, event)
	} else {
		// 成功：重置连续失败计数
		m.rdb.Del(ctx, failKey(accountID))
	}

	if err := m.scorer.Record(ctx, accountID, event, map[string]interface{}{
		"status_code": statusCode,
	}); err != nil {
		log.Warn().Err(err).Str("account", accountID).Msg("record health event failed")
	}

	// 检查是否需要自动下线
	m.checkThreshold(ctx, accountID)
}

// handleConsecutiveFailure 追踪连续失败计数，达阈值时额外扣分
func (m *Monitor) handleConsecutiveFailure(ctx context.Context, accountID string, event EventType) {
	key := failKey(accountID)
	count, _ := m.rdb.Incr(ctx, key).Result()
	m.rdb.Expire(ctx, key, 10*time.Minute) // 10 分钟内无请求则重置

	if count >= consecutiveFailThreshold {
		m.rdb.Del(ctx, key) // 重置计数，下一轮重新累积
		if err := m.scorer.Record(ctx, accountID, EventConsecutiveFail, map[string]interface{}{
			"consecutive_count": int(count),
		}); err != nil {
			log.Warn().Err(err).Msg("record consecutive fail event failed")
		}
	}
}

// checkThreshold 检查健康分是否低于阈值，低于则自动挂起
func (m *Monitor) checkThreshold(ctx context.Context, accountID string) {
	var score int
	err := m.db.QueryRow(ctx,
		"SELECT health_score FROM accounts WHERE id = $1", accountID,
	).Scan(&score)
	if err != nil {
		return
	}

	if score < suspendThreshold {
		_, err := m.db.Exec(ctx, `
			UPDATE accounts SET status = 'suspended', updated_at = NOW()
			WHERE id = $1 AND status = 'active'`, accountID)
		if err != nil {
			log.Error().Err(err).Str("account", accountID).Msg("auto-suspend failed")
			return
		}
		// 更新 Redis 状态
		m.rdb.HSet(ctx, "acct:"+accountID, "status", "suspended")
		log.Warn().
			Str("account", accountID).
			Int("health", score).
			Msg("account auto-suspended due to low health score")
	}
}

// RunStable24hCheck 扫描所有连续 24 小时内无故障的账号，给予稳定性加分
// 此方法应以定时任务方式每 24 小时调用一次
func (m *Monitor) RunStable24hCheck(ctx context.Context) {
	// 查找 24 小时内无故障事件的活跃账号
	rows, err := m.db.Query(ctx, `
		SELECT a.id
		FROM accounts a
		WHERE a.status = 'active'
		  AND NOT EXISTS (
		    SELECT 1 FROM health_events h
		    WHERE h.account_id = a.id
		      AND h.score_delta < 0
		      AND h.created_at > NOW() - INTERVAL '24 hours'
		  )`)
	if err != nil {
		log.Error().Err(err).Msg("stable24h check query failed")
		return
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var accountID string
		if err := rows.Scan(&accountID); err != nil {
			continue
		}
		if err := m.scorer.Record(ctx, accountID, EventStable24h, nil); err != nil {
			log.Warn().Err(err).Str("account", accountID).Msg("stable24h score failed")
		}
		count++
	}
	log.Info().Int("accounts", count).Msg("stable24h bonus applied")
}

// failKey Redis key 用于追踪连续失败次数
func failKey(accountID string) string {
	return fmt.Sprintf("fail_streak:%s", accountID)
}

// isFailure 判断 HTTP 状态码是否为失败
func isFailure(statusCode int) bool {
	return statusCode >= 400
}
