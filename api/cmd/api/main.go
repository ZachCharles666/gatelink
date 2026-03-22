package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ZachCharles666/gatelink/api/internal/accounting"
	"github.com/ZachCharles666/gatelink/api/internal/admin"
	apirouter "github.com/ZachCharles666/gatelink/api/internal/api"
	"github.com/ZachCharles666/gatelink/api/internal/auth"
	"github.com/ZachCharles666/gatelink/api/internal/buyer"
	"github.com/ZachCharles666/gatelink/api/internal/engine"
	"github.com/ZachCharles666/gatelink/api/internal/poller"
	"github.com/ZachCharles666/gatelink/api/internal/proxy"
	"github.com/ZachCharles666/gatelink/api/internal/seller"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()

	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	sellerSvc := seller.NewService()
	settlementSvc := accounting.NewSettlementService(sellerSvc)
	engineClient := engine.New()
	sellerH := seller.NewHandler(sellerSvc, settlementSvc, engineClient)
	buyerSvc := buyer.NewService()
	buyerRepo := auth.BuyerRepo(buyerSvc)
	var invalidator auth.APIKeyCacheInvalidator
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Warn().Err(err).Str("redis_url", redisURL).Msg("invalid REDIS_URL, api key cache disabled")
		} else {
			rdb := redis.NewClient(opts)
			cachedRepo := auth.NewCachedBuyerRepo(buyerSvc, rdb)
			buyerRepo = cachedRepo
			invalidator = cachedRepo
			log.Info().Msg("buyer api key redis cache enabled")
		}
	}
	buyerH := buyer.NewHandler(buyerSvc, invalidator)
	adminH := admin.NewHandler(buyerSvc, sellerSvc)
	accountingSvc := accounting.NewService(buyerSvc, sellerSvc)
	proxyH := proxy.NewHandler(engineClient, accountingSvc)
	accountPoller := poller.NewAccountPoller(sellerSvc, func(accountID, status string) {
		log.Info().Str("account_id", accountID).Str("status", status).Msg("account status updated")
	})

	apirouter.SetupRoutes(r, sellerH, buyerH, adminH, proxyH, buyerRepo)
	go accountPoller.Start(context.Background())

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	log.Info().Str("port", port).Msg("api service started")

	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatal().Err(err).Msg("server error")
	}
}
