package buyer

import (
	"errors"
	"strconv"

	"github.com/ZachCharles666/gatelink/api/internal/auth"
	"github.com/ZachCharles666/gatelink/api/internal/response"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc         *Service
	invalidator auth.APIKeyCacheInvalidator
}

func NewHandler(svc *Service, invalidators ...auth.APIKeyCacheInvalidator) *Handler {
	var invalidator auth.APIKeyCacheInvalidator
	if len(invalidators) > 0 {
		invalidator = invalidators[0]
	}

	return &Handler{svc: svc, invalidator: invalidator}
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password" binding:"required"`
		Code     string `json:"code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Email == "" && req.Phone == "" {
		response.BadRequest(c, "email or phone is required")
		return
	}

	apiKey, err := auth.GenerateBuyerAPIKey()
	if err != nil {
		response.InternalError(c)
		return
	}

	buyer, err := h.svc.Register(c.Request.Context(), req.Email, req.Phone, req.Password, apiKey)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	token, err := auth.GenerateToken(buyer.ID, "buyer")
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{
		"token":    token,
		"buyer_id": buyer.ID,
		"api_key":  apiKey,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.Email == "" && req.Phone == "" {
		response.BadRequest(c, "email or phone is required")
		return
	}

	buyer, err := h.svc.Login(c.Request.Context(), req.Email, req.Phone, req.Password)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	token, err := auth.GenerateToken(buyer.ID, "buyer")
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"token": token, "buyer_id": buyer.ID})
}

func (h *Handler) GetBalance(c *gin.Context) {
	buyerID := c.GetString("user_id")

	buyer, err := h.svc.GetBalance(c.Request.Context(), buyerID)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	response.OK(c, gin.H{
		"balance_usd":        buyer.BalanceUSD,
		"total_consumed_usd": buyer.TotalConsumedUSD,
		"tier":               buyer.Tier,
	})
}

func (h *Handler) Topup(c *gin.Context) {
	buyerID := c.GetString("user_id")
	var req struct {
		AmountUSD float64 `json:"amount_usd" binding:"required,gt=0"`
		TxHash    string  `json:"tx_hash" binding:"required"`
		Network   string  `json:"network" binding:"required,oneof=TRC20 ERC20"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	record, err := h.svc.CreateTopup(c.Request.Context(), buyerID, req.AmountUSD, req.TxHash, req.Network)
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			response.BadRequest(c, "tx_hash already submitted")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{
		"topup_id":   record.ID,
		"amount_usd": record.AmountUSD,
		"network":    record.Network,
		"status":     record.Status,
		"message":    "充值申请已提交，等待管理员审核（通常 1-24 小时）",
	})
}

func (h *Handler) ListTopupRecords(c *gin.Context) {
	buyerID := c.GetString("user_id")
	page := parsePage(c)

	records, total, err := h.svc.ListTopupRecords(c.Request.Context(), buyerID, page, 20)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"total": total, "records": records})
}

func (h *Handler) GetUsage(c *gin.Context) {
	buyerID := c.GetString("user_id")
	page := parsePage(c)

	records, total, err := h.svc.GetUsageRecords(c.Request.Context(), buyerID, page, 20)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"total": total, "records": records})
}

func (h *Handler) ResetAPIKey(c *gin.Context) {
	buyerID := c.GetString("user_id")
	currentBuyer, err := h.svc.FindByID(c.Request.Context(), buyerID)
	if err != nil {
		response.InternalError(c)
		return
	}

	newKey, err := auth.GenerateBuyerAPIKey()
	if err != nil {
		response.InternalError(c)
		return
	}
	if err := h.svc.ResetAPIKey(c.Request.Context(), buyerID, newKey); err != nil {
		response.InternalError(c)
		return
	}
	if h.invalidator != nil {
		h.invalidator.InvalidateAPIKey(c.Request.Context(), currentBuyer.APIKey)
		h.invalidator.InvalidateAPIKey(c.Request.Context(), newKey)
	}

	response.OK(c, gin.H{
		"api_key": newKey,
		"message": "API Key 已重置，旧 Key 立即失效",
	})
}

func parsePage(c *gin.Context) int {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}
