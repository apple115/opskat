// Package etcd 实现 etcd binder。
package etcd

import (
	"context"
	"fmt"
	"io"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/opskat/opskat/internal/connpool"
	"github.com/opskat/opskat/internal/service/asset_svc"
	"github.com/opskat/opskat/internal/service/credential_resolver"
	"github.com/opskat/opskat/internal/service/etcd_svc"
)

// watchEntry 持有单个 watch 会话的上下文和 etcd 客户端资源。
type watchEntry struct {
	cancel  context.CancelFunc
	client  *clientv3.Client
	closer  io.Closer
}

// activeWatches 管理由 Etcd binder 启动的活跃 watch 会话。
var activeWatches sync.Map // string(watchID) -> *watchEntry

// EtcdStartWatch 启动对指定 etcd 资产的 watch 监控。
// prefix 为空字符串时监听所有 key。
func (e *Etcd) EtcdStartWatch(assetID int64, prefix string) (string, error) {
	ctx := e.ctx

	asset, err := asset_svc.Asset().Get(ctx, assetID)
	if err != nil {
		return "", fmt.Errorf("资产不存在: %w", err)
	}
	if !asset.IsEtcd() {
		return "", fmt.Errorf("资产不是 etcd 类型")
	}
	cfg, err := asset.GetEtcdConfig()
	if err != nil {
		return "", fmt.Errorf("获取 etcd 配置失败: %w", err)
	}
	password, err := credential_resolver.Default().ResolveEtcdPassword(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("解析 etcd 凭据失败: %w", err)
	}

	watchCtx, cancel := context.WithCancel(ctx)
	client, closer, err := connpool.DialEtcd(watchCtx, asset, cfg, password, e.pool)
	if err != nil {
		cancel()
		return "", fmt.Errorf("连接 etcd 失败: %w", err)
	}

	onEvent := func(event etcd_svc.WatchEvent) {
		wailsRuntime.EventsEmit(e.ctx, fmt.Sprintf("etcd:watch:%d", assetID), event)
	}

	watchID, err := e.service.WatchManager().Start(watchCtx, assetID, prefix, client, onEvent)
	if err != nil {
		cancel()
		if closer != nil {
			_ = closer.Close()
		}
		_ = client.Close()
		return "", err
	}

	activeWatches.Store(watchID, &watchEntry{cancel: cancel, client: client, closer: closer})
	return watchID, nil
}

// EtcdStopWatch 停止指定 watch 会话并释放连接资源。
func (e *Etcd) EtcdStopWatch(watchID string) {
	if val, ok := activeWatches.Load(watchID); ok {
		entry := val.(*watchEntry)
		entry.cancel()
		e.service.WatchManager().Stop(watchID)
		_ = entry.client.Close()
		if entry.closer != nil {
			_ = entry.closer.Close()
		}
		activeWatches.Delete(watchID)
	}
}
