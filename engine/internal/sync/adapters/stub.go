package adapters

// stub.go 为尚未接入控制台 API 的厂商提供占位实现。
// Gemini、Qwen、GLM、Kimi 的控制台 API 各不相同，MVP 阶段先以 stub 占位。

import (
	"context"
	"fmt"
	"time"

	syncer "github.com/yourname/gatelink-engine/internal/sync"
)

// StubConsole 用于尚未实现的厂商控制台适配器（返回空数据）
type StubConsole struct {
	vendor string
}

func NewStub(vendor string) *StubConsole {
	return &StubConsole{vendor: vendor}
}

func (s *StubConsole) Vendor() string { return s.vendor }

func (s *StubConsole) FetchDailyUsage(ctx context.Context, apiKey string, date time.Time) (*syncer.DailyUsage, error) {
	return nil, fmt.Errorf("%s console sync not implemented in MVP", s.vendor)
}
