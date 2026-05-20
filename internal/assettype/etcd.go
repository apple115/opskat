package assettype

import (
	"context"
	"fmt"

	"github.com/opskat/opskat/internal/model/entity/asset_entity"
	"github.com/opskat/opskat/internal/model/entity/policy"
	"github.com/opskat/opskat/internal/service/credential_resolver"
	"github.com/opskat/opskat/internal/service/credential_svc"
)

type etcdHandler struct{}

func init() {
	Register(&etcdHandler{})
	policy.RegisterDefaultPolicy("etcd", func() any { return asset_entity.DefaultEtcdPolicy() })
}

func (h *etcdHandler) Type() string     { return asset_entity.AssetTypeEtcd }
func (h *etcdHandler) DefaultPort() int { return 2379 }

func (h *etcdHandler) SafeView(a *asset_entity.Asset) map[string]any {
	cfg, err := a.GetEtcdConfig()
	if err != nil || cfg == nil {
		return nil
	}
	return map[string]any{
		"host": cfg.Host, "port": cfg.Port,
		"username": cfg.Username,
	}
}

func (h *etcdHandler) ResolvePassword(ctx context.Context, a *asset_entity.Asset) (string, error) {
	cfg, err := a.GetEtcdConfig()
	if err != nil {
		return "", fmt.Errorf("get etcd config failed: %w", err)
	}
	return credential_resolver.Default().ResolveEtcdPassword(ctx, cfg)
}

func (h *etcdHandler) ValidateCreateArgs(args map[string]any) error {
	return validateRemoteServerArgs(args)
}

func (h *etcdHandler) DefaultPolicy() any { return asset_entity.DefaultEtcdPolicy() }

func (h *etcdHandler) ApplyCreateArgs(_ context.Context, a *asset_entity.Asset, args map[string]any) error {
	cfg := &asset_entity.EtcdConfig{
		Host:       ArgString(args, "host"),
		Port:       ArgInt(args, "port"),
		Username:   ArgString(args, "username"),
		SSHAssetID: ArgInt64(args, "ssh_asset_id"),
	}
	if password := ArgString(args, "password"); password != "" {
		encrypted, err := credential_svc.Default().Encrypt(password)
		if err != nil {
			return fmt.Errorf("encrypt etcd password: %w", err)
		}
		cfg.Password = encrypted
	}
	return a.SetEtcdConfig(cfg)
}

func (h *etcdHandler) ApplyUpdateArgs(_ context.Context, a *asset_entity.Asset, args map[string]any) error {
	cfg, err := a.GetEtcdConfig()
	if err != nil || cfg == nil {
		return err
	}
	if v := ArgString(args, "host"); v != "" {
		cfg.Host = v
	}
	if v := ArgInt(args, "port"); v > 0 {
		cfg.Port = v
	}
	if v := ArgString(args, "username"); v != "" {
		cfg.Username = v
	}
	if _, ok := args["ssh_asset_id"]; ok {
		cfg.SSHAssetID = ArgInt64(args, "ssh_asset_id")
	}
	if password := ArgString(args, "password"); password != "" {
		encrypted, err := credential_svc.Default().Encrypt(password)
		if err != nil {
			return fmt.Errorf("encrypt etcd password: %w", err)
		}
		cfg.Password = encrypted
		cfg.CredentialID = 0
	}
	return a.SetEtcdConfig(cfg)
}
