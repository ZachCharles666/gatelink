package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/yourname/tokenglide-engine/internal/db"
)

func TestDatabaseConnection(t *testing.T) {
	godotenv.Load("../../.env")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping DB tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	// 验证连接健康
	if err := pool.Healthy(ctx); err != nil {
		t.Fatalf("health check failed: %v", err)
	}

	// 验证所有表存在
	tables := []string{
		"vendor_pricing", "sellers", "buyers",
		"accounts", "usage_records", "health_events", "settlements",
	}

	for _, table := range tables {
		var count int
		err := pool.QueryRow(ctx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name=$1",
			table,
		).Scan(&count)
		if err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if count == 0 {
			t.Errorf("table %s does not exist", table)
		}
	}

	// 验证 vendor_pricing 初始数据
	var pricingCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM vendor_pricing").Scan(&pricingCount)
	if pricingCount < 6 {
		t.Errorf("expected at least 6 pricing rows, got %d", pricingCount)
	}

	t.Logf("DB connection OK, stats: %+v", pool.Stats())
}
