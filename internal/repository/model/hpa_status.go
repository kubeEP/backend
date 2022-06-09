package model

import (
	gormDatatype "github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/gorm/datatype"
	"gorm.io/gorm"
	"time"
)

type HPAStatus struct {
	CreatedAt            time.Time         `gorm:"primaryKey;default:now()"`
	ScheduledHPAConfigID gormDatatype.UUID `gorm:"primaryKey"`
	Replicas             int32
	AvailableReplicas    int32
	ReadyReplicas        int32
	UnavailableReplicas  int32
	ScheduledHPAConfig   ScheduledHPAConfig `gorm:"ForeignKey:ScheduledHPAConfigID;constraint:OnDelete:CASCADE"`
}

func (HPAStatus) TableName() string {
	return "hpa_status"
}

func (h *HPAStatus) AdditionalMigration(db *gorm.DB) error {
	tableName := h.TableName()
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
			`select create_hypertable(?,'created_at', 'scheduled_hpa_config_id', '4')`,
			tableName,
		).Error
	}
	return nil
}
