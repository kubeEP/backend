package model

import (
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/pkg/gorm/datatype"
)

type HPAUpdateStatus string

const (
	HPAUpdateFailed  HPAUpdateStatus = "FAILED"
	HPAUpdateSuccess HPAUpdateStatus = "SUCCESS"
	HPAUpdatePending HPAUpdateStatus = "PENDING"
)

type ScheduledHPAConfig struct {
	BaseModel
	Name      string
	MinPods   *int32
	MaxPods   int32
	Namespace string
	Status    HPAUpdateStatus `gorm:"default:PENDING"`
	Message   string
	EventID   gormDatatype.UUID
	Event     Event `gorm:"ForeignKey:EventID;constraint:OnDelete:CASCADE"`
}

func (s *ScheduledHPAConfig) TableName() string {
	return "scheduled_hpa_configs"
}
