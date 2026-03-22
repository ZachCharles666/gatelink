package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/yourname/gatelink-engine/internal/crypto"
	"github.com/yourname/gatelink-engine/internal/db"
	"github.com/yourname/gatelink-engine/internal/health"
	"github.com/yourname/gatelink-engine/internal/scheduler"
	"github.com/yourname/gatelink-engine/pkg/adapters"
)

// AccountsHandler 处理账号相关 API
type AccountsHandler struct {
	db       *db.Pool
	pool     *scheduler.Pool
	scorer   *health.Scorer
	registry *vendor.Registry
	ks       *crypto.Keystore
}

func NewAccountsHandler(db *db.Pool, pool *scheduler.Pool, scorer *health.Scorer, registry *vendor.Registry, ks *crypto.Keystore) *AccountsHandler {
	return &AccountsHandler{db: db, pool: pool, scorer: scorer, registry: registry, ks: ks}
}

// HandleHealth GET /internal/v1/accounts/:id/health
// 返回账号当前健康分 + 最近 20 条事件
func (h *AccountsHandler) HandleHealth(c *gin.Context) {
	accountID := c.Param("id")

	var score int
	var status, vendorName string
	err := h.db.QueryRow(c.Request.Context(),
		"SELECT health_score, status, vendor FROM accounts WHERE id = $1", accountID,
	).Scan(&score, &status, &vendorName)
	if err != nil {
		if db.IsNotFound(err) {
			NotFound(c, "account not found")
			return
		}
		InternalError(c)
		return
	}

	// 最近 20 条健康事件
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT event_type, score_delta, score_after, created_at
		FROM health_events
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT 20`, accountID)
	if err != nil {
		InternalError(c)
		return
	}
	defer rows.Close()

	type event struct {
		Type       string    `json:"type"`
		Delta      int       `json:"delta"`
		ScoreAfter int       `json:"score_after"`
		CreatedAt  time.Time `json:"created_at"`
	}
	var events []event
	for rows.Next() {
		var e event
		if err := rows.Scan(&e.Type, &e.Delta, &e.ScoreAfter, &e.CreatedAt); err != nil {
			continue
		}
		events = append(events, e)
	}

	OK(c, gin.H{
		"account_id":   accountID,
		"health_score": score,
		"status":       status,
		"vendor":       vendorName,
		"recent_events": events,
	})
}

// HandleVerify POST /internal/v1/accounts/:id/verify
// 触发 API Key 验证，更新账号状态
func (h *AccountsHandler) HandleVerify(c *gin.Context) {
	accountID := c.Param("id")

	var encryptedKey, vendorName string
	err := h.db.QueryRow(c.Request.Context(),
		"SELECT api_key_encrypted, vendor FROM accounts WHERE id = $1", accountID,
	).Scan(&encryptedKey, &vendorName)
	if err != nil {
		if db.IsNotFound(err) {
			NotFound(c, "account not found")
			return
		}
		InternalError(c)
		return
	}

	// 获取适配器
	adapter, ok := h.registry.Get(vendor.Vendor(vendorName))
	if !ok {
		Fail(c, 400, CodeInvalidParam, "unsupported vendor: "+vendorName)
		return
	}

	// MVP：仅做格式校验（完整验证需解密 key + 网络请求，留给 Week 10 集成测试）
	result, err := adapter.ValidateKey(c.Request.Context(), "format-check-only")
	if err != nil {
		InternalError(c)
		return
	}

	OK(c, gin.H{
		"account_id": accountID,
		"vendor":     vendorName,
		"valid":      result.Valid,
		"error_msg":  result.ErrorMsg,
	})
}

// HandleConsoleUsage GET /internal/v1/accounts/:id/console-usage
// 返回最近 30 天的 Console 同步记录
func (h *AccountsHandler) HandleConsoleUsage(c *gin.Context) {
	accountID := c.Param("id")

	// 从 usage_records 按日聚合（MVP：Console 同步记录表未单独建，用 usage_records 代替）
	rows, err := h.db.Query(c.Request.Context(), `
		SELECT
			DATE(created_at) AS date,
			SUM(cost_usd)::float AS total_cost_usd,
			SUM(input_tokens) AS input_tokens,
			SUM(output_tokens) AS output_tokens,
			COUNT(*) AS request_count
		FROM usage_records
		WHERE account_id = $1
		  AND created_at >= NOW() - INTERVAL '30 days'
		GROUP BY DATE(created_at)
		ORDER BY date DESC`, accountID)
	if err != nil {
		InternalError(c)
		return
	}
	defer rows.Close()

	type dailyRecord struct {
		Date         string  `json:"date"`
		TotalCostUSD float64 `json:"total_cost_usd"`
		InputTokens  int64   `json:"input_tokens"`
		OutputTokens int64   `json:"output_tokens"`
		RequestCount int64   `json:"request_count"`
	}
	var records []dailyRecord
	for rows.Next() {
		var r dailyRecord
		if err := rows.Scan(&r.Date, &r.TotalCostUSD, &r.InputTokens, &r.OutputTokens, &r.RequestCount); err != nil {
			continue
		}
		records = append(records, r)
	}

	OK(c, gin.H{
		"account_id": accountID,
		"records":    records,
	})
}

// HandleDiff GET /internal/v1/accounts/:id/diff
// 返回平台记录 vs Console 的差异汇总（MVP：从健康事件日志中读取对账结果）
func (h *AccountsHandler) HandleDiff(c *gin.Context) {
	accountID := c.Param("id")

	rows, err := h.db.Query(c.Request.Context(), `
		SELECT event_type, detail, created_at
		FROM health_events
		WHERE account_id = $1
		  AND event_type IN ('reconcile_pass', 'reconcile_fail')
		ORDER BY created_at DESC
		LIMIT 30`, accountID)
	if err != nil {
		InternalError(c)
		return
	}
	defer rows.Close()

	type diffRecord struct {
		Type      string    `json:"type"`
		Detail    *string   `json:"detail,omitempty"`
		CreatedAt time.Time `json:"created_at"`
	}
	var records []diffRecord
	for rows.Next() {
		var r diffRecord
		if err := rows.Scan(&r.Type, &r.Detail, &r.CreatedAt); err != nil {
			continue
		}
		records = append(records, r)
	}

	OK(c, gin.H{
		"account_id": accountID,
		"diffs":      records,
	})
}

// CreateAccountRequest POST /internal/v1/accounts 请求体
type CreateAccountRequest struct {
	SellerID             string    `json:"seller_id" binding:"required"`
	Vendor               string    `json:"vendor" binding:"required"`
	APIKey               string    `json:"api_key" binding:"required"`
	TotalCreditsUSD      float64   `json:"total_credits_usd" binding:"required"`
	AuthorizedCreditsUSD float64   `json:"authorized_credits_usd" binding:"required"`
	ExpectedRate         float64   `json:"expected_rate"`
	ExpireAt             time.Time `json:"expire_at" binding:"required"`
}

// HandleCreate POST /internal/v1/accounts
// 接收明文 api_key，由 engine 加密写库并将账号加入调度池
func (h *AccountsHandler) HandleCreate(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	ctx := c.Request.Context()

	// 1. 验证 vendor 合法
	if _, ok := h.registry.Get(vendor.Vendor(req.Vendor)); !ok {
		BadRequest(c, "unsupported vendor: "+req.Vendor)
		return
	}

	// 2. 加密 api_key（明文只存在于当前栈帧，加密后立即丢弃）
	encryptedKey, err := h.ks.Encrypt(req.APIKey)
	if err != nil {
		log.Error().Err(err).Msg("encrypt api key failed")
		InternalError(c)
		return
	}
	hint := crypto.Hint(req.APIKey)

	// 3. 确保 seller 在 engine DB 中存在（满足 FK 约束；seller 完整档案由 api 服务管理）
	if _, err := h.db.Exec(ctx,
		`INSERT INTO sellers (id, status) VALUES ($1, 'active') ON CONFLICT (id) DO NOTHING`,
		req.SellerID,
	); err != nil {
		log.Error().Err(err).Str("seller_id", req.SellerID).Msg("upsert seller stub failed")
		InternalError(c)
		return
	}

	// 4. 写入 accounts 表
	expectedRate := req.ExpectedRate
	if expectedRate == 0 {
		expectedRate = 0.75
	}
	var accountID string
	err = h.db.QueryRow(ctx, `
		INSERT INTO accounts
		  (seller_id, vendor, api_key_encrypted, api_key_hint,
		   total_credits_usd, authorized_credits_usd, consumed_credits_usd,
		   expected_rate, expire_at, health_score, status)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $7, $8, 80, 'active')
		RETURNING id`,
		req.SellerID, req.Vendor, encryptedKey, hint,
		req.TotalCreditsUSD, req.AuthorizedCreditsUSD,
		expectedRate, req.ExpireAt,
	).Scan(&accountID)
	if err != nil {
		log.Error().Err(err).Str("seller_id", req.SellerID).Msg("insert account failed")
		InternalError(c)
		return
	}

	// 5. 写入 Redis 调度池；失败时回滚 DB，保持 DB 与 pool 一致
	upsertErr := h.pool.Upsert(ctx, &scheduler.AccountInfo{
		ID:           accountID,
		SellerID:     req.SellerID,
		Vendor:       req.Vendor,
		Health:       80,
		BalanceUSD:   req.AuthorizedCreditsUSD,
		RPMLimit:     60,
		Status:       "active",
		EncryptedKey: encryptedKey,
		Score:        80,
	})
	if upsertErr != nil {
		log.Error().Err(upsertErr).Str("account_id", accountID).Msg("pool upsert failed, rolling back DB insert")
		if _, delErr := h.db.Exec(ctx, "DELETE FROM accounts WHERE id = $1", accountID); delErr != nil {
			log.Error().Err(delErr).Str("account_id", accountID).Msg("rollback delete failed, account stuck in DB without pool entry")
		}
		InternalError(c)
		return
	}

	OK(c, gin.H{
		"account_id":   accountID,
		"api_key_hint": hint,
		"vendor":       req.Vendor,
		"status":       "active",
	})
}
