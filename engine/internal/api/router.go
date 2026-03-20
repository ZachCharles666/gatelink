package api

import (
	"github.com/gin-gonic/gin"
	"github.com/yourname/tokenglide-engine/internal/db"
)

// Router 持有所有依赖，用于注册路由
type Router struct {
	db *db.Pool
}

// New 创建路由器
func New(db *db.Pool) *Router {
	return &Router{db: db}
}

// Register 注册所有路由到 gin engine
func (r *Router) Register(engine *gin.Engine) {
	engine.Use(Logger(), Recovery())

	engine.GET("/health", r.handleHealth)

	internal := engine.Group("/internal/v1", InternalOnly())
	{
		// Week 3+ 实现
		_ = internal
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
