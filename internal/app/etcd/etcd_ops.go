package etcd

import (
	"github.com/opskat/opskat/internal/app/i18n"
	"github.com/opskat/opskat/internal/service/etcd_svc"
)

func (e *Etcd) EtcdRangeKeys(req etcd_svc.EtcdRangeRequest) (etcd_svc.EtcdRangeResponse, error) {
	return e.service.RangeKeys(i18n.Ctx(e.ctx, e.lang.Lang()), req)
}

func (e *Etcd) EtcdGetKey(req etcd_svc.EtcdKeyRequest) (etcd_svc.EtcdKeyDetail, error) {
	return e.service.GetKey(i18n.Ctx(e.ctx, e.lang.Lang()), req)
}

func (e *Etcd) EtcdPutKey(req etcd_svc.EtcdPutRequest) error {
	return e.service.PutKey(i18n.Ctx(e.ctx, e.lang.Lang()), req)
}

func (e *Etcd) EtcdDeleteKeys(req etcd_svc.EtcdDeleteRequest) error {
	return e.service.DeleteKeys(i18n.Ctx(e.ctx, e.lang.Lang()), req)
}

func (e *Etcd) EtcdGetStatus(assetID int64) (etcd_svc.EtcdStatus, error) {
	return e.service.GetStatus(i18n.Ctx(e.ctx, e.lang.Lang()), assetID)
}

func (e *Etcd) EtcdGetMembers(assetID int64) ([]etcd_svc.EtcdMember, error) {
	return e.service.GetMembers(i18n.Ctx(e.ctx, e.lang.Lang()), assetID)
}

func (e *Etcd) EtcdCommandHistory(assetID int64, limit int) []etcd_svc.EtcdCommandHistoryEntry {
	return e.service.CommandHistory(assetID, limit)
}
