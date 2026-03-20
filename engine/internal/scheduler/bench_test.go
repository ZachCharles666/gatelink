package scheduler

import (
	"testing"
)

// BenchmarkScore 调度评分算法性能基准
func BenchmarkScore(b *testing.B) {
	acct := &AccountInfo{
		ID:         "bench-acct",
		Status:     "active",
		Health:     80,
		BalanceUSD: 150,
		RPMLimit:   60,
		RPMUsed:    20,
		PeakDaily:  30,
		DailyUsed:  10,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Score(acct)
	}
}

// BenchmarkIsHardExcluded 硬性排除逻辑性能基准
func BenchmarkIsHardExcluded(b *testing.B) {
	acct := &AccountInfo{
		Status:     "active",
		Health:     80,
		BalanceUSD: 150,
		RPMLimit:   60,
		RPMUsed:    20,
		PeakDaily:  30,
		DailyUsed:  10,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = acct.IsHardExcluded()
	}
}
