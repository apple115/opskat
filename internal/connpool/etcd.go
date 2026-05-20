package connpool

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/opskat/opskat/internal/model/entity/asset_entity"
	"github.com/opskat/opskat/internal/sshpool"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// DialEtcd 创建 etcd 连接（直连或通过 SSH 隧道）
// password 为已解析的明文密码，由调用方负责解密
func DialEtcd(ctx context.Context, asset *asset_entity.Asset, cfg *asset_entity.EtcdConfig, password string, sshPool *sshpool.Pool) (*clientv3.Client, io.Closer, error) {
	endpoints := []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)}

	var tlsConfig *tls.Config
	if cfg.TLS {
		var err error
		tlsConfig, err = buildEtcdTLSConfig(cfg)
		if err != nil {
			return nil, nil, err
		}
	}

	dialTimeout := 5 * time.Second
	if cfg.CommandTimeoutSeconds > 0 {
		dialTimeout = time.Duration(cfg.CommandTimeoutSeconds) * time.Second
	}

	config := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
		TLS:         tlsConfig,
		Username:    cfg.Username,
		Password:    password,
	}

	var tunnel *SSHTunnel
	tunnelID := asset.SSHTunnelID
	if tunnelID == 0 {
		tunnelID = cfg.SSHAssetID // backward compat
	}
	if tunnelID > 0 && sshPool != nil {
		tunnel = NewSSHTunnel(tunnelID, cfg.Host, cfg.Port, sshPool)
		config.DialOptions = []grpc.DialOption{
			grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
				return tunnel.Dial(ctx)
			}),
		}
	}

	client, err := clientv3.New(config)
	if err != nil {
		if tunnel != nil {
			_ = tunnel.Close()
		}
		return nil, nil, fmt.Errorf("etcd 连接失败: %w", err)
	}

	// 验证连接
	ctxTimeout, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	if _, err := client.Status(ctxTimeout, endpoints[0]); err != nil {
		_ = client.Close()
		if tunnel != nil {
			_ = tunnel.Close()
		}
		return nil, nil, fmt.Errorf("etcd 连接验证失败: %w", err)
	}

	if tunnel == nil {
		return client, nil, nil
	}
	return client, tunnel, nil
}

func buildEtcdTLSConfig(cfg *asset_entity.EtcdConfig) (*tls.Config, error) {
	return BuildTLSConfig("Etcd", TLSFields{
		ServerName: cfg.TLSServerName,
		Insecure:   cfg.TLSInsecure,
		CAFile:     cfg.TLSCAFile,
		CertFile:   cfg.TLSCertFile,
		KeyFile:    cfg.TLSKeyFile,
	})
}

// EtcdClientCloser wraps *clientv3.Client to satisfy io.Closer for panel caches.
type EtcdClientCloser struct {
	*clientv3.Client
}

func (e *EtcdClientCloser) Close() error {
	if e.Client != nil {
		return e.Client.Close()
	}
	return nil
}

// NewEtcdClientCloser creates a closer wrapping an etcd client.
func NewEtcdClientCloser(client *clientv3.Client) *EtcdClientCloser {
	return &EtcdClientCloser{Client: client}
}
