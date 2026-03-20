package config

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// NewRedis 从 REDIS_URL 创建 Redis 客户端并验证连接
// 支持格式：redis://:password@host:port/db
func NewRedis(ctx context.Context, redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	u, err := url.Parse(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	opt := &redis.Options{
		Addr: u.Host,
		DB:   0,
	}

	if u.User != nil {
		if pass, ok := u.User.Password(); ok {
			opt.Password = pass
		}
	}

	if u.Path != "" && u.Path != "/" {
		db, err := strconv.Atoi(u.Path[1:])
		if err == nil {
			opt.DB = db
		}
	}

	client := redis.NewClient(opt)

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	log.Info().Str("addr", opt.Addr).Int("db", opt.DB).Msg("redis connected")
	return client, nil
}
