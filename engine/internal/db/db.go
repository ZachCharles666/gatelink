package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Pool 是全局连接池，所有模块通过此访问数据库
type Pool struct {
	*pgxpool.Pool
}

// New 初始化数据库连接池
func New(ctx context.Context, databaseURL string) (*Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	// 连接池配置
	config.MaxConns = 20
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	// 连接前回调：不记录查询参数，避免泄露敏感数据
	config.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		return true
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// 验证连接
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	log.Info().
		Int32("max_conns", config.MaxConns).
		Msg("database pool initialized")

	return &Pool{pool}, nil
}

// Healthy 检查数据库连接是否正常
func (p *Pool) Healthy(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return p.Ping(ctx)
}

// Stats 返回连接池状态（用于监控）
func (p *Pool) Stats() map[string]interface{} {
	s := p.Pool.Stat()
	return map[string]interface{}{
		"acquired_conns":     s.AcquiredConns(),
		"idle_conns":         s.IdleConns(),
		"total_conns":        s.TotalConns(),
		"max_conns":          s.MaxConns(),
		"constructing_conns": s.ConstructingConns(),
	}
}
