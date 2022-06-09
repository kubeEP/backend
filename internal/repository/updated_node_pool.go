package repository

import (
	"github.com/google/uuid"
	"github.com/hsjsjsj009/kubeEP/kubeEP-BE/internal/repository/model"
	"gorm.io/gorm"
)

type UpdatedNodePool interface {
	GetAllUpdatedNodePoolByEventID(
		tx *gorm.DB,
		eventID uuid.UUID,
	) ([]*model.UpdatedNodePool, error)
}

type updatedNodePool struct {
}

func newUpdatedNodePool() UpdatedNodePool {
	return &updatedNodePool{}
}

func (u *updatedNodePool) GetAllUpdatedNodePoolByEventID(
	tx *gorm.DB,
	eventID uuid.UUID,
) ([]*model.UpdatedNodePool, error) {
	var output []*model.UpdatedNodePool
	err := tx.Model(&model.UpdatedNodePool{}).Where("event_id = ?", eventID).Find(&output).Error
	return output, err
}
