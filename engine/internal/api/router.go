package api

import (
	"github.com/gin-gonic/gin"
	"github.com/yourname/gatelink-engine/internal/audit"
	"github.com/yourname/gatelink-engine/internal/crypto"
	"github.com/yourname/gatelink-engine/internal/db"
	"github.com/yourname/gatelink-engine/internal/health"
	"github.com/yourname/gatelink-engine/internal/proxy"
	"github.com/yourname/gatelink-engine/internal/scheduler"
	"github.com/yourname/gatelink-engine/pkg/adapters"
)

// Router 持有所有依赖，用于注册路由
type Router struct {
	db         *db.Pool
	pool       *scheduler.Pool
	engine     *scheduler.Engine
	forwarder  *proxy.Forwarder
	registry   *vendor.Registry
	classifier *audit.Classifier
	scorer     *health.Scorer
	ks         *crypto.Keystore
}

// New 创建路由器
func New(
	db *db.Pool,
	pool *scheduler.Pool,
	engine *scheduler.Engine,
	forwarder *proxy.Forwarder,
	registry *vendor.Registry,
	classifier *audit.Classifier,
	scorer *health.Scorer,
	ks *crypto.Keystore,
) *Router {
	return &Router{
		db:         db,
		pool:       pool,
		engine:     engine,
		forwarder:  forwarder,
		registry:   registry,
		classifier: classifier,
		scorer:     scorer,
		ks:         ks,
	}
}

// Register 注册所有路由到 gin engine
func (r *Router) Register(engine *gin.Engine) {
	engine.Use(Logger(), Recovery())

	engine.GET("/health", r.handleHealth)

	accountsH := NewAccountsHandler(r.db, r.pool, r.scorer, r.registry, r.ks)
	eventsH := NewEventsHandler(r.db)

	internal := engine.Group("/internal/v1", InternalOnly(), r.injectRegistry())
	{
		// 调度 + 审核
		internal.POST("/dispatch", NewDispatchHandler(r.engine, r.forwarder).Handle)
		internal.POST("/audit", NewAuditHandler(r.classifier).Handle)

		// 账号池状态
		internal.GET("/pool/status", r.handlePoolStatus)

		// 账号管理
		internal.POST("/accounts", accountsH.HandleCreate)
		internal.GET("/accounts/:id/health", accountsH.HandleHealth)
		internal.POST("/accounts/:id/verify", accountsH.HandleVerify)
		internal.GET("/accounts/:id/console-usage", accountsH.HandleConsoleUsage)
		internal.GET("/accounts/:id/diff", accountsH.HandleDiff)
		internal.GET("/accounts/:id/events", eventsH.HandleAccountEvents)
	}
}

func (r *Router) injectRegistry() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("registry", r.registry)
		c.Next()
	}
}

func (r *Router) handleHealth(c *gin.Context) {
	dbStatus := "ok"
	if err := r.db.Healthy(c.Request.Context()); err != nil {
		dbStatus = "error"
	}
	OK(c, gin.H{
		"service":  "engine",
		"version":  "0.1.0",
		"database": dbStatus,
		"db_stats": r.db.Stats(),
	})
}

func (r *Router) handlePoolStatus(c *gin.Context) {
	stats, err := r.pool.Stats(c.Request.Context())
	if err != nil {
		InternalError(c)
		return
	}
	OK(c, gin.H{
		"pool_counts": stats,
	})
}
