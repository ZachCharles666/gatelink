package auth

import (
	"context"
	"strings"

	"github.com/ZachCharles666/gatelink/api/internal/response"
	"github.com/gin-gonic/gin"
)

type BuyerRepo interface {
	FindByAPIKey(ctx context.Context, apiKey string) (*BuyerInfo, error)
}

type BuyerInfo struct {
	ID         string
	BalanceUSD float64
	Tier       string
	Status     string
}

func BuyerAPIKeyMiddleware(repo BuyerRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		buyer, err := repo.FindByAPIKey(c.Request.Context(), strings.TrimPrefix(header, "Bearer "))
		if err != nil || buyer.Status != "active" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set("buyer_id", buyer.ID)
		c.Set("buyer_balance", buyer.BalanceUSD)
		c.Set("buyer_tier", buyer.Tier)
		c.Next()
	}
}
