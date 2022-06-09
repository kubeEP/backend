package model

import (
	gormDatatype "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/gorm/datatype"
	"gorm.io/gorm"
	"time"
)

type NodePoolStatus struct {
	CreatedAt         time.Time         `gorm:"primaryKey;default:now()"`
	UpdatedNodePoolID gormDatatype.UUID `gorm:"primaryKey"`
	NodeCount         int32
	UpdatedNodePool   UpdatedNodePool `gorm:"ForeignKey:UpdatedNodePoolID;constraint:OnDelete:CASCADE"`
}

func (NodePoolStatus) TableName() string {
	return "node_pool_status"
}

func (n *NodePoolStatus) AdditionalMigration(db *gorm.DB) error {
	tableName := n.TableName()
	var exist bool
	row := db.Raw(
		"select exists(select * from timescaledb_information.hypertables where hypertable_name = ?)",
		tableName,
	).Row()
	if err := row.Err(); err != nil {
		return err
	}
	if err := row.Scan(&exist); err != nil {
		return err
	}
	if !exist {
		return db.Exec(
			`select create_hypertable(?,'created_at', 'updated_node_pool_id', 4)`,
			tableName,
		).Error
	}
	return nil
}
