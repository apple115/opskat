package etcd_svc

import (
	"context"
	"fmt"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// WatchEventType 表示 watch 事件类型。
type WatchEventType string

const (
	WatchEventPut    WatchEventType = "put"
	WatchEventDelete WatchEventType = "delete"
	WatchEventError  WatchEventType = "error"
)

// WatchEvent 表示一个 etcd watch 事件。
type WatchEvent struct {
	Type       WatchEventType `json:"type"`
	Key        string         `json:"key"`
	Value      string         `json:"value,omitempty"`
	PrevValue  string         `json:"prevValue,omitempty"`
	Revision   int64          `json:"revision"`
	Timestamp  int64          `json:"timestamp"`
}

// WatchSession 管理单个 watch 会话。
type WatchSession struct {
	id      string
	cancel  context.CancelFunc
	done    chan struct{}
	onEvent func(event WatchEvent)
}

// WatchManager 管理所有活跃的 etcd watch 会话。
type WatchManager struct {
	sessions sync.Map // string -> *WatchSession
}

// NewWatchManager 创建 watch 管理器。
func NewWatchManager() *WatchManager {
	return &WatchManager{}
}

// Start 启动对指定 etcd 资产的 watch。
func (m *WatchManager) Start(ctx context.Context, assetID int64, prefix string, client *clientv3.Client, onEvent func(event WatchEvent)) (string, error) {
	if client == nil {
		return "", fmt.Errorf("etcd client is nil")
	}

	watchID := fmt.Sprintf("%d:%s:%d", assetID, prefix, time.Now().UnixNano())
	watchCtx, cancel := context.WithCancel(ctx)

	session := &WatchSession{
		id:      watchID,
		cancel:  cancel,
		done:    make(chan struct{}),
		onEvent: onEvent,
	}
	m.sessions.Store(watchID, session)

	go m.runWatch(watchCtx, session, client, prefix)
	return watchID, nil
}

// Stop 停止指定 watch 会话。
func (m *WatchManager) Stop(watchID string) {
	if val, ok := m.sessions.LoadAndDelete(watchID); ok {
		session := val.(*WatchSession)
		session.cancel()
		<-session.done
	}
}

// StopAll 停止所有 watch 会话。
func (m *WatchManager) StopAll() {
	m.sessions.Range(func(key, val any) bool {
		m.Stop(key.(string))
		return true
	})
}

func (m *WatchManager) runWatch(ctx context.Context, session *WatchSession, client *clientv3.Client, prefix string) {
	defer close(session.done)

	var opts []clientv3.OpOption
	if prefix != "" {
		opts = append(opts, clientv3.WithPrefix())
	}
	// 获取 prevValue
	opts = append(opts, clientv3.WithPrevKV())

	watchCh := client.Watch(ctx, prefix, opts...)
	for wresp := range watchCh {
		if wresp.Err() != nil {
			// 发送错误事件后退出
			session.onEvent(WatchEvent{
				Type:      "error",
				Key:       prefix,
				Value:     wresp.Err().Error(),
				Timestamp: time.Now().UnixMilli(),
			})
			return
		}
		for _, ev := range wresp.Events {
			var evtType WatchEventType
			switch ev.Type {
			case clientv3.EventTypePut:
				evtType = WatchEventPut
			case clientv3.EventTypeDelete:
				evtType = WatchEventDelete
			}

			event := WatchEvent{
				Type:      evtType,
				Key:       string(ev.Kv.Key),
				Value:     string(ev.Kv.Value),
				Revision:  ev.Kv.ModRevision,
				Timestamp: time.Now().UnixMilli(),
			}
			if ev.PrevKv != nil {
				event.PrevValue = string(ev.PrevKv.Value)
			}

			if session.onEvent != nil {
				session.onEvent(event)
			}
		}
	}
}
