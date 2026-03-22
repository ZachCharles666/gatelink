// cmd/seed/main.go — 联调测试账号种子脚本（仅用于开发联调，勿在生产执行）
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/yourname/gatelink-engine/internal/config"
	"github.com/yourname/gatelink-engine/internal/crypto"
	"github.com/yourname/gatelink-engine/internal/db"
	"github.com/yourname/gatelink-engine/internal/scheduler"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://GateLink:password@localhost:5432/GateLink?sslmode=disable"
	}

	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		fmt.Fprintln(os.Stderr, "ENCRYPTION_KEY not set")
		os.Exit(1)
	}

	ctx := context.Background()

	// DB
	dbPool, err := db.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db: %v\n", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Redis
	rdb, err := config.NewRedis(ctx, os.Getenv("REDIS_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "redis: %v\n", err)
		os.Exit(1)
	}

	// Keystore
	ks, err := crypto.NewWithKey(encKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "keystore: %v\n", err)
		os.Exit(1)
	}

	// 1. 插入 seller（如果 phone 已存在则跳过）
	var sellerID string
	err = dbPool.QueryRow(ctx,
		`INSERT INTO sellers (phone, display_name, status)
		 VALUES ('00000000000', 'Dev-A Test Seller', 'active')
		 ON CONFLICT (phone) DO UPDATE SET display_name=EXCLUDED.display_name
		 RETURNING id`,
	).Scan(&sellerID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "insert seller: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("seller_id:", sellerID)

	// 2. 加密一个测试 API key（仅用于联调，不会实际调用厂商）
	testAPIKey := "sk-ant-test-integration-debug-key-do-not-use-in-prod"
	encryptedKey, err := ks.Encrypt(testAPIKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "encrypt: %v\n", err)
		os.Exit(1)
	}

	// 3. 插入 account（如果 hint 已存在同 seller 则跳过）
	var accountID string
	err = dbPool.QueryRow(ctx,
		`INSERT INTO accounts
		 (seller_id, vendor, api_key_encrypted, api_key_hint,
		  total_credits_usd, authorized_credits_usd, consumed_credits_usd,
		  expected_rate, expire_at, health_score, status)
		 VALUES ($1, 'anthropic', $2, 'sk-ant-test***',
		  1000, 1000, 0, 0.75, $3, 90, 'active')
		 ON CONFLICT DO NOTHING
		 RETURNING id`,
		sellerID, encryptedKey, time.Now().Add(365*24*time.Hour),
	).Scan(&accountID)
	if err != nil || accountID == "" {
		// 可能已存在，查出来用
		err2 := dbPool.QueryRow(ctx,
			`SELECT id FROM accounts WHERE seller_id=$1 AND vendor='anthropic' AND api_key_hint='sk-ant-test***'`,
			sellerID,
		).Scan(&accountID)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "insert/fetch account: insert=%v fetch=%v\n", err, err2)
			os.Exit(1)
		}
		fmt.Println("account already exists, reusing:", accountID)
	} else {
		fmt.Println("account_id:", accountID)
	}

	// 4. 写入 Redis pool
	pool := scheduler.NewPool(rdb)
	err = pool.Upsert(ctx, &scheduler.AccountInfo{
		ID:           accountID,
		SellerID:     sellerID,
		Vendor:       "anthropic",
		Model:        "",    // 空=支持所有模型
		Health:       90,
		BalanceUSD:   1000,
		RPMLimit:     60,
		RPMUsed:      0,
		DailyLimit:   0,     // 0=不限
		DailyUsed:    0,
		PeakDaily:    0,
		Status:       "active",
		EncryptedKey: encryptedKey,
		Score:        90,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pool upsert: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("pool upsert: ok")
	fmt.Printf("\n✅ 测试账号就绪\n  account_id: %s\n  vendor: anthropic\n  pool: pool:anthropic\n", accountID)
	fmt.Println("\n⚠️  注意：该账号使用假 API Key，dispatch 会触发厂商调用失败（5001），这是正常的。")
	fmt.Println("  若需真实 dispatch 通过，请替换 testAPIKey 为有效的 Anthropic key。")
}
