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
	"github.com/yourname/tokenglide-engine/internal/db"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg(".env file not found, using system env")
	}

	// 初始化日志
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Str("service", "engine").Logger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化数据库
	dbPool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()
	log.Info().Msg("database connected")

	// 初始化路由
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// 健康检查（含 DB 状态）
	r.GET("/health", func(c *gin.Context) {
		dbStatus := "ok"
		if err := dbPool.Healthy(c.Request.Context()); err != nil {
			dbStatus = "error: " + err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"service":  "engine",
			"version":  "0.1.0",
			"database": dbStatus,
			"db_stats": dbPool.Stats(),
		})
	})

	port := os.Getenv("ENGINE_PORT")
	if port == "" {
		port = "8081"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: r,
	}

	// 优雅退出
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Info().Msg("shutting down...")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutCancel()
		srv.Shutdown(shutCtx)
		cancel()
	}()

	log.Info().Str("port", port).Msg("engine starting")
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("engine failed")
	}
}
