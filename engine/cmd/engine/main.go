package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/yourname/gatelink-engine/internal/api"
	"github.com/yourname/gatelink-engine/internal/audit"
	"github.com/yourname/gatelink-engine/internal/config"
	"github.com/yourname/gatelink-engine/internal/crypto"
	"github.com/yourname/gatelink-engine/internal/db"
	"github.com/yourname/gatelink-engine/internal/health"
	"github.com/yourname/gatelink-engine/internal/proxy"
	"github.com/yourname/gatelink-engine/internal/scheduler"
	syncer "github.com/yourname/gatelink-engine/internal/sync"
	syncadapters "github.com/yourname/gatelink-engine/internal/sync/adapters"
	"github.com/yourname/gatelink-engine/pkg/adapters"
	anthropicadapter "github.com/yourname/gatelink-engine/pkg/adapters/anthropic"
	geminiandapter "github.com/yourname/gatelink-engine/pkg/adapters/gemini"
	glmadapter "github.com/yourname/gatelink-engine/pkg/adapters/glm"
	kimiadapter "github.com/yourname/gatelink-engine/pkg/adapters/kimi"
	openaiadapter "github.com/yourname/gatelink-engine/pkg/adapters/openai"
	qwenadapter "github.com/yourname/gatelink-engine/pkg/adapters/qwen"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg(".env file not found, using system env")
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Str("service", "engine").Logger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 加密模块
	ks, err := crypto.Global()
	if err != nil {
		log.Fatal().Err(err).Msg("encryption key initialization failed")
	}
	log.Info().Msg("encryption key loaded")

	// 数据库
	dbPool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("database connection failed")
	}
	defer dbPool.Close()
	log.Info().Msg("database connected")

	// Redis
	redisClient, err := config.NewRedis(ctx, os.Getenv("REDIS_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("redis connection failed")
	}
	defer redisClient.Close()

	// 账号池 + 调度引擎
	pool := scheduler.NewPool(redisClient)
	engine := scheduler.NewEngine(pool)

	// 健康度系统
	healthScorer := health.NewScorer(dbPool, redisClient)
	_ = health.NewMonitor(healthScorer, dbPool, redisClient)

	// 内容审核
	filter := audit.NewFilter()
	classifier := audit.NewClassifier(filter)

	// 适配器注册表
	registry := vendor.NewRegistry()
	registry.Register(anthropicadapter.New())
	registry.Register(openaiadapter.New())
	registry.Register(geminiandapter.New())
	registry.Register(qwenadapter.New())
	registry.Register(glmadapter.New())
	registry.Register(kimiadapter.New())
	log.Info().Msg("vendor adapters registered (6 vendors)")

	// Console 用量同步器（每日 UTC 02:00 自动执行对账）
	consoleSyncer := syncer.New(dbPool, redisClient, healthScorer,
		syncadapters.NewAnthropic(),
		syncadapters.NewOpenAI(),
		syncadapters.NewStub("gemini"),
		syncadapters.NewStub("qwen"),
		syncadapters.NewStub("glm"),
		syncadapters.NewStub("kimi"),
	)
	consoleSyncer.StartSchedule(ctx)
	log.Info().Msg("console sync scheduler started")

	// 转发器
	forwarder := proxy.New(registry, ks, dbPool, engine)

	// Gin + 路由
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	ginEngine := gin.New()
	router := api.New(dbPool, pool, engine, forwarder, registry, classifier, healthScorer, ks)
	router.Register(ginEngine)

	port := os.Getenv("ENGINE_PORT")
	if port == "" {
		port = "8081"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      ginEngine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Info().Msg("shutting down gracefully...")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutCancel()
		srv.Shutdown(shutCtx)
		cancel()
	}()

	log.Info().Str("port", port).Msg("engine started")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("server error")
	}
}
