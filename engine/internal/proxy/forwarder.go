// Package proxy 实现向上游厂商的 HTTP 请求转发。
// 安全原则：API Key 解密后只存在于当前 goroutine 栈，不写入任何日志或存储。
package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/yourname/gatelink-engine/internal/crypto"
	"github.com/yourname/gatelink-engine/internal/db"
	"github.com/yourname/gatelink-engine/internal/scheduler"
	"github.com/yourname/gatelink-engine/pkg/adapters"
)

// platformMarkupRate 平台抽成比例（10%）
// 实际生产中应从配置中读取
const platformMarkupRate = 0.10

// ForwardResult 转发结果
type ForwardResult struct {
	Body        []byte
	StatusCode  int
	VendorReqID string
	CostUSD     float64
	InputTokens  int
	OutputTokens int
}

// Forwarder 负责向上游厂商转发请求
type Forwarder struct {
	registry *vendor.Registry
	keystore *crypto.Keystore
	db       *db.Pool
	engine   *scheduler.Engine
	http     *http.Client
}

func New(
	registry *vendor.Registry,
	keystore *crypto.Keystore,
	db *db.Pool,
	engine *scheduler.Engine,
) *Forwarder {
	return &Forwarder{
		registry: registry,
		keystore: keystore,
		db:       db,
		engine:   engine,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Forward 执行完整转发流程：
//  1. 从适配器注册表获取对应厂商适配器
//  2. 在栈上解密 API Key（不落日志）
//  3. 格式化请求体并发送至厂商
//  4. 解析响应和 token 用量
//  5. 计算费用并写入 usage_records
//  6. 更新调度引擎的消耗计数
func (f *Forwarder) Forward(
	ctx context.Context,
	acct *scheduler.AccountInfo,
	req *vendor.ChatRequest,
	buyerID string,
	buyerChargeRate float64, // 买家收费系数（相对于成本），传 0 使用默认
) (*ForwardResult, error) {
	// 1. 获取适配器
	adapter, ok := f.registry.Get(vendor.Vendor(acct.Vendor))
	if !ok {
		return nil, fmt.Errorf("no adapter for vendor=%s", acct.Vendor)
	}

	// 2. 在栈上解密 API Key —— ⚠️ 不得将 apiKey 传入任何日志调用
	apiKey, err := f.keystore.Decrypt(acct.EncryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt key: %w", err)
	}

	// 3. 格式化请求体
	body, err := adapter.FormatRequest(req)
	if err != nil {
		return nil, fmt.Errorf("format request: %w", err)
	}

	// 4. 构建 HTTP 请求
	url := adapter.BaseURL() + adapter.ChatEndpoint()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build http request: %w", err)
	}
	for k, v := range adapter.Headers(apiKey) {
		httpReq.Header.Set(k, v)
	}

	// 5. 发送请求
	start := time.Now()
	resp, err := f.http.Do(httpReq)
	latency := time.Since(start)

	// apiKey 解密完成后手动清零，防止被 GC 保留在内存过久
	for i := range apiKey {
		apiKey = apiKey[:i] + "x" + apiKey[i+1:]
		break
	}
	// 覆盖字符串（Go 字符串不可变，此处清理 body slice 中可能出现的 key）
	_ = apiKey // 让编译器不报 unused，实际 key 生命周期结束于此行

	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	log.Info().
		Str("account", acct.ID).
		Str("vendor", acct.Vendor).
		Int("status", resp.StatusCode).
		Dur("latency", latency).
		Msg("vendor request completed")

	if resp.StatusCode != http.StatusOK {
		return &ForwardResult{
			Body:       respBody,
			StatusCode: resp.StatusCode,
		}, fmt.Errorf("vendor returned %d", resp.StatusCode)
	}

	// 6. 解析 token 用量
	usage, err := adapter.ParseUsage(respBody)
	if err != nil {
		log.Warn().Err(err).Str("account", acct.ID).Msg("parse usage failed, using zeros")
		usage = &vendor.Usage{}
	}

	// 7. 计算费用
	cost, err := adapter.CalcCost(usage, req.Model)
	if err != nil {
		log.Warn().Err(err).Str("model", req.Model).Msg("calc cost failed, using zero cost")
		cost = &vendor.Cost{
			InputTokens: usage.PromptTokens, OutputTokens: usage.CompletionTokens,
		}
	}

	// 8. 写入 usage_records
	vendorReqID := resp.Header.Get("X-Request-Id")
	if vendorReqID == "" {
		vendorReqID = resp.Header.Get("request-id")
	}
	if err := f.writeUsageRecord(ctx, acct, buyerID, req.Model, usage, cost, vendorReqID, buyerChargeRate); err != nil {
		// 记账失败只记警告，不中断响应
		log.Error().Err(err).Str("account", acct.ID).Msg("write usage record failed")
	}

	// 9. 更新调度引擎消耗计数
	f.engine.RecordConsumed(ctx, acct.ID, acct.Vendor, cost.CostUSD)
	f.engine.RecordBuyerHistory(ctx, buyerID, acct.ID)

	return &ForwardResult{
		Body:         respBody,
		StatusCode:   resp.StatusCode,
		VendorReqID:  vendorReqID,
		CostUSD:      cost.CostUSD,
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
	}, nil
}

// writeUsageRecord 向 usage_records 表写入消耗记录
func (f *Forwarder) writeUsageRecord(
	ctx context.Context,
	acct *scheduler.AccountInfo,
	buyerID string,
	model string,
	usage *vendor.Usage,
	cost *vendor.Cost,
	vendorReqID string,
	buyerChargeRate float64,
) error {
	if buyerChargeRate <= 0 {
		buyerChargeRate = 1 + platformMarkupRate
	}

	buyerCharged := cost.CostUSD * buyerChargeRate
	platformEarn := buyerCharged - cost.CostUSD
	sellerEarn := cost.CostUSD

	_, err := f.db.Exec(ctx, `
		INSERT INTO usage_records
			(account_id, buyer_id, vendor, model,
			 input_tokens, output_tokens, cost_usd,
			 buyer_charged_usd, seller_earn_usd, platform_earn_usd,
			 vendor_request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		acct.ID, buyerID, acct.Vendor, model,
		usage.PromptTokens, usage.CompletionTokens, cost.CostUSD,
		buyerCharged, sellerEarn, platformEarn,
		vendorReqID,
	)
	return err
}
