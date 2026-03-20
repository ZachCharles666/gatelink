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
	"github.com/yourname/tokenglide-engine/internal/api"
	"github.com/yourname/tokenglide-engine/internal/crypto"
	"github.com/yourname/tokenglide-engine/internal/db"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg(".env file not found, using system env")
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Str("service", "engine").Logger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 初始化加密模块（启动时验证密钥配置）
	if _, err := crypto.Global(); err != nil {
		log.Fatal().Err(err).Msg("encryption key initialization failed")
	}
	log.Info().Msg("encryption key loaded")

	// 初始化数据库
	dbPool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal().Err(err).Msg("database connection failed")
	}
	defer dbPool.Close()
	log.Info().Msg("database connected")

	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	router := api.New(dbPool)
	router.Register(engine)

	port := os.Getenv("ENGINE_PORT")
	if port == "" {
		port = "8081"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      engine,
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
