package assettype

import (
	"context"
	"testing"

	"github.com/opskat/opskat/internal/model/entity/asset_entity"
	"github.com/smartystreets/goconvey/convey"
)

func TestEtcdHandler(t *testing.T) {
	convey.Convey("Etcd Handler", t, func() {
		h := &etcdHandler{}
		convey.Convey("Type and DefaultPort", func() {
			convey.So(h.Type(), convey.ShouldEqual, "etcd")
			convey.So(h.DefaultPort(), convey.ShouldEqual, 2379)
		})
		convey.Convey("SafeView", func() {
			a := &asset_entity.Asset{Type: "etcd", Status: 1}
			_ = a.SetEtcdConfig(&asset_entity.EtcdConfig{
				Host: "10.0.0.1", Port: 2379, Username: "root",
				Password: "secret",
			})
			view := h.SafeView(a)
			convey.So(view["host"], convey.ShouldEqual, "10.0.0.1")
			convey.So(view["port"], convey.ShouldEqual, 2379)
			convey.So(view["username"], convey.ShouldEqual, "root")
			_, hasPassword := view["password"]
			convey.So(hasPassword, convey.ShouldBeFalse)
		})
		convey.Convey("ApplyCreateArgs", func() {
			a := &asset_entity.Asset{Type: "etcd"}
			err := h.ApplyCreateArgs(context.Background(), a, map[string]any{
				"host": "10.0.0.1", "port": float64(2379),
				"username": "root", "ssh_asset_id": float64(7),
			})
			convey.So(err, convey.ShouldBeNil)
			cfg, _ := a.GetEtcdConfig()
			convey.So(cfg.Host, convey.ShouldEqual, "10.0.0.1")
			convey.So(cfg.Port, convey.ShouldEqual, 2379)
			convey.So(cfg.Username, convey.ShouldEqual, "root")
			convey.So(cfg.SSHAssetID, convey.ShouldEqual, 7)
		})
		convey.Convey("ApplyUpdateArgs", func() {
			a := &asset_entity.Asset{Type: "etcd"}
			_ = a.SetEtcdConfig(&asset_entity.EtcdConfig{
				Host: "10.0.0.1", Port: 2379, Username: "root",
			})
			err := h.ApplyUpdateArgs(context.Background(), a, map[string]any{"host": "10.0.0.2"})
			convey.So(err, convey.ShouldBeNil)
			cfg, _ := a.GetEtcdConfig()
			convey.So(cfg.Host, convey.ShouldEqual, "10.0.0.2")
			convey.So(cfg.Port, convey.ShouldEqual, 2379)
			convey.So(cfg.Username, convey.ShouldEqual, "root")
		})
		convey.Convey("Registered", func() {
			h, ok := Get("etcd")
			convey.So(ok, convey.ShouldBeTrue)
			convey.So(h.Type(), convey.ShouldEqual, "etcd")
		})
	})
}
