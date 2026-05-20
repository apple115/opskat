package helper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/cago-frame/cago/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/opskat/opskat/internal/ai/aictx"
	"github.com/opskat/opskat/internal/ai/permission"
	"github.com/opskat/opskat/internal/connpool"
	"github.com/opskat/opskat/internal/model/entity/asset_entity"
	"github.com/opskat/opskat/internal/service/asset_svc"
	"github.com/opskat/opskat/internal/service/credential_resolver"
	"github.com/opskat/opskat/internal/service/etcd_svc"
)

// --- etcd 连接缓存 ---

type etcdCacheKeyType struct{}

// EtcdClientCache 在同一次 AI Send 中复用 etcd 连接
type EtcdClientCache = ConnCache[*clientv3.Client]

// NewEtcdClientCache 创建 etcd 连接缓存
func NewEtcdClientCache() *EtcdClientCache {
	return NewConnCache[*clientv3.Client]("Etcd")
}

// WithEtcdCache 将 etcd 缓存注入 context
func WithEtcdCache(ctx context.Context, cache *EtcdClientCache) context.Context {
	return context.WithValue(ctx, etcdCacheKeyType{}, cache)
}

func getEtcdCache(ctx context.Context) *EtcdClientCache {
	if cache, ok := ctx.Value(etcdCacheKeyType{}).(*EtcdClientCache); ok {
		return cache
	}
	return nil
}

// --- Handler ---

func HandleExecEtcd(ctx context.Context, args map[string]any) (string, error) {
	assetID := aictx.ArgInt64(args, "asset_id")
	command := aictx.ArgString(args, "command")
	if assetID == 0 || command == "" {
		return "", fmt.Errorf("missing required parameters: asset_id, command")
	}

	// 权限检查
	if checker := permission.GetPolicyChecker(ctx); checker != nil {
		result := checker.CheckForAsset(ctx, assetID, asset_entity.AssetTypeEtcd, command)
		aictx.RecordDecision(ctx, result)
		if result.Decision != aictx.Allow {
			return result.Message, nil
		}
	}

	asset, err := asset_svc.Asset().Get(ctx, assetID)
	if err != nil {
		return "", fmt.Errorf("asset not found: %w", err)
	}
	if !asset.IsEtcd() {
		return "", fmt.Errorf("asset is not etcd type")
	}
	cfg, err := asset.GetEtcdConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get etcd config: %w", err)
	}

	client, closer, err := getOrDialEtcd(ctx, asset, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to connect to etcd: %w", err)
	}
	if getEtcdCache(ctx) == nil {
		if client != nil {
			defer func() {
				if err := client.Close(); err != nil {
					logger.Default().Warn("close etcd connection", zap.Error(err))
				}
			}()
		}
		if closer != nil {
			defer func() {
				if err := closer.Close(); err != nil {
					logger.Default().Warn("close etcd tunnel", zap.Error(err))
				}
			}()
		}
	}

	return etcd_svc.ExecuteEtcd(ctx, client, command)
}

func getOrDialEtcd(ctx context.Context, asset *asset_entity.Asset, cfg *asset_entity.EtcdConfig) (*clientv3.Client, io.Closer, error) {
	dialFn := func() (*clientv3.Client, io.Closer, error) {
		password, err := credential_resolver.Default().ResolveEtcdPassword(ctx, cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve credentials: %w", err)
		}
		return connpool.DialEtcd(ctx, asset, cfg, password, getSSHPool(ctx))
	}
	if cache := getEtcdCache(ctx); cache != nil {
		return cache.GetOrDial(asset.ID, dialFn)
	}
	return dialFn()
}

// FormatEtcdResult formats an etcd operation result to JSON.
func FormatEtcdResult(result any) (string, error) {
	var out map[string]any
	switch v := result.(type) {
	case string:
		out = map[string]any{"type": "string", "value": v}
	case int64:
		out = map[string]any{"type": "integer", "value": v}
	case []any:
		out = map[string]any{"type": "list", "value": v}
	case map[string]any:
		out = map[string]any{"type": "map", "value": v}
	case nil:
		out = map[string]any{"type": "nil", "value": nil}
	default:
		out = map[string]any{"type": fmt.Sprintf("%T", v), "value": fmt.Sprint(v)}
	}
	data, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("failed to marshal etcd result: %w", err)
	}
	return string(data), nil
}
