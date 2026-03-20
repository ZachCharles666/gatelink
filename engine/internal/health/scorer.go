// Package health 管理账号健康度评分系统。
// 评分范围：0–100，初始值 80，由事件驱动更新。
// 事件分值会同时更新 PostgreSQL（accounts.health_score）和 Redis 账号快照（acct:{id}.health）。
package health

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/yourname/gatelink-engine/internal/db"
)

// EventType 健康度事件类型
type EventType string

const (
	// 请求结果事件
	EventSuccess          EventType = "success"           // 请求成功            +0.5
	EventClient4xx        EventType = "client_4xx"        // 客户端错误(非401/3) -5
	EventServer5xx        EventType = "server_5xx"        // 服务器错误           -2
	EventRateLimited      EventType = "rate_limited"      // 429 频率限制         -15
	EventConsecutiveFail  EventType = "consecutive_fail"  // 连续失败             -20
	EventUnauthorized     EventType = "unauthorized"      // 401/403 认证失败     -100
	// 对账事件
	EventReconcilePass   EventType = "reconcile_pass"    // 对账通过             +10
	EventReconcileFail   EventType = "reconcile_fail"    // 对账异常             -30
	// 稳定性事件
	EventStable24h       EventType = "stable_24h"        // 24小时稳定           +5
)

// eventDeltas 各事件对应的分值变化（正=加分，负=扣分）
// 注意：success 用浮点 +0.5，但 DB 字段是 SMALLINT，存储时取整或累积
var eventDeltas = map[EventType]int{
	EventSuccess:         0, // 特殊处理：累积计数后批量+0.5
	EventClient4xx:       -5,
	EventServer5xx:       -2,
	EventRateLimited:     -15,
	EventConsecutiveFail: -20,
	EventUnauthorized:    -100,
	EventReconcilePass:   +10,
	EventReconcileFail:   -30,
	EventStable24h:       +5,
}

// Scorer 负责健康度分值计算和更新
type Scorer struct {
	db  *db.Pool
	rdb *redis.Client
}

func NewScorer(db *db.Pool, rdb *redis.Client) *Scorer {
	return &Scorer{db: db, rdb: rdb}
}

// Record 记录一次健康度事件，更新账号分值
// detail 可为 nil，用于附加 JSON 信息（如 HTTP status 码、错误原因等）
func (s *Scorer) Record(ctx context.Context, accountID string, event EventType, detail map[string]interface{}) error {
	delta := eventDeltas[event]

	// 特殊事件：success 每 2 次成功才加 1 分（实现 +0.5）
	if event == EventSuccess {
		delta = s.computeSuccessDelta(ctx, accountID)
	}

	if delta == 0 && event == EventSuccess {
		// 未累积到 +1 分，只记录成功计数，不写 DB
		return nil
	}

	// 从 DB 读取当前分值
	var currentScore int
	err := s.db.QueryRow(ctx,
		"SELECT health_score FROM accounts WHERE id = $1", accountID,
	).Scan(&currentScore)
	if err != nil {
		return fmt.Errorf("get health score: %w", err)
	}

	// 计算新分值（限定在 0–100）
	newScore := clamp(currentScore+delta, 0, 100)

	// 事务：更新 accounts + 插入 health_events
	_, err = s.db.Exec(ctx, `
		WITH updated AS (
			UPDATE accounts SET health_score = $1, updated_at = NOW()
			WHERE id = $2
			RETURNING id
		)
		INSERT INTO health_events (account_id, event_type, score_delta, score_after, detail)
		SELECT id, $3, $4, $1, $5::jsonb
		FROM updated`,
		newScore, accountID, string(event), delta, marshalDetail(detail),
	)
	if err != nil {
		return fmt.Errorf("update health score: %w", err)
	}

	// 同步更新 Redis 账号快照
	s.rdb.HSet(ctx, "acct:"+accountID, "health", strconv.Itoa(newScore))

	log.Info().
		Str("account", accountID).
		Str("event", string(event)).
		Int("delta", delta).
		Int("score", newScore).
		Msg("health score updated")

	return nil
}

// computeSuccessDelta 管理成功计数，每 2 次成功返回 +1，其余返回 0
// 使用 Redis incr 做原子计数
func (s *Scorer) computeSuccessDelta(ctx context.Context, accountID string) int {
	key := "health_success_cnt:" + accountID
	count, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return 0
	}
	if count%2 == 0 {
		return 1 // 每 2 次成功加 1 分，等效 +0.5/次
	}
	return 0
}

// clamp 将值限定在 [min, max] 范围内
func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// marshalDetail 将 map 转为 JSON 字符串供数据库存储
func marshalDetail(detail map[string]interface{}) string {
	if detail == nil {
		return "null"
	}
	var buf []byte
	buf = append(buf, '{')
	first := true
	for k, v := range detail {
		if !first {
			buf = append(buf, ',')
		}
		first = false
		buf = append(buf, '"')
		buf = append(buf, k...)
		buf = append(buf, '"', ':')
		switch val := v.(type) {
		case string:
			buf = append(buf, '"')
			buf = append(buf, val...)
			buf = append(buf, '"')
		case int:
			buf = append(buf, strconv.Itoa(val)...)
		case float64:
			buf = append(buf, strconv.FormatFloat(val, 'f', 4, 64)...)
		case bool:
			if val {
				buf = append(buf, "true"...)
			} else {
				buf = append(buf, "false"...)
			}
		default:
			buf = append(buf, '"', '"')
		}
	}
	buf = append(buf, '}')
	return string(buf)
}

// HTTPStatusToEvent 将 HTTP 状态码转换为对应的健康度事件
func HTTPStatusToEvent(statusCode int) EventType {
	switch {
	case statusCode == 401 || statusCode == 403:
		return EventUnauthorized
	case statusCode == 429:
		return EventRateLimited
	case statusCode >= 400 && statusCode < 500:
		return EventClient4xx
	case statusCode >= 500:
		return EventServer5xx
	case statusCode >= 200 && statusCode < 300:
		return EventSuccess
	default:
		return EventServer5xx
	}
}
