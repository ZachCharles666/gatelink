// Package syncer 提供 Console 用量同步功能。
// 每日凌晨 2 点从各厂商控制台 API 拉取实际消耗，与本地 usage_records 对比，
// 差异 > 5% 时触发健康扣分并发送事件。
package syncer

import (
	"context"
	"time"
)

// DailyUsage 厂商控制台返回的单日用量数据
type DailyUsage struct {
	Date       time.Time // 日期（UTC 00:00:00）
	TotalCost  float64   // 当日消耗金额（美元）
	InputTokens  int     // 输入 tokens（如 API 支持）
	OutputTokens int     // 输出 tokens（如 API 支持）
}

// ConsoleAdapter 各厂商控制台 API 适配器接口
type ConsoleAdapter interface {
	// Vendor 返回厂商标识
	Vendor() string

	// FetchDailyUsage 从控制台 API 拉取指定日期的用量数据
	// apiKey 是控制台访问凭证（可能与 chat API 的 key 不同）
	FetchDailyUsage(ctx context.Context, apiKey string, date time.Time) (*DailyUsage, error)
}
