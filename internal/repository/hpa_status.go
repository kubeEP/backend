package repository

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"gorm.io/gorm"
)

type HPAStatus interface {
	GetAllHPAStatusByScheduledHPAConfigID(
		tx *gorm.DB,
		scheduledHPAConfigID uuid.UUID,
	) ([]*model.HPAStatus, error)
}

type hpaStatus struct {
}

func newHpaStatus() HPAStatus {
	return &hpaStatus{}
}

func (h *hpaStatus) GetAllHPAStatusByScheduledHPAConfigID(
	tx *gorm.DB,
	scheduledHPAConfigID uuid.UUID,
) ([]*model.HPAStatus, error) {
	var data []*model.HPAStatus
	err := tx.Model(&model.HPAStatus{}).Where(
		"scheduled_hpa_config_id = ?",
		scheduledHPAConfigID,
	).Find(&data).Error
	return data, err
}
