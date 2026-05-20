package etcd_svc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/opskat/opskat/internal/connpool"
	"github.com/opskat/opskat/internal/service/asset_svc"
	"github.com/opskat/opskat/internal/service/credential_resolver"
	"github.com/opskat/opskat/internal/sshpool"

	"github.com/cago-frame/cago/pkg/logger"
)

const defaultEtcdPageSize = int64(100)

type etcdExecutor interface {
	Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error)
	Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error)
	Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error)
}

type Service struct {
	sshPool *sshpool.Pool
	history *CommandHistory
}

func New(sshPool *sshpool.Pool) *Service {
	return &Service{
		sshPool: sshPool,
		history: NewCommandHistory(200),
	}
}

func (s *Service) RangeKeys(ctx context.Context, req EtcdRangeRequest) (EtcdRangeResponse, error) {
	var out EtcdRangeResponse
	err := s.withClient(ctx, req.AssetID, func(ctx context.Context, exec etcdExecutor, client *clientv3.Client) error {
		var err error
		out, err = rangeKeys(ctx, exec, req)
		return err
	})
	return out, err
}

func (s *Service) GetKey(ctx context.Context, req EtcdKeyRequest) (EtcdKeyDetail, error) {
	var out EtcdKeyDetail
	err := s.withClient(ctx, req.AssetID, func(ctx context.Context, exec etcdExecutor, client *clientv3.Client) error {
		var err error
		out, err = getKey(ctx, exec, req)
		return err
	})
	return out, err
}

func (s *Service) PutKey(ctx context.Context, req EtcdPutRequest) error {
	return s.withClient(ctx, req.AssetID, func(ctx context.Context, exec etcdExecutor, client *clientv3.Client) error {
		return putKey(ctx, exec, req)
	})
}

func (s *Service) DeleteKeys(ctx context.Context, req EtcdDeleteRequest) error {
	return s.withClient(ctx, req.AssetID, func(ctx context.Context, exec etcdExecutor, client *clientv3.Client) error {
		return deleteKeys(ctx, exec, req)
	})
}

func (s *Service) GetStatus(ctx context.Context, assetID int64) (EtcdStatus, error) {
	var out EtcdStatus
	err := s.withClient(ctx, assetID, func(ctx context.Context, exec etcdExecutor, client *clientv3.Client) error {
		var err error
		out, err = getStatus(ctx, client, assetID)
		return err
	})
	return out, err
}

func (s *Service) GetMembers(ctx context.Context, assetID int64) ([]EtcdMember, error) {
	var out []EtcdMember
	err := s.withClient(ctx, assetID, func(ctx context.Context, exec etcdExecutor, client *clientv3.Client) error {
		var err error
		out, err = getMembers(ctx, client)
		return err
	})
	return out, err
}

func (s *Service) CommandHistory(assetID int64, limit int) []EtcdCommandHistoryEntry {
	return s.history.List(assetID, limit)
}

func (s *Service) withClient(ctx context.Context, assetID int64, fn func(context.Context, etcdExecutor, *clientv3.Client) error) error {
	asset, err := asset_svc.Asset().Get(ctx, assetID)
	if err != nil {
		return fmt.Errorf("资产不存在: %w", err)
	}
	if !asset.IsEtcd() {
		return fmt.Errorf("资产不是 etcd 类型")
	}
	cfg, err := asset.GetEtcdConfig()
	if err != nil {
		return fmt.Errorf("获取 etcd 配置失败: %w", err)
	}
	password, err := credential_resolver.Default().ResolveEtcdPassword(ctx, cfg)
	if err != nil {
		return fmt.Errorf("解析 etcd 凭据失败: %w", err)
	}

	timeout := 30 * time.Second
	if cfg.CommandTimeoutSeconds > 0 {
		timeout = time.Duration(cfg.CommandTimeoutSeconds) * time.Second
	}
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, closer, err := connpool.DialEtcd(opCtx, asset, cfg, password, s.sshPool)
	if err != nil {
		return fmt.Errorf("连接 etcd 失败: %w", err)
	}
	defer closeEtcdClient(client, closer)

	return fn(opCtx, &goEtcdExecutor{client: client, history: s.history, assetID: assetID}, client)
}

func closeEtcdClient(client *clientv3.Client, closer io.Closer) {
	if client != nil {
		if err := client.Close(); err != nil {
			logger.Default().Warn("close etcd client failed", zap.Error(err))
		}
	}
	if closer != nil {
		if err := closer.Close(); err != nil {
			logger.Default().Warn("close etcd tunnel failed", zap.Error(err))
		}
	}
}

type goEtcdExecutor struct {
	client  *clientv3.Client
	history *CommandHistory
	assetID int64
}

func (e *goEtcdExecutor) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	start := time.Now()
	resp, err := e.client.Get(ctx, key, opts...)
	e.record("GET "+key, time.Since(start), err)
	return resp, err
}

func (e *goEtcdExecutor) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	start := time.Now()
	resp, err := e.client.Put(ctx, key, val, opts...)
	e.record("PUT "+key, time.Since(start), err)
	return resp, err
}

func (e *goEtcdExecutor) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	start := time.Now()
	resp, err := e.client.Delete(ctx, key, opts...)
	cmd := "DELETE " + key
	if len(opts) > 0 {
		cmd = "DELETE_RANGE " + key
	}
	e.record(cmd, time.Since(start), err)
	return resp, err
}

func (e *goEtcdExecutor) record(command string, d time.Duration, err error) {
	if e.history != nil {
		e.history.Add(EtcdCommandHistoryEntry{
			AssetID:    e.assetID,
			Command:    command,
			CostMillis: d.Milliseconds(),
			Error:      errorString(err),
			Timestamp:  time.Now().UnixMilli(),
		})
	}
}

func errorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// CommandHistory is a thread-safe in-memory ring buffer for command history.
type CommandHistory struct {
	mu      sync.RWMutex
	entries []EtcdCommandHistoryEntry
	limit   int
}

func NewCommandHistory(limit int) *CommandHistory {
	return &CommandHistory{limit: limit}
}

func (h *CommandHistory) Add(entry EtcdCommandHistoryEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.entries = append(h.entries, entry)
	if len(h.entries) > h.limit {
		h.entries = h.entries[len(h.entries)-h.limit:]
	}
}

func (h *CommandHistory) List(assetID int64, limit int) []EtcdCommandHistoryEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var out []EtcdCommandHistoryEntry
	for i := len(h.entries) - 1; i >= 0 && len(out) < limit; i-- {
		if h.entries[i].AssetID == assetID {
			out = append(out, h.entries[i])
		}
	}
	return out
}
