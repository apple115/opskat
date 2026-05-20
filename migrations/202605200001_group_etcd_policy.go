package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// migration202605200001 为 groups 表添加 etcd_policy 字段
func migration202605200001() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202605200001",
		Migrate: func(tx *gorm.DB) error {
			return tx.Exec(`
				ALTER TABLE groups ADD COLUMN etcd_policy TEXT
			`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
