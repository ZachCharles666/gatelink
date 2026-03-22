package api

import (
	"github.com/ZachCharles666/gatelink/api/internal/admin"
	"github.com/ZachCharles666/gatelink/api/internal/auth"
	"github.com/ZachCharles666/gatelink/api/internal/buyer"
	"github.com/ZachCharles666/gatelink/api/internal/proxy"
	"github.com/ZachCharles666/gatelink/api/internal/response"
	"github.com/ZachCharles666/gatelink/api/internal/seller"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, sellerH *seller.Handler, buyerH *buyer.Handler, adminH *admin.Handler, proxyH *proxy.Handler, buyerRepo auth.BuyerRepo) {
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{
			"service": "api",
			"version": "0.1.0",
		})
	})

	sellerGroup := r.Group("/api/v1/seller")
	sellerGroup.POST("/auth/register", sellerH.Register)
	sellerGroup.POST("/auth/login", sellerH.Login)
	sellerAuth := sellerGroup.Group("")
	sellerAuth.Use(auth.RequireRole("seller"))
	{
		sellerAuth.GET("/accounts", sellerH.ListAccounts)
		sellerAuth.GET("/accounts/:id", sellerH.GetAccount)
		sellerAuth.POST("/accounts", sellerH.AddAccount)
		sellerAuth.PATCH("/accounts/:id/authorization", sellerH.UpdateAuthorization)
		sellerAuth.DELETE("/accounts/:id/authorization", sellerH.RevokeAuthorization)
		sellerAuth.GET("/accounts/:id/usage", sellerH.GetAccountUsage)
		sellerAuth.GET("/earnings", sellerH.GetEarnings)
		sellerAuth.GET("/settlements", sellerH.ListSettlements)
		sellerAuth.POST("/settlements/request", sellerH.RequestSettlement)
	}

	buyerGroup := r.Group("/api/v1/buyer")
	buyerGroup.POST("/auth/register", buyerH.Register)
	buyerGroup.POST("/auth/login", buyerH.Login)
	buyerAuth := buyerGroup.Group("")
	buyerAuth.Use(auth.RequireRole("buyer"))
	{
		buyerAuth.GET("/balance", buyerH.GetBalance)
		buyerAuth.GET("/usage", buyerH.GetUsage)
		buyerAuth.POST("/topup", buyerH.Topup)
		buyerAuth.GET("/topup/records", buyerH.ListTopupRecords)
		buyerAuth.POST("/apikeys/reset", buyerH.ResetAPIKey)
	}

	if adminH != nil {
		adminGroup := r.Group("/api/v1/admin")
		adminGroup.Use(auth.RequireRole("admin"))
		{
			adminGroup.GET("/topup/pending", adminH.ListPendingTopup)
			adminGroup.POST("/topup/:id/confirm", adminH.ConfirmTopup)
			adminGroup.POST("/topup/:id/reject", adminH.RejectTopup)
			adminGroup.GET("/settlements/pending", adminH.ListPendingSettlements)
			adminGroup.POST("/settlements/:id/pay", adminH.PaySettlement)
			adminGroup.POST("/accounts/:id/force-suspend", adminH.ForceSuspend)
		}
	}

	proxyGroup := r.Group("/v1")
	proxyGroup.Use(auth.BuyerAPIKeyMiddleware(buyerRepo))
	{
		proxyGroup.POST("/chat/completions", proxyH.ChatCompletions)
		proxyGroup.GET("/models", proxyH.ListModels)
	}
}
