// Package query 实现 query binder：SQL/Mongo/Redis 执行 + 三种面板连接缓存 + 表导出。
package query

import (
	"context"
	"database/sql"
	"time"

	"github.com/opskat/opskat/internal/connpool"
	"github.com/opskat/opskat/internal/sshpool"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	panelConnIdleTTL       = 5 * time.Minute
	panelConnEvictInterval = 30 * time.Second
)

// LangProvider 由 system binder 实现。
type LangProvider interface {
	Lang() string
}

// Query binder：DB/Mongo/Redis/etcd 查询执行 + 面板连接缓存。
type Query struct {
	appCtx context.Context
	ctx    context.Context
	lang   LangProvider

	pool *sshpool.Pool

	dbPanelCache    *panelConnCache[*sql.DB]
	redisPanelCache *panelConnCache[*redis.Client]
	mongoPanelCache *panelConnCache[*connpool.MongoClientCloser]
	etcdPanelCache  *panelConnCache[*clientv3.Client]

	evictCtx context.Context
	evictCxl context.CancelFunc
}

// New 构造 query binder。
func New(appCtx context.Context, lang LangProvider, pool *sshpool.Pool) *Query {
	return &Query{appCtx: appCtx, lang: lang, pool: pool}
}

// Startup 初始化四个面板连接缓存 + 各自的 evictor 协程。
func (q *Query) Startup(ctx context.Context) {
	q.ctx = ctx
	q.dbPanelCache = newPanelConnCache[*sql.DB]("database", panelConnIdleTTL)
	q.redisPanelCache = newPanelConnCache[*redis.Client]("redis", panelConnIdleTTL)
	q.mongoPanelCache = newPanelConnCache[*connpool.MongoClientCloser]("mongodb", panelConnIdleTTL)
	q.etcdPanelCache = newPanelConnCache[*clientv3.Client]("etcd", panelConnIdleTTL)
	q.evictCtx, q.evictCxl = context.WithCancel(ctx)
	go q.dbPanelCache.startEvictor(q.evictCtx, panelConnEvictInterval)
	go q.redisPanelCache.startEvictor(q.evictCtx, panelConnEvictInterval)
	go q.mongoPanelCache.startEvictor(q.evictCtx, panelConnEvictInterval)
	go q.etcdPanelCache.startEvictor(q.evictCtx, panelConnEvictInterval)
}

// Cleanup 关闭 evictor 并释放所有缓存连接。
func (q *Query) Cleanup() {
	if q.evictCxl != nil {
		q.evictCxl()
	}
	if q.dbPanelCache != nil {
		_ = q.dbPanelCache.Close()
	}
	if q.redisPanelCache != nil {
		_ = q.redisPanelCache.Close()
	}
	if q.mongoPanelCache != nil {
		_ = q.mongoPanelCache.Close()
	}
	if q.etcdPanelCache != nil {
		_ = q.etcdPanelCache.Close()
	}
}
