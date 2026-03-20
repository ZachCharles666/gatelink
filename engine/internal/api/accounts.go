package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourname/gatelink-engine/internal/db"
	"github.com/yourname/gatelink-engine/internal/health"
	"github.com/yourname/gatelink-engine/pkg/adapters"
)

// AccountsHandler 处理账号相关 API
type AccountsHandler struct {
	db      *db.Pool
	scorer  *health.Scorer
	registry *vendor.Registry
}

func NewAccountsHandler(db *db.Pool, scorer *health.Scorer, registry *vendor.Registry) *AccountsHandler {
	return &AccountsHandler{db: db, scorer: scorer, registry: registry}
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
