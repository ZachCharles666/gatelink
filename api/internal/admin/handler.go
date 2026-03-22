package admin

import (
	"context"

	"github.com/ZachCharles666/gatelink/api/internal/buyer"
	"github.com/ZachCharles666/gatelink/api/internal/response"
	"github.com/ZachCharles666/gatelink/api/internal/seller"
	"github.com/gin-gonic/gin"
)

type buyerAdmin interface {
	ListPendingTopups(ctx context.Context, limit int) ([]buyer.PendingTopupRecord, error)
	ConfirmTopup(ctx context.Context, topupID string) (*buyer.TopupRecord, *buyer.Buyer, error)
	RejectTopup(ctx context.Context, topupID, reason string) (*buyer.TopupRecord, error)
}

type sellerAdmin interface {
	ForceSuspendAccount(ctx context.Context, accountID string) (*seller.Account, error)
	ListPendingSettlements(ctx context.Context, limit int) ([]seller.PendingSettlement, error)
	PaySettlement(ctx context.Context, settlementID, txHash string) (*seller.Settlement, error)
}

type Handler struct {
	buyers  buyerAdmin
	sellers sellerAdmin
}

func NewHandler(buyers buyerAdmin, sellers sellerAdmin) *Handler {
	return &Handler{buyers: buyers, sellers: sellers}
}

func (h *Handler) ListPendingTopup(c *gin.Context) {
	records, err := h.buyers.ListPendingTopups(c.Request.Context(), 50)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"records": records})
}

func (h *Handler) ConfirmTopup(c *gin.Context) {
	topupID := c.Param("id")

	record, adminBuyer, err := h.buyers.ConfirmTopup(c.Request.Context(), topupID)
	if err != nil {
		if buyer.IsTopupUnavailable(err) {
			response.NotFound(c, "topup record not found or already processed")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{
		"topup_id":    record.ID,
		"buyer_id":    record.BuyerID,
		"amount_usd":  record.AmountUSD,
		"balance_usd": adminBuyer.BalanceUSD,
		"message":     "充值已确认，买家余额已更新",
	})
}

func (h *Handler) RejectTopup(c *gin.Context) {
	topupID := c.Param("id")
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	if _, err := h.buyers.RejectTopup(c.Request.Context(), topupID, req.Reason); err != nil {
		if buyer.IsTopupUnavailable(err) {
			response.NotFound(c, "topup record not found or already processed")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"message": "充值已拒绝"})
}

func (h *Handler) ForceSuspend(c *gin.Context) {
	accountID := c.Param("id")

	account, err := h.sellers.ForceSuspendAccount(c.Request.Context(), accountID)
	if err != nil {
		if seller.IsAccountNotFound(err) {
			response.NotFound(c, "account not found")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"account_id": account.ID, "status": account.Status})
}

func (h *Handler) ListPendingSettlements(c *gin.Context) {
	settlements, err := h.sellers.ListPendingSettlements(c.Request.Context(), 50)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"settlements": settlements})
}

func (h *Handler) PaySettlement(c *gin.Context) {
	settlementID := c.Param("id")
	var req struct {
		TxHash string `json:"tx_hash" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	settlement, err := h.sellers.PaySettlement(c.Request.Context(), settlementID, req.TxHash)
	if err != nil {
		if seller.IsSettlementUnavailable(err) {
			response.NotFound(c, "settlement not found or already processed")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{
		"settlement_id": settlement.ID,
		"status":        settlement.Status,
		"message":       "结算已完成",
	})
}
