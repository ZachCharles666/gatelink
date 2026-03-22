package seller

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/ZachCharles666/gatelink/api/internal/auth"
	engineclient "github.com/ZachCharles666/gatelink/api/internal/engine"
	"github.com/ZachCharles666/gatelink/api/internal/response"
	"github.com/gin-gonic/gin"
)

type engineAPI interface {
	VerifyAccount(ctx context.Context, accountID, apiKey string) (*engineclient.VerifyResult, error)
	GetAccountHealth(ctx context.Context, accountID string) (*engineclient.AccountHealth, error)
	GetConsoleUsage(ctx context.Context, accountID string) (*engineclient.ConsoleUsage, error)
	GetAccountDiff(ctx context.Context, accountID string) (*engineclient.DiffResult, error)
}

type settlementAPI interface {
	RequestSettlement(ctx context.Context, sellerID string) error
}

type Handler struct {
	svc        *Service
	settlement settlementAPI
	engine     engineAPI
}

func NewHandler(svc *Service, settlement settlementAPI, engine engineAPI) *Handler {
	return &Handler{svc: svc, settlement: settlement, engine: engine}
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Phone       string `json:"phone" binding:"required"`
		Code        string `json:"code" binding:"required"`
		DisplayName string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Code != "123456" {
		response.BadRequest(c, "invalid verification code")
		return
	}

	seller, err := h.svc.Register(c.Request.Context(), req.Phone, req.DisplayName)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	token, err := auth.GenerateToken(seller.ID, "seller")
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"token": token, "seller_id": seller.ID})
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Phone string `json:"phone" binding:"required"`
		Code  string `json:"code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Code != "123456" {
		response.Unauthorized(c)
		return
	}

	seller, err := h.svc.FindByPhone(c.Request.Context(), req.Phone)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	token, err := auth.GenerateToken(seller.ID, "seller")
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"token": token, "seller_id": seller.ID})
}

func (h *Handler) ListAccounts(c *gin.Context) {
	sellerID := c.GetString("user_id")
	accounts, err := h.svc.ListAccountsBySeller(c.Request.Context(), sellerID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"accounts": accounts})
}

func (h *Handler) GetAccount(c *gin.Context) {
	sellerID := c.GetString("user_id")
	accountID := c.Param("id")

	if !h.ensureOwnership(c, accountID, sellerID) {
		return
	}

	account, err := h.svc.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, account)
}

func (h *Handler) AddAccount(c *gin.Context) {
	sellerID := c.GetString("user_id")
	var req struct {
		Vendor               string  `json:"vendor" binding:"required"`
		APIKey               string  `json:"api_key" binding:"required"`
		AuthorizedCreditsUSD float64 `json:"authorized_credits_usd" binding:"required,gt=0"`
		ExpectedRate         float64 `json:"expected_rate" binding:"required,gte=0.5,lte=0.95"`
		ExpireAt             string  `json:"expire_at" binding:"required"`
		TotalCreditsUSD      float64 `json:"total_credits_usd"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	accountID, err := h.svc.PreCreateAccount(
		c.Request.Context(),
		sellerID,
		req.Vendor,
		req.AuthorizedCreditsUSD,
		req.ExpectedRate,
		req.TotalCreditsUSD,
		req.ExpireAt,
	)
	if err != nil {
		if errors.Is(err, errInvalidExpireAt) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	if h.engine == nil {
		_ = h.svc.DeleteAccount(c.Request.Context(), accountID)
		response.InternalError(c)
		return
	}

	verifyResult, err := h.engine.VerifyAccount(c.Request.Context(), accountID, req.APIKey)
	if err != nil {
		_ = h.svc.DeleteAccount(c.Request.Context(), accountID)
		if engineclient.IsNotFound(err) {
			response.Fail(c, http.StatusServiceUnavailable, response.CodeInternalError, "engine verify requires shared account persistence before live verification")
			return
		}
		response.InternalError(c)
		return
	}
	if !verifyResult.Valid {
		_ = h.svc.DeleteAccount(c.Request.Context(), accountID)
		response.BadRequest(c, fmt.Sprintf("API Key format check failed: %s", verifyErrorMessage(verifyResult)))
		return
	}

	response.OK(c, gin.H{
		"account_id":   accountID,
		"health_score": 80,
		"status":       "pending_verify",
		"message":      "格式检查通过，账号验证中。建议发起一次测试请求验证 Key 实际有效性。",
	})
}

func (h *Handler) UpdateAuthorization(c *gin.Context) {
	sellerID := c.GetString("user_id")
	accountID := c.Param("id")
	var req struct {
		AuthorizedCreditsUSD float64 `json:"authorized_credits_usd" binding:"required,gt=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !h.ensureOwnership(c, accountID, sellerID) {
		return
	}
	if err := h.svc.UpdateAuthorization(c.Request.Context(), accountID, req.AuthorizedCreditsUSD); err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{
		"account_id":             accountID,
		"authorized_credits_usd": req.AuthorizedCreditsUSD,
	})
}

func (h *Handler) RevokeAuthorization(c *gin.Context) {
	sellerID := c.GetString("user_id")
	accountID := c.Param("id")

	if !h.ensureOwnership(c, accountID, sellerID) {
		return
	}

	revokedAmount, err := h.svc.RevokeAuthorization(c.Request.Context(), accountID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{
		"revoked_amount_usd": revokedAmount,
		"message":            "授权已撤回，剩余额度归还成功",
	})
}

func (h *Handler) GetAccountUsage(c *gin.Context) {
	sellerID := c.GetString("user_id")
	accountID := c.Param("id")

	if !h.ensureOwnership(c, accountID, sellerID) {
		return
	}

	account, err := h.svc.GetAccount(c.Request.Context(), accountID)
	if err != nil {
		response.InternalError(c)
		return
	}
	if h.engine == nil {
		response.InternalError(c)
		return
	}

	type usageResult struct {
		health *engineclient.AccountHealth
		usage  *engineclient.ConsoleUsage
		diff   *engineclient.DiffResult
		err    error
	}

	ctx := c.Request.Context()
	var (
		wg       sync.WaitGroup
		result   usageResult
		resultMu sync.Mutex
	)

	setErr := func(err error) {
		if err == nil {
			return
		}
		resultMu.Lock()
		defer resultMu.Unlock()
		if result.err == nil {
			result.err = err
		}
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		health, err := h.engine.GetAccountHealth(ctx, accountID)
		if err != nil {
			setErr(err)
			return
		}
		resultMu.Lock()
		result.health = health
		resultMu.Unlock()
	}()
	go func() {
		defer wg.Done()
		usage, err := h.engine.GetConsoleUsage(ctx, accountID)
		if err != nil {
			setErr(err)
			return
		}
		resultMu.Lock()
		result.usage = usage
		resultMu.Unlock()
	}()
	go func() {
		defer wg.Done()
		diff, err := h.engine.GetAccountDiff(ctx, accountID)
		if err != nil {
			setErr(err)
			return
		}
		resultMu.Lock()
		result.diff = diff
		resultMu.Unlock()
	}()
	wg.Wait()

	if result.err != nil {
		response.InternalError(c)
		return
	}

	remaining := account.AuthorizedCreditsUSD - account.ConsumedCreditsUSD
	if remaining < 0 {
		remaining = 0
	}

	response.OK(c, gin.H{
		"authorized":    account.AuthorizedCreditsUSD,
		"consumed":      account.ConsumedCreditsUSD,
		"remaining":     remaining,
		"health_score":  getHealthScore(result.health),
		"daily_records": getRecords(result.usage),
		"diff_events":   getDiffs(result.diff),
	})
}

func (h *Handler) GetEarnings(c *gin.Context) {
	sellerID := c.GetString("user_id")

	seller, err := h.svc.GetSeller(c.Request.Context(), sellerID)
	if err != nil {
		response.InternalError(c)
		return
	}

	settlements, err := h.svc.GetRecentSettlements(c.Request.Context(), sellerID, 5)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{
		"pending_usd":      seller.PendingEarnUSD,
		"total_earned_usd": seller.TotalEarnedUSD,
		"settlements":      settlements,
	})
}

func (h *Handler) ListSettlements(c *gin.Context) {
	sellerID := c.GetString("user_id")
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		response.BadRequest(c, "invalid page")
		return
	}

	settlements, total, err := h.svc.ListSettlements(c.Request.Context(), sellerID, page, 20)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{
		"total":       total,
		"settlements": settlements,
	})
}

func (h *Handler) RequestSettlement(c *gin.Context) {
	sellerID := c.GetString("user_id")
	if h.settlement == nil {
		response.InternalError(c)
		return
	}
	if err := h.settlement.RequestSettlement(c.Request.Context(), sellerID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.OK(c, gin.H{"message": "结算申请已提交，等待管理员处理"})
}

func (h *Handler) ensureOwnership(c *gin.Context, accountID, sellerID string) bool {
	err := h.svc.VerifyOwnership(c.Request.Context(), accountID, sellerID)
	switch {
	case err == nil:
		return true
	case errors.Is(err, errForbidden):
		response.Fail(c, 403, response.CodeForbidden, "forbidden")
	case errors.Is(err, errAccountNotFound):
		response.NotFound(c, "account not found")
	default:
		response.InternalError(c)
	}

	return false
}

func verifyErrorMessage(result *engineclient.VerifyResult) string {
	if result == nil || result.ErrorMsg == "" {
		return "unknown verify error"
	}
	return result.ErrorMsg
}

func getHealthScore(health *engineclient.AccountHealth) int {
	if health == nil {
		return 80
	}
	return health.HealthScore
}

func getRecords(usage *engineclient.ConsoleUsage) []gin.H {
	if usage == nil {
		return []gin.H{}
	}

	records := make([]gin.H, 0, len(usage.Records))
	for _, record := range usage.Records {
		records = append(records, gin.H{
			"date":           record.Date,
			"total_cost_usd": record.TotalCostUSD,
			"input_tokens":   record.InputTokens,
			"output_tokens":  record.OutputTokens,
			"request_count":  record.RequestCount,
			"source":         "platform_record",
		})
	}

	return records
}

func getDiffs(diff *engineclient.DiffResult) []gin.H {
	if diff == nil {
		return []gin.H{}
	}

	diffs := make([]gin.H, 0, len(diff.Diffs))
	for _, event := range diff.Diffs {
		diffs = append(diffs, gin.H{
			"type":       event.Type,
			"detail":     event.Detail,
			"created_at": event.CreatedAt,
		})
	}

	return diffs
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "seller-fallback-id"
	}
	return hex.EncodeToString(buf)
}
